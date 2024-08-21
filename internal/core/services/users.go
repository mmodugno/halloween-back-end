package services

import (
	"context"
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type UserClient struct {
	client Client
}
type Client interface {
	InsertUser(u models.User) error
	GetAllUsers() ([]models.User, error)
}

func (c *UserClient) InsertUser(u models.User) error {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO users(is_admin, name, pw_code, costume, has_voted) VALUES (?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	pw := generatePass(u.Name)

	res, err := stmt.ExecContext(ctx, u.IsAdmin, u.Name, pw, u.Costume, false)
	if err != nil {
		log.Printf("Error %s when inserting row into products table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d products created ", rows)
	return nil
}
func (c *UserClient) LogIn(pass string) (*models.LoggedUser, error) {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, is_admin, has_voted FROM users WHERE pw_code = ?`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()

	var user models.LoggedUser
	row := stmt.QueryRowContext(ctx, pass)
	if err := row.Scan(&user.ID, &user.IsAdmin, &user.HasVoted); err != nil {
		log.Printf("Error scanning row: %s", err)
		return nil, err
	}

	log.Printf("User logged in: %+v", user)
	return &user, nil
}

func (c *UserClient) GetAllUsers() ([]models.User, error) {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := "SELECT id, is_admin, name, pw_code, has_voted, costume FROM users"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when querying users table", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.HasVoted, &u.Costume); err != nil {
			log.Printf("Error %s when scanning row", err)
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error %s during row iteration", err)
		return nil, err
	}

	return users, nil
}

func connectToDB() (*sql.DB, error) {
	db, err := repository.DBConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return nil, err
	}
	log.Printf("Successfully connected to database")

	err = repository.CreateUsersTable(db)
	if err != nil {
		log.Printf("Create users table failed with error %s", err)
		return nil, err
	}
	return db, nil
}

func generatePass(name string) string {
	id := ""
	for i := 0; i < 4; i++ {
		id += strconv.Itoa(rand.Intn(10))
	}
	return name + id
}

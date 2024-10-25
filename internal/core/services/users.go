package services

import (
	"context"
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type UserClient struct {
	client Client
}
type Client interface {
	InsertUser(u models.User) error
	GetAllUsers() ([]models.User, error)
	GetUserByPathphrase(pw string) (*models.User, error)
	Vote(user *models.User) error
}

func (c *UserClient) InsertUser(u models.User) error {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO users(is_admin, name, pw_code, costume, pending_votes) VALUES (?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	pw := generatePass(strings.ToLower(u.Name), false)

	res, err := stmt.ExecContext(ctx, u.IsAdmin, u.Name, pw, u.Costume, u.PendingVotes) //2 votes for each user
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

func (c *UserClient) InsertUsers(users []models.User, mock bool) error {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO users(is_admin, name, pw_code, costume, pending_votes) VALUES (?, ?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	for _, u := range users {
		pw := generatePass(strings.ToLower(u.Name), mock)

		_, err := stmt.ExecContext(ctx, u.IsAdmin, u.Name, pw, u.Costume, u.PendingVotes)
		if err != nil {
			log.Printf("Error %s when inserting row into users table", err)
			return err
		}
	}
	log.Printf("%d users created ", len(users))
	return nil
}

func (c *UserClient) LogIn(pass string) (*models.LoggedUser, error) {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, is_admin, pending_votes FROM users WHERE pw_code = ?`
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
	if err := row.Scan(&user.ID, &user.IsAdmin, &user.PendingVotes); err != nil {
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

	query := "SELECT id, is_admin, name, pw_code, pending_votes, costume FROM users ORDER BY name ASC;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when querying users table", err)
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.PendingVotes, &u.Costume); err != nil {
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

func (c *UserClient) GetUserByPathphrase(pw string) (*models.User, error) {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := "SELECT id, is_admin, name, pw_code, pending_votes, costume FROM users WHERE pw_code = ?;"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return nil, err
	}
	defer stmt.Close()

	var user models.User
	row := stmt.QueryRowContext(ctx, pw)
	if err := row.Scan(&user.ID, &user.IsAdmin, &user.Name, &user.PWCode, &user.PendingVotes, &user.Costume); err != nil {
		log.Printf("Error scanning row: %s", err)
		return nil, err
	}

	log.Printf("User logged in: %+v", user)
	return &user, nil

}

func (c *UserClient) Vote(user *models.User) error {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	// Update the HasVoted field in the database
	updateQuery := "UPDATE users SET pending_votes = ? WHERE id = ?"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, updateQuery)
	defer stmt.Close()

	_, err = db.Exec(updateQuery, max(user.PendingVotes-1, 0), user.ID)
	if err != nil {
		log.Printf("Error %s when updating pending votes field in DB", err)
		return err
	}

	return nil

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

func generatePass(name string, mock bool) string {
	if !mock {
		id := ""
		for i := 0; i < 2; i++ {
			id += strconv.Itoa(rand.Intn(10))
		}
		return name + id
	}
	return name + "11"
}

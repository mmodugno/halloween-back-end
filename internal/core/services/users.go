package services

import (
	"context"
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
	"time"
)

type UserClient struct {
	client Client
}
type Client interface {
	InsertUser(u models.User) error
}

func (c *UserClient) InsertUser(u models.User) error {
	db, err := connectToDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO users(is_admin, username, pw_code, pending_votes) VALUES (?, ?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, u.IsAdmin, u.Username, u.PWCode, u.PendingVotes)
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

func connectToDB() (*sql.DB, error) {
	db, err := repository.DBConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return nil, err
	}
	log.Printf("Successfully connected to database")

	err = repository.CreateUserTable(db)
	if err != nil {
		log.Printf("Create users table failed with error %s", err)
		return nil, err
	}
	return db, nil
}

package services

import (
	"context"
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
	"time"
)

type CostumeClient struct {
	client CClient
}
type CClient interface {
	InsertCostume(req models.Costume) error
}

func (c *CostumeClient) InsertCostume(req models.Costume) error {
	db, err := connectToCostumeDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO costumes(description, owner, votes) VALUES (?, ?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, req.Description, req.Owner, 0)
	if err != nil {
		log.Printf("Error %s when inserting row into costumes table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d costumes created ", rows)
	return nil
}

func connectToCostumeDB() (*sql.DB, error) {
	db, err := repository.DBConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return nil, err
	}
	log.Printf("Successfully connected to database")

	err = repository.CreateCustomeTable(db)
	if err != nil {
		log.Printf("Create costumes table failed with error %s", err)
		return nil, err
	}
	return db, nil
}

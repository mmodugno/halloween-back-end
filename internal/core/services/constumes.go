package services

import (
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
)

type CostumeClient struct {
	client CClient
}
type CClient interface {
	InsertCostume(req models.Costume) error
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

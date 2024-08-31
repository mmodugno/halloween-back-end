package services

import (
	"context"
	"database/sql"
	"halloween/internal/adapters/repository"
	"halloween/internal/models"
	"log"
	"time"
)

type VotesClient struct {
	client VClient
}
type VClient interface {
	InsertVote(u models.Vote) error
	GetWinner([]models.VoteResult, error)
}

func (c *VotesClient) InsertVote(u models.Vote) error {
	db, err := connectToVotesDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return err
	}
	defer db.Close()

	query := "INSERT INTO votes(voter_passphrase, user_costume_id, message) VALUES (?, ?, ?)"

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, u.VoterPassphrase, u.UserCostumeID, u.Message)
	if err != nil {
		log.Printf("Error %s when inserting row into votes table", err)
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

func (c *VotesClient) GetWinners() ([]models.VoteResult, error) {
	db, err := connectToVotesDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := `SELECT u.costume, u.name, COUNT(*) as vote_count 
				FROM votes v JOIN users u 
				ON v.user_costume_id = u.id 
				GROUP BY v.user_costume_id
				ORDER BY vote_count DESC
				LIMIT 3;`

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	results := make([]models.VoteResult, 0)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when querying tables", err)
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var v models.VoteResult
		if err := rows.Scan(&v.Costume, &v.Name, &v.VotesCount); err != nil {
			log.Printf("Error %s when scanning row", err)
			return nil, err
		}
		results = append(results, v)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error %s during row iteration", err)
		return results, err
	}

	return results, nil
}

func (c *VotesClient) GetResults() ([]models.VoteResult, error) {
	db, err := connectToVotesDB()
	if err != nil {
		log.Printf("Error %s when connecting to DB", err)
		return nil, err
	}
	defer db.Close()

	query := `SELECT u.costume, u.name, COUNT(*) as vote_count 
				FROM votes v JOIN users u 
				ON v.user_costume_id = u.id 
				GROUP BY v.user_costume_id
				ORDER BY vote_count DESC;`

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when querying tables", err)
		return []models.VoteResult{}, err
	}
	defer rows.Close()

	results := make([]models.VoteResult, 0)

	for rows.Next() {
		var v models.VoteResult
		if err := rows.Scan(&v.Costume, &v.Name, &v.VotesCount); err != nil {
			log.Printf("Error %s when scanning row", err)
			return nil, err
		}
		messages, err := c.getMessages(v.Name)
		if err != nil {
			log.Printf("Error %s when getting messages", err)
			return nil, err
		}
		v.Data = messages
		results = append(results, v)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error %s during row iteration", err)
		return nil, err
	}

	return results, nil
}

func (c *VotesClient) getMessages(costume string) ([]models.VoteData, error) {
	db, err := connectToVotesDB()
	query := `SELECT u1.name, v.message
		FROM 
			votes v
		JOIN 
			users u1 ON u1.pw_code = v.voter_passphrase 
		JOIN 
			users u2 ON u2.id = v.user_costume_id
		WHERE u2.name = ?`

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	rows, err := db.QueryContext(ctx, query, costume)
	if err != nil {
		log.Printf("Error %s when querying votes table", err)
		return nil, err
	}
	defer rows.Close()

	var votes []models.VoteData
	for rows.Next() {
		var u models.VoteData
		if err := rows.Scan(&u.User, &u.Message); err != nil {
			log.Printf("Error %s when scanning row", err)
			return nil, err
		}
		votes = append(votes, u)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error %s during row iteration", err)
		return nil, err
	}

	return votes, nil
}

func connectToVotesDB() (*sql.DB, error) {
	db, err := repository.DBConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return nil, err
	}
	log.Printf("Successfully connected to database")

	err = repository.CreateVotesTable(db)
	if err != nil {
		log.Printf("Create votes table failed with error %s", err)
		return nil, err
	}
	return db, nil
}

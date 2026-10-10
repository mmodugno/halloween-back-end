package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"halloween/internal/models"
)

var (
	ErrNotFound         = errors.New("user does not exist")
	ErrNoVotesLeft      = errors.New("user has already voted: no votes left")
	ErrSelfVote         = errors.New("you can't vote for yourself")
	ErrAlreadyVotedFor  = errors.New("user has already voted for this costume")
	ErrUnknownCandidate = errors.New("candidate does not exist")
	ErrVotingClosed     = errors.New("voting is closed")
	ErrVoteNotFound     = errors.New("vote does not exist")
	ErrUndoExpired      = errors.New("too late to undo this vote")
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// Create inserts all users in a single transaction and returns them with their
// new ID and generated pass code. With fixedCodes every code is name+"11"
// (used by the mock data loader).
func (s *UserService) Create(ctx context.Context, users []models.User, fixedCodes bool) ([]models.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	created := make([]models.User, 0, len(users))
	for _, u := range users {
		u.PWCode = generatePass(u.Name, fixedCodes)
		res, err := tx.ExecContext(ctx,
			"INSERT INTO users(is_admin, name, pw_code, costume, pending_votes) VALUES (?, ?, ?, ?, ?)",
			u.IsAdmin, u.Name, u.PWCode, u.Costume, u.PendingVotes)
		if err != nil {
			return nil, fmt.Errorf("inserting user %q: %w", u.Name, err)
		}
		u.ID, _ = res.LastInsertId()
		created = append(created, u)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// List returns every user ordered by name, without their pass codes.
func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, is_admin, name, pending_votes, costume FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PendingVotes, &u.Costume); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserService) GetByPassphrase(ctx context.Context, pw string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx,
		"SELECT id, is_admin, name, pw_code, pending_votes, costume FROM users WHERE pw_code = ?", pw).
		Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.PendingVotes, &u.Costume)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func generatePass(name string, fixed bool) string {
	suffix := "11"
	if !fixed {
		suffix = fmt.Sprintf("%02d", rand.Intn(100))
	}
	return strings.ToLower(name) + suffix
}

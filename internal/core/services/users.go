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
	ErrMissingName      = errors.New("name is required")
	ErrUserNotFound     = errors.New("user not found")
	ErrMessageNotFound  = errors.New("message not found")
	ErrForbidden        = errors.New("only admins can do this")
)

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// Create inserts all users in a single transaction and returns them with their
// new ID and generated pass code.
func (s *UserService) Create(ctx context.Context, users []models.User) ([]models.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	created := make([]models.User, 0, len(users))
	for _, u := range users {
		u.Name = strings.TrimSpace(u.Name)
		u.Costume = strings.TrimSpace(u.Costume)
		if u.Name == "" {
			return nil, ErrMissingName
		}
		if u.PWCode, err = uniquePass(ctx, tx, u.Name); err != nil {
			return nil, err
		}
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

// Update changes a user's name, costume, remaining votes and admin flag.
func (s *UserService) Update(ctx context.Context, u models.User) error {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return ErrMissingName
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE users SET name = ?, costume = ?, pending_votes = ?, is_admin = ? WHERE id = ?",
		u.Name, strings.TrimSpace(u.Costume), u.PendingVotes, u.IsAdmin, u.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Delete removes a user and every vote they cast or received. People who
// had voted for them get that vote back.
func (s *UserService) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var pw string
	err = tx.QueryRowContext(ctx, "SELECT pw_code FROM users WHERE id = ? FOR UPDATE", id).Scan(&pw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	steps := []struct {
		query string
		args  []any
	}{
		{`UPDATE users u JOIN votes v ON v.voter_passphrase = u.pw_code
			SET u.pending_votes = u.pending_votes + 1 WHERE v.user_costume_id = ?`, []any{id}},
		{"DELETE FROM votes WHERE user_costume_id = ? OR voter_passphrase = ?", []any{id, pw}},
		{"DELETE FROM users WHERE id = ?", []any{id}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Count returns how many users exist (0 means the app isn't set up yet).
func (s *UserService) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&n)
	return n, err
}

// ListWithCodes returns every user including their pass code (admin only).
func (s *UserService) ListWithCodes(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, is_admin, name, pw_code, pending_votes, costume FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.PendingVotes, &u.Costume); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
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

// uniquePass picks a pass code not used by anyone else: the name in
// lowercase without accents or spaces plus two digits (e.g. "maga42").
func uniquePass(ctx context.Context, tx *sql.Tx, name string) (string, error) {
	base := passBase(name)
	for attempt := 0; attempt < 50; attempt++ {
		// Two digits first; three if the name is very common.
		code := fmt.Sprintf("%s%02d", base, rand.Intn(100))
		if attempt >= 30 {
			code = fmt.Sprintf("%s%03d", base, rand.Intn(1000))
		}
		var taken int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE pw_code = ?", code).Scan(&taken); err != nil {
			return "", err
		}
		if taken == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not generate a unique pass code")
}

var accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

func passBase(name string) string {
	var b strings.Builder
	for _, r := range accents.Replace(strings.ToLower(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "user"
	}
	return b.String()
}

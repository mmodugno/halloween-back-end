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
		if u.Name == "" {
			return nil, ErrMissingName
		}
		// People with the same costume text end up in the same costume.
		if u.CostumeID, u.Costume, err = findOrCreateCostume(ctx, tx, u.Costume); err != nil {
			return nil, err
		}
		if err := insertUser(ctx, tx, &u); err != nil {
			return nil, fmt.Errorf("inserting user %q: %w", u.Name, err)
		}
		created = append(created, u)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// Update changes a user's name, remaining votes and admin flag, and moves
// them to another costume: the one with id CostumeID, or else the one named
// Costume (created if it doesn't exist; empty means no costume).
func (s *UserService) Update(ctx context.Context, u *models.User) error {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return ErrMissingName
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentID int64
	var currentName string
	var members int
	err = tx.QueryRowContext(ctx, `SELECT u.costume_id, COALESCE(c.name, ''),
			(SELECT COUNT(*) FROM users m WHERE m.costume_id = u.costume_id)
		FROM users u LEFT JOIN costumes c ON c.id = u.costume_id
		WHERE u.id = ? FOR UPDATE`, u.ID).Scan(&currentID, &currentName, &members)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	newName := strings.Join(strings.Fields(u.Costume), " ")
	switch {
	case u.CostumeID > 0 && u.CostumeID != currentID:
		err = tx.QueryRowContext(ctx, "SELECT name FROM costumes WHERE id = ?", u.CostumeID).Scan(&u.Costume)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCostumeNotFound
		}
	case currentID != 0 && strings.EqualFold(newName, currentName):
		u.CostumeID, u.Costume = currentID, currentName
	default:
		var taken int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM costumes WHERE LOWER(name) = LOWER(?)", newName).Scan(&taken)
		if err == nil && taken == 0 && currentID != 0 && members == 1 && newName != "" {
			// Alone in the costume: just rename it, keeping its votes.
			u.CostumeID, u.Costume = currentID, newName
			_, err = tx.ExecContext(ctx, "UPDATE costumes SET name = ? WHERE id = ?", newName, currentID)
		} else if err == nil {
			u.CostumeID, u.Costume, err = findOrCreateCostume(ctx, tx, newName)
		}
	}
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET name = ?, costume = ?, costume_id = ?, pending_votes = ?, is_admin = ? WHERE id = ?",
		u.Name, u.Costume, u.CostumeID, u.PendingVotes, u.IsAdmin, u.ID); err != nil {
		return err
	}
	if err := dropEmptyCostumes(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete removes a user and the votes they cast. If nobody else wears their
// costume, it goes too, and whoever voted for it gets the vote back.
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

	if _, err := tx.ExecContext(ctx, "DELETE FROM votes WHERE voter_passphrase = ?", pw); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id); err != nil {
		return err
	}
	if err := dropEmptyCostumes(ctx, tx); err != nil {
		return err
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
		"SELECT id, is_admin, name, pw_code, pending_votes, costume, costume_id FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.PendingVotes, &u.Costume, &u.CostumeID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// List returns every user ordered by name, without their pass codes.
func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, is_admin, name, pending_votes, costume, costume_id FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PendingVotes, &u.Costume, &u.CostumeID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserService) GetByPassphrase(ctx context.Context, pw string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRowContext(ctx,
		"SELECT id, is_admin, name, pw_code, pending_votes, costume, costume_id FROM users WHERE pw_code = ?", pw).
		Scan(&u.ID, &u.IsAdmin, &u.Name, &u.PWCode, &u.PendingVotes, &u.Costume, &u.CostumeID)
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

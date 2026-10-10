package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"halloween/internal/models"
)

type VoteService struct {
	db *sql.DB
}

func NewVoteService(db *sql.DB) *VoteService {
	return &VoteService{db: db}
}

// Cast validates and stores a vote, decrementing the voter's pending votes.
// Everything runs in one transaction with the voter row locked, so two votes
// sent at the same time can't both use the last remaining vote.
func (s *VoteService) Cast(ctx context.Context, passphrase string, candidateID int64, message string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var voterID, pending int64
	err = tx.QueryRowContext(ctx,
		"SELECT id, pending_votes FROM users WHERE pw_code = ? FOR UPDATE", passphrase).
		Scan(&voterID, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if err := checkVotingOpen(ctx, tx); err != nil {
		return err
	}

	switch {
	case pending <= 0:
		return ErrNoVotesLeft
	case voterID == candidateID:
		return ErrSelfVote
	}

	var candidates, previous int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = ?", candidateID).Scan(&candidates); err != nil {
		return err
	}
	if candidates == 0 {
		return ErrUnknownCandidate
	}
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM votes WHERE voter_passphrase = ? AND user_costume_id = ?",
		passphrase, candidateID).Scan(&previous); err != nil {
		return err
	}
	if previous > 0 {
		return ErrAlreadyVotedFor
	}

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO votes(voter_passphrase, user_costume_id, message) VALUES (?, ?, ?)",
		passphrase, candidateID, message); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET pending_votes = pending_votes - 1 WHERE id = ?", voterID); err != nil {
		return err
	}
	return tx.Commit()
}

// UndoWindow is how long after voting someone can take the vote back.
const UndoWindow = 30 * time.Second

// Undo removes a recent vote and gives the voter the vote back.
func (s *VoteService) Undo(ctx context.Context, passphrase string, candidateID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := checkVotingOpen(ctx, tx); err != nil {
		return err
	}

	var voteID, age int64
	err = tx.QueryRowContext(ctx,
		`SELECT id, TIMESTAMPDIFF(SECOND, created_at, NOW()) FROM votes
		WHERE voter_passphrase = ? AND user_costume_id = ? FOR UPDATE`,
		passphrase, candidateID).Scan(&voteID, &age)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrVoteNotFound
	}
	if err != nil {
		return err
	}
	if age > int64(UndoWindow.Seconds()) {
		return ErrUndoExpired
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM votes WHERE id = ?", voteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE users SET pending_votes = pending_votes + 1 WHERE pw_code = ?", passphrase); err != nil {
		return err
	}
	return tx.Commit()
}

// Messages show up in the feed a random 2 to 8 minutes after the vote, so
// nobody can tell who wrote one by watching who just voted.
const (
	FeedMinDelay    = 2 * time.Minute
	FeedDelaySpread = 6 * time.Minute
)

// Feed returns the latest published messages, newest first, without saying
// who wrote them or when.
func (s *VoteService) Feed(ctx context.Context, limit int) ([]models.FeedItem, error) {
	// The delay is derived from the vote id, so it is stable across requests.
	rows, err := s.db.QueryContext(ctx, `SELECT id, costume, message FROM (
			SELECT v.id, u.costume, v.message,
				v.created_at + INTERVAL (? + CRC32(CONCAT('feed', v.id)) % ?) SECOND AS visible_at
			FROM votes v JOIN users u ON u.id = v.user_costume_id
			WHERE v.message <> ''
		) published
		WHERE visible_at <= NOW()
		ORDER BY visible_at DESC
		LIMIT ?`,
		int(FeedMinDelay.Seconds()), int(FeedDelaySpread.Seconds())+1, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.FeedItem, 0)
	for rows.Next() {
		var item models.FeedItem
		if err := rows.Scan(&item.ID, &item.Costume, &item.Message); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// checkVotingOpen fails unless the admin started the vote and hasn't closed it.
func checkVotingOpen(ctx context.Context, tx *sql.Tx) error {
	var started, finished bool
	if err := tx.QueryRowContext(ctx, "SELECT started, finished FROM game_state WHERE id = 1").
		Scan(&started, &finished); err != nil {
		return err
	}
	if !started || finished {
		return ErrVotingClosed
	}
	return nil
}

// Reset deletes every vote and gives everyone the same number of votes.
func (s *VoteService) Reset(ctx context.Context, votesPerUser int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM votes"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET pending_votes = ?", votesPerUser); err != nil {
		return err
	}
	return tx.Commit()
}

// Seed inserts votes as-is, skipping validation. Used by the mock data loader.
func (s *VoteService) Seed(ctx context.Context, votes []models.Vote) error {
	for _, v := range votes {
		if v.UserCostumeID == 0 {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			"INSERT INTO votes(voter_passphrase, user_costume_id, message) VALUES (?, ?, ?)",
			v.VoterPassphrase, v.UserCostumeID, v.Message); err != nil {
			return err
		}
	}
	return nil
}

const resultsQuery = `SELECT u.id, u.costume, u.name, COUNT(*) AS vote_count
	FROM votes v JOIN users u ON v.user_costume_id = u.id
	GROUP BY u.id, u.costume, u.name
	ORDER BY vote_count DESC`

// Winners returns the top 3 costumes by votes.
func (s *VoteService) Winners(ctx context.Context) ([]models.VoteResult, error) {
	return s.results(ctx, resultsQuery+" LIMIT 3")
}

// Results returns every voted costume with the messages it received.
func (s *VoteService) Results(ctx context.Context) ([]models.VoteResult, error) {
	results, err := s.results(ctx, resultsQuery)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT v.user_costume_id, u.name, v.message
		FROM votes v JOIN users u ON u.pw_code = v.voter_passphrase`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byCandidate := make(map[int64][]models.VoteData)
	for rows.Next() {
		var id int64
		var d models.VoteData
		if err := rows.Scan(&id, &d.User, &d.Message); err != nil {
			return nil, err
		}
		byCandidate[id] = append(byCandidate[id], d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range results {
		results[i].Data = byCandidate[results[i].ID]
	}
	return results, nil
}

// Messages returns who voted for a candidate and what they wrote.
func (s *VoteService) Messages(ctx context.Context, candidateID string) ([]models.VoteData, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.name, v.message
		FROM votes v JOIN users u ON u.pw_code = v.voter_passphrase
		WHERE v.user_costume_id = ?`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]models.VoteData, 0)
	for rows.Next() {
		var d models.VoteData
		if err := rows.Scan(&d.User, &d.Message); err != nil {
			return nil, err
		}
		messages = append(messages, d)
	}
	return messages, rows.Err()
}

func (s *VoteService) results(ctx context.Context, query string) ([]models.VoteResult, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]models.VoteResult, 0)
	for rows.Next() {
		var r models.VoteResult
		if err := rows.Scan(&r.ID, &r.Costume, &r.Name, &r.VotesCount); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

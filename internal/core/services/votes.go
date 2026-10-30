package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"halloween/internal/models"
)

type VoteService struct {
	db *sql.DB
}

func NewVoteService(db *sql.DB) *VoteService {
	return &VoteService{db: db}
}

// Cast validates and stores a vote for a costume, decrementing the voter's
// pending votes. Nobody can vote for the costume they're wearing.
// Everything runs in one transaction with the voter row locked, so two votes
// sent at the same time can't both use the last remaining vote.
func (s *VoteService) Cast(ctx context.Context, passphrase string, costumeID int64, message string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var voterID, pending, ownCostume int64
	err = tx.QueryRowContext(ctx,
		"SELECT id, pending_votes, costume_id FROM users WHERE pw_code = ? FOR UPDATE", passphrase).
		Scan(&voterID, &pending, &ownCostume)
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
	case ownCostume == costumeID:
		return ErrSelfVote
	}

	var candidates, previous, sameGroup int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM costumes c
		WHERE c.id = ? AND EXISTS (SELECT 1 FROM users u WHERE u.costume_id = c.id)`, costumeID).Scan(&candidates); err != nil {
		return err
	}
	if candidates == 0 {
		return ErrUnknownCandidate
	}
	// Friends set by the admin in the same group can't vote for each other.
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM costumes mine JOIN costumes theirs
		ON theirs.friend_group_id = mine.friend_group_id
		WHERE mine.id = ? AND theirs.id = ? AND mine.friend_group_id <> 0`, ownCostume, costumeID).Scan(&sameGroup); err != nil {
		return err
	}
	if sameGroup > 0 {
		return ErrSameGroup
	}
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM votes WHERE voter_passphrase = ? AND costume_id = ?",
		passphrase, costumeID).Scan(&previous); err != nil {
		return err
	}
	if previous > 0 {
		return ErrAlreadyVotedFor
	}

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO votes(voter_passphrase, user_costume_id, costume_id, message) VALUES (?, 0, ?, ?)",
		passphrase, costumeID, message); err != nil {
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
func (s *VoteService) Undo(ctx context.Context, passphrase string, costumeID int64) error {
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
		WHERE voter_passphrase = ? AND costume_id = ? FOR UPDATE`,
		passphrase, costumeID).Scan(&voteID, &age)
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

// visibleAt is when a vote's message shows up in the feed. The delay is
// derived from the vote id, so it is stable across requests.
const visibleAt = `v.created_at + INTERVAL (? + CRC32(CONCAT('feed', v.id)) % ?) SECOND`

func delayArgs() []any {
	return []any{int(FeedMinDelay.Seconds()), int(FeedDelaySpread.Seconds()) + 1}
}

// Feed returns the latest published messages, newest first, without saying
// who wrote them or when. Messages hidden by an admin are left out.
func (s *VoteService) Feed(ctx context.Context, limit int) ([]models.FeedItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, costume, message FROM (
			SELECT v.id, c.name AS costume, v.message, `+visibleAt+` AS visible_at
			FROM votes v JOIN costumes c ON c.id = v.costume_id
			WHERE v.message <> '' AND NOT v.hidden
		) published
		WHERE visible_at <= NOW()
		ORDER BY visible_at DESC
		LIMIT ?`, append(delayArgs(), limit)...)
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

// AdminMessages returns every message right away (even before it's published
// in the feed) with its author, so the admin can hide one in time.
func (s *VoteService) AdminMessages(ctx context.Context) ([]models.AdminMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT v.id, c.name, COALESCE(voter.name, ''), v.message, v.hidden,
			`+visibleAt+` <= NOW()
		FROM votes v
		JOIN costumes c ON c.id = v.costume_id
		LEFT JOIN users voter ON voter.pw_code = v.voter_passphrase
		WHERE v.message <> ''
		ORDER BY v.id DESC`, delayArgs()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]models.AdminMessage, 0)
	for rows.Next() {
		var m models.AdminMessage
		if err := rows.Scan(&m.ID, &m.Costume, &m.Author, &m.Message, &m.Hidden, &m.Published); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// SetHidden hides a message from the live feed (or shows it again). The vote
// itself still counts.
func (s *VoteService) SetHidden(ctx context.Context, id int64, hidden bool) error {
	res, err := s.db.ExecContext(ctx, "UPDATE votes SET hidden = ? WHERE id = ? AND message <> ''", hidden, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// Stats are the public totals shown on the big screen.
type Stats struct {
	Guests   int `json:"guests"`
	Cast     int `json:"cast"`
	Possible int `json:"possible"`
}

func (s *VoteService) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	var pending int
	err := s.db.QueryRowContext(ctx, `SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM votes),
			(SELECT COALESCE(SUM(pending_votes), 0) FROM users)`).Scan(&st.Guests, &st.Cast, &pending)
	st.Possible = st.Cast + pending
	return st, err
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

// Name is everyone wearing the costume, e.g. "Lu, Sol y Tomi".
const resultsQuery = `SELECT c.id, c.name,
		(SELECT GROUP_CONCAT(m.name ORDER BY m.id SEPARATOR '||') FROM users m WHERE m.costume_id = c.id),
		COUNT(*) AS vote_count
	FROM votes v JOIN costumes c ON c.id = v.costume_id
	GROUP BY c.id, c.name
	ORDER BY vote_count DESC, MIN(v.id)`

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

	rows, err := s.db.QueryContext(ctx, `SELECT v.costume_id, u.name, v.message
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

// Messages returns who voted for a costume and what they wrote.
func (s *VoteService) Messages(ctx context.Context, costumeID int64) ([]models.VoteData, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.name, v.message
		FROM votes v JOIN users u ON u.pw_code = v.voter_passphrase
		WHERE v.costume_id = ?`, costumeID)
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
		var members sql.NullString
		if err := rows.Scan(&r.ID, &r.Costume, &members, &r.VotesCount); err != nil {
			return nil, err
		}
		r.Name = JoinNames(strings.Split(members.String, "||"))
		results = append(results, r)
	}
	return results, rows.Err()
}

// JoinNames lists names the Spanish way: "Lu", "Lu y Tomi", "Lu, Sol y Tomi".
func JoinNames(names []string) string {
	clean := names[:0:0]
	for _, n := range names {
		if n != "" {
			clean = append(clean, n)
		}
	}
	if len(clean) <= 1 {
		return strings.Join(clean, "")
	}
	return strings.Join(clean[:len(clean)-1], ", ") + " y " + clean[len(clean)-1]
}

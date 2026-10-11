package services

import (
	"context"
	"database/sql"
)

// GameService stores whether the vote has started and finished.
type GameService struct {
	db *sql.DB
}

func NewGameService(db *sql.DB) *GameService {
	return &GameService{db: db}
}

type GameState struct {
	Started  bool `json:"started"`
	Finished bool `json:"finished"`
	// Registration settings: guests can sign up while RegistrationOpen and
	// the vote hasn't started; each new guest gets VotesPerUser votes.
	RegistrationOpen bool  `json:"registration_open"`
	VotesPerUser     int64 `json:"votes_per_user"`
}

func (s *GameService) State(ctx context.Context) (GameState, error) {
	var st GameState
	err := s.db.QueryRowContext(ctx,
		"SELECT started, finished, registration_open, votes_per_user FROM game_state WHERE id = 1").
		Scan(&st.Started, &st.Finished, &st.RegistrationOpen, &st.VotesPerUser)
	return st, err
}

// SetRegistration opens or closes sign-ups and sets new guests' votes.
func (s *GameService) SetRegistration(ctx context.Context, open bool, votesPerUser int64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE game_state SET registration_open = ?, votes_per_user = ? WHERE id = 1", open, votesPerUser)
	return err
}

// Set stores whether the vote started and finished (used by the admin controls).
func (s *GameService) Set(ctx context.Context, st GameState) error {
	_, err := s.db.ExecContext(ctx, "UPDATE game_state SET started = ?, finished = ? WHERE id = 1", st.Started, st.Finished)
	return err
}

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
}

func (s *GameService) State(ctx context.Context) (GameState, error) {
	var st GameState
	err := s.db.QueryRowContext(ctx, "SELECT started, finished FROM game_state WHERE id = 1").
		Scan(&st.Started, &st.Finished)
	return st, err
}

func (s *GameService) SetStarted(ctx context.Context, started bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE game_state SET started = ? WHERE id = 1", started)
	return err
}

func (s *GameService) SetFinished(ctx context.Context, finished bool) error {
	_, err := s.db.ExecContext(ctx, "UPDATE game_state SET finished = ? WHERE id = 1", finished)
	return err
}

// Set stores the whole state at once (used by the admin controls).
func (s *GameService) Set(ctx context.Context, st GameState) error {
	_, err := s.db.ExecContext(ctx, "UPDATE game_state SET started = ?, finished = ? WHERE id = 1", st.Started, st.Finished)
	return err
}

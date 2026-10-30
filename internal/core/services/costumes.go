package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"halloween/internal/models"
)

var (
	ErrRegistrationClosed = errors.New("registration is closed")
	ErrAlreadyRegistered  = errors.New("someone with that name is already in this costume")
	ErrDuplicateName      = errors.New("the same name is listed twice")
	ErrMissingCostume     = errors.New("costume is required")
	ErrCostumeNotFound    = errors.New("costume not found")
	ErrTooManyPartners    = errors.New("too many people in one costume")
	ErrTooLong            = errors.New("name or costume is too long")
	ErrGuestLimit         = errors.New("the guest list is full")
	ErrSameGroup          = errors.New("you can't vote for your own group of friends")
)

const (
	MaxPartners    = 8
	maxNameLen     = 40
	maxCostumeLen  = 60
	maxGuests      = 500
	maxGroupLen    = 60
	costumesSelect = `SELECT c.id, c.name, c.friend_group_id, u.name FROM costumes c
		JOIN users u ON u.costume_id = c.id
		ORDER BY c.name, c.id, u.id`
)

// Costumes lists every costume that has someone in it, with its members.
func (s *UserService) Costumes(ctx context.Context) ([]models.Costume, error) {
	rows, err := s.db.QueryContext(ctx, costumesSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	costumes := make([]models.Costume, 0)
	for rows.Next() {
		var id, group int64
		var name, member string
		if err := rows.Scan(&id, &name, &group, &member); err != nil {
			return nil, err
		}
		if n := len(costumes); n == 0 || costumes[n-1].ID != id {
			costumes = append(costumes, models.Costume{ID: id, Name: name, GroupID: group})
		}
		last := &costumes[len(costumes)-1]
		last.Members = append(last.Members, member)
	}
	return costumes, rows.Err()
}

// Registration is what a guest fills in: their costume (new, or an existing
// one to join), their name and the names of whoever shares the costume.
type Registration struct {
	Costume   string   `json:"costume"`
	CostumeID int64    `json:"costume_id"`
	Name      string   `json:"name"`
	Partners  []string `json:"partners"`
}

// Register signs up a guest and their costume partners in one go. Everyone
// gets their own pass code and the votes the admin set for new guests. It
// only works before the vote starts and while the admin keeps it open.
func (s *UserService) Register(ctx context.Context, reg Registration) (models.Costume, []models.User, error) {
	var costume models.Costume
	names, err := cleanNames(reg.Name, reg.Partners)
	if err != nil {
		return costume, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return costume, nil, err
	}
	defer tx.Rollback()

	var started, finished, open bool
	var votes int64
	if err := tx.QueryRowContext(ctx,
		"SELECT started, finished, registration_open, votes_per_user FROM game_state WHERE id = 1 FOR UPDATE").
		Scan(&started, &finished, &open, &votes); err != nil {
		return costume, nil, err
	}
	if started || finished || !open {
		return costume, nil, ErrRegistrationClosed
	}
	var guests int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&guests); err != nil {
		return costume, nil, err
	}
	if guests+len(names) > maxGuests {
		return costume, nil, ErrGuestLimit
	}

	if reg.CostumeID > 0 {
		// Joining a costume someone already registered.
		err := tx.QueryRowContext(ctx, "SELECT id, name FROM costumes WHERE id = ? FOR UPDATE", reg.CostumeID).
			Scan(&costume.ID, &costume.Name)
		if errors.Is(err, sql.ErrNoRows) {
			return costume, nil, ErrCostumeNotFound
		}
		if err != nil {
			return costume, nil, err
		}
		for _, name := range names {
			var taken int
			if err := tx.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE costume_id = ? AND LOWER(name) = LOWER(?)", costume.ID, name).
				Scan(&taken); err != nil {
				return costume, nil, err
			}
			if taken > 0 {
				return costume, nil, ErrAlreadyRegistered
			}
		}
	} else {
		costume.Name = strings.Join(strings.Fields(reg.Costume), " ")
		switch {
		case costume.Name == "":
			return costume, nil, ErrMissingCostume
		case len([]rune(costume.Name)) > maxCostumeLen:
			return costume, nil, ErrTooLong
		}
		res, err := tx.ExecContext(ctx, "INSERT INTO costumes(name) VALUES (?)", costume.Name)
		if err != nil {
			return costume, nil, err
		}
		costume.ID, _ = res.LastInsertId()
	}

	users := make([]models.User, 0, len(names))
	for _, name := range names {
		u := models.User{Name: name, Costume: costume.Name, CostumeID: costume.ID, PendingVotes: votes}
		if err := insertUser(ctx, tx, &u); err != nil {
			return costume, nil, err
		}
		users = append(users, u)
	}
	if err := tx.Commit(); err != nil {
		return costume, nil, err
	}
	return costume, users, nil
}

// cleanNames trims the registrant's and partners' names (dropping empty
// partner rows) and rejects repeated or overly long ones.
func cleanNames(name string, partners []string) ([]string, error) {
	names := []string{strings.Join(strings.Fields(name), " ")}
	if names[0] == "" {
		return nil, ErrMissingName
	}
	for _, p := range partners {
		if p = strings.Join(strings.Fields(p), " "); p != "" {
			names = append(names, p)
		}
	}
	if len(names) > MaxPartners+1 {
		return nil, ErrTooManyPartners
	}
	seen := map[string]bool{}
	for _, n := range names {
		if len([]rune(n)) > maxNameLen {
			return nil, ErrTooLong
		}
		key := strings.ToLower(n)
		if seen[key] {
			return nil, ErrDuplicateName
		}
		seen[key] = true
	}
	return names, nil
}

// insertUser stores a user with a fresh pass code, filling in ID and PWCode.
func insertUser(ctx context.Context, tx *sql.Tx, u *models.User) error {
	var err error
	if u.PWCode, err = uniquePass(ctx, tx, u.Name); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		"INSERT INTO users(is_admin, name, pw_code, costume, costume_id, pending_votes) VALUES (?, ?, ?, ?, ?, ?)",
		u.IsAdmin, u.Name, u.PWCode, u.Costume, u.CostumeID, u.PendingVotes)
	if err != nil {
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

// RenameCostume changes a costume's name for all its members.
func (s *UserService) RenameCostume(ctx context.Context, id int64, name string) error {
	name = strings.Join(strings.Fields(name), " ")
	switch {
	case name == "":
		return ErrMissingCostume
	case len([]rune(name)) > maxCostumeLen:
		return ErrTooLong
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE costumes SET name = ? WHERE id = ?", name, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCostumeNotFound
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET costume = ? WHERE costume_id = ?", name, id); err != nil {
		return err
	}
	return tx.Commit()
}

// createCostume adds a new costume with that name, even if another one is
// called the same. An empty name means no costume (id 0).
func createCostume(ctx context.Context, tx *sql.Tx, name string) (int64, string, error) {
	name = strings.Join(strings.Fields(name), " ")
	switch {
	case name == "":
		return 0, "", nil
	case len([]rune(name)) > maxCostumeLen:
		return 0, "", ErrTooLong
	}
	res, err := tx.ExecContext(ctx, "INSERT INTO costumes(name) VALUES (?)", name)
	if err != nil {
		return 0, "", err
	}
	id, _ := res.LastInsertId()
	return id, name, nil
}

// dropEmptyCostumes deletes costumes nobody is wearing anymore. Their votes
// are deleted too, and whoever cast them gets the vote back.
func dropEmptyCostumes(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT c.id FROM costumes c
		WHERE NOT EXISTS (SELECT 1 FROM users u WHERE u.costume_id = c.id)`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range ids {
		for _, q := range []string{
			`UPDATE users u JOIN votes v ON v.voter_passphrase = u.pw_code
				SET u.pending_votes = u.pending_votes + 1 WHERE v.costume_id = ?`,
			"DELETE FROM votes WHERE costume_id = ?",
			"DELETE FROM costumes WHERE id = ?",
		} {
			if _, err := tx.ExecContext(ctx, q, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// SetFriendGroup puts a costume in the group of friends with that name
// (ignoring case), creating the group if needed. An empty name takes it out
// of its group. Groups left without costumes are deleted.
func (s *UserService) SetFriendGroup(ctx context.Context, costumeID int64, group string) error {
	group = strings.Join(strings.Fields(group), " ")
	if len([]rune(group)) > maxGroupLen {
		return ErrTooLong
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var groupID int64
	if group != "" {
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM friend_groups WHERE LOWER(name) = LOWER(?) ORDER BY id LIMIT 1", group).Scan(&groupID)
		if errors.Is(err, sql.ErrNoRows) {
			res, err := tx.ExecContext(ctx, "INSERT INTO friend_groups(name) VALUES (?)", group)
			if err != nil {
				return err
			}
			groupID, _ = res.LastInsertId()
		} else if err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, "UPDATE costumes SET friend_group_id = ? WHERE id = ?", groupID, costumeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCostumeNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM friend_groups
		WHERE id NOT IN (SELECT friend_group_id FROM costumes)`); err != nil {
		return err
	}
	return tx.Commit()
}

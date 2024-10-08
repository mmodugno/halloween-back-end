package models

type User struct {
	ID           int64  `json:"id"`
	IsAdmin      bool   `json:"is_admin"`
	Name         string `json:"name"`
	PWCode       string `json:"pw_code"`
	PendingVotes int64  `json:"pending_votes"`
	Costume      string `json:"costume"`
}

type LoggedUser struct {
	ID           int64 `json:"id"`
	IsAdmin      bool  `json:"is_admin"`
	PendingVotes int64 `json:"pending_votes"`
}

package models

type User struct {
	ID           int64  `json:"id"`
	IsAdmin      bool   `json:"is_admin"`
	Username     string `json:"username"`
	PWCode       string `json:"pw_code"`
	PendingVotes int    `json:"pending_votes"`
}

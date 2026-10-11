package models

type User struct {
	ID           int64  `json:"id"`
	IsAdmin      bool   `json:"is_admin"`
	Name         string `json:"name"`
	PWCode       string `json:"pw_code,omitempty"`
	PendingVotes int64  `json:"pending_votes"`
	Costume      string `json:"costume"`
	CostumeID    int64  `json:"costume_id"`
}

type LoggedUser struct {
	ID           int64 `json:"id"`
	IsAdmin      bool  `json:"is_admin"`
	PendingVotes int64 `json:"pending_votes"`
	CostumeID    int64 `json:"costume_id"`
}

// Costume is what people vote for. Several people can share one.
type Costume struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

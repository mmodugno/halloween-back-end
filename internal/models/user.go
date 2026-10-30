package models

type User struct {
	ID           int64  `json:"id"`
	IsAdmin      bool   `json:"is_admin"`
	Name         string `json:"name"`
	PWCode       string `json:"pw_code,omitempty"`
	PendingVotes int64  `json:"pending_votes"`
	Costume      string `json:"costume"`
	CostumeID    int64  `json:"costume_id"`
	// Friend group of the user's costume. Only filled for the admin.
	FriendGroup string `json:"friend_group,omitempty"`
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
	// GroupID tells which costumes belong to the same group of friends (0 =
	// none). The group's name is only shown to the admin.
	GroupID int64 `json:"group_id,omitempty"`
}

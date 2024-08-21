package models

type User struct {
	ID       int64  `json:"id"`
	IsAdmin  bool   `json:"is_admin"`
	Name     string `json:"name"`
	PWCode   string `json:"pw_code"`
	HasVoted bool   `json:"has_voted"`
	Costume  string `json:"costume"`
}

type LoggedUser struct {
	ID       int64 `json:"id"`
	IsAdmin  bool  `json:"is_admin"`
	HasVoted bool  `json:"has_voted"`
}

package models

type Costume struct {
	Owner       string `json:"owner"`
	Description string `json:"description"`
	Votes       int    `json:"votes"`
}

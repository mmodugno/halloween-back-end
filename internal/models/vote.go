package models

type Vote struct {
	ID              string `json:"id"`
	VoterPassphrase string `json:"voter_passphrase"`
	UserCostumeID   int    `json:"user_costume_id"`
	Message         string `json:"message"`
}

type VoteResult struct {
	Costume    string     `json:"costume"`
	Name       string     `json:"name"`
	VotesCount int        `json:"votes_count"`
	Data       []VoteData `json:"data,omitempty"`
}

type VoteData struct {
	User    string `json:"user"`
	Message string `json:"message,omitempty"`
}

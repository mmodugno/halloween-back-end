package models

type Vote struct {
	ID              string `json:"id"`
	VoterPassphrase string `json:"voter_passphrase"`
	UserCostumeID   string `json:"user_costume_id"`
	Message         string `json:"message"`
}

type VoteResult struct {
	Costume    string `json:"costume"`
	Name       string `json:"name"`
	VotesCount int    `json:"votes_count"`
}

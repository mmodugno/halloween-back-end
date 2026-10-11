package models

type Vote struct {
	ID              string `json:"id"`
	VoterPassphrase string `json:"voter_passphrase"`
	CostumeID       int64  `json:"costume_id"`
	Message         string `json:"message"`
}

type VoteResult struct {
	ID         int64      `json:"id"`
	Costume    string     `json:"costume"`
	Name       string     `json:"name"`
	VotesCount int        `json:"votes_count"`
	Data       []VoteData `json:"data,omitempty"`
}

type VoteData struct {
	User    string `json:"user"`
	Message string `json:"message,omitempty"`
}

// FeedItem is an anonymous message shown live while people vote.
type FeedItem struct {
	ID      int64  `json:"id"`
	Costume string `json:"costume"`
	Message string `json:"message"`
}

// AdminMessage is a vote message as the admin sees it while moderating.
type AdminMessage struct {
	ID        int64  `json:"id"`
	Costume   string `json:"costume"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	Hidden    bool   `json:"hidden"`
	Published bool   `json:"published"`
}

package handlers

import (
	"encoding/json"
	"fmt"
	"halloween/internal/core/services"
	"halloween/internal/models"
	"net/http"
)

func PostVote(w http.ResponseWriter, r *http.Request) {
	var vote models.Vote
	err := json.NewDecoder(r.Body).Decode(&vote)
	if err != nil || vote.UserCostumeID == "" {
		ErrorBuilder(w, err, http.StatusBadRequest)
		return
	}

	passphrase := r.Header.Get("User")
	if passphrase == "" {
		ErrorBuilder(w, fmt.Errorf("passphrase is required"), http.StatusBadRequest)
		return
	}
	ucli := &services.UserClient{}
	user, err := ucli.GetUserByPathphrase(passphrase)
	if err != nil {
		ErrorBuilder(w, fmt.Errorf("user does not exist"), http.StatusBadRequest)
		return
	}

	//If the user has already voted cannot vote again
	if user.HasVoted {
		ErrorBuilder(w, fmt.Errorf("user has already voted"), http.StatusBadRequest)
		return
	}

	vote.VoterPassphrase = passphrase
	cli := &services.VotesClient{}
	err = cli.InsertVote(vote)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	//Set user's has_vote value true
	err = ucli.Vote(user)
	if err != nil {
		ErrorBuilder(w, fmt.Errorf("error in updating users' vote"), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("created %s's vote successfully", vote.VoterPassphrase),
		Code:    http.StatusCreated,
	})
	return
}

func GetWinners(w http.ResponseWriter, _ *http.Request) {
	cli := &services.VotesClient{}
	winners, err := cli.GetWinner()
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(winners)
	return
}

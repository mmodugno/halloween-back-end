package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"halloween/internal/core/services"
	"halloween/internal/models"
	"log"
	"net/http"
)

func PostVote(w http.ResponseWriter, r *http.Request) {
	var vote models.Vote
	err := json.NewDecoder(r.Body).Decode(&vote)

	log.Println(vote)

	if err != nil || vote.UserCostumeID == 0 {
		ErrorBuilder(w, err, http.StatusBadRequest)
		return
	}

	passphrase := r.Header.Get("User")
	if passphrase == "" {
		ErrorBuilder(w, fmt.Errorf("passphrase is required"), http.StatusBadRequest)
		return
	}
	ucli := &services.UserClient{}
	log.Println(passphrase)
	user, err := ucli.GetUserByPathphrase(passphrase)
	if err != nil {
		ErrorBuilder(w, fmt.Errorf("user does not exist"), http.StatusBadRequest)
		return
	}

	//If the user has already voted cannot vote again
	if user.PendingVotes == 0 {
		ErrorBuilder(w, fmt.Errorf("user has already voted twice"), http.StatusInternalServerError)
		return
	}

	vote.VoterPassphrase = passphrase
	cli := &services.VotesClient{}
	err = cli.InsertVote(vote)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
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
	winners, err := cli.GetWinners()
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(winners)
	return
}

func GetResults(w http.ResponseWriter, _ *http.Request) {
	cli := &services.VotesClient{}
	winners, err := cli.GetResults()
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(winners)
	return
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cli := &services.VotesClient{}
	messages, err := cli.GetMessages(id)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(messages)
	return
}

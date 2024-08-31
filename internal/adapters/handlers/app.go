package handlers

import (
	"encoding/json"
	"fmt"
	"halloween/internal/core/services"
	"halloween/internal/models"
	"io/ioutil"
	"net/http"
	"os"
)

func PutFinish(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	isFinished = true
	w.WriteHeader(http.StatusOK)
}

func CancelFinish(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	isFinished = false
	w.WriteHeader(http.StatusOK)
}

func GetFinish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	type Response struct {
		Message bool
		Code    int
	}

	json.NewEncoder(w).Encode(Response{
		Message: isFinished,
		Code:    http.StatusOK,
	})
}

func TestsBach(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonFile, err := os.Open("mock.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	users := make([]models.User, 0)
	json.Unmarshal(byteValue, &users)
	ucli := &services.UserClient{}
	ucli.InsertUsers(users, true)

	votes := make([]models.Vote, 0)
	json.Unmarshal(byteValue, &votes)
	vcli := &services.VotesClient{}
	vcli.InsertVotes(votes)

	w.WriteHeader(http.StatusOK)
}

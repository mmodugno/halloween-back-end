package handlers

import (
	"encoding/json"
	"fmt"
	"halloween/internal/core/services"
	"halloween/internal/models"
	"net/http"
)

// PostUser : creates a new user
func PostUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		ErrorBuilder(w, err, http.StatusBadRequest)
		return
	}
	uclient := &services.UserClient{}
	err = uclient.InsertUser(user)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("created %s's user successfully", user.Username),
		Code:    http.StatusCreated,
	})
	return
}

func ErrorBuilder(w http.ResponseWriter, e error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("error: %s", e.Error()),
		Code:    code,
	})
}

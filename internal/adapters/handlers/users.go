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
		Message: fmt.Sprintf("created %s's user successfully", user.Name),
		Code:    http.StatusCreated,
	})
	return
}

// PostUsers : creates multiple new users
func PostUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	err := json.NewDecoder(r.Body).Decode(&users)
	if err != nil {
		ErrorBuilder(w, err, http.StatusBadRequest)
		return
	}
	uclient := &services.UserClient{}
	err = uclient.InsertUsers(users, false)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("created %d users successfully", len(users)),
		Code:    http.StatusCreated,
	})
	return
}

func Login(w http.ResponseWriter, r *http.Request) {
	password := r.Header.Get("User")
	if password == "" {
		ErrorBuilder(w, fmt.Errorf("error getting password"), http.StatusInternalServerError)
		return
	}
	uclient := &services.UserClient{}
	user, err := uclient.LogIn(password)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	json.NewEncoder(w).Encode(user)
	return
}

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	uclient := &services.UserClient{}
	users, err := uclient.GetAllUsers()
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(users)
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

func GetUserByPassphrase(w http.ResponseWriter, r *http.Request) {
	pw := r.Header.Get("User")
	if pw == "" {
		ErrorBuilder(w, fmt.Errorf("error getting passphrase"), http.StatusInternalServerError)
		return
	}
	uclient := &services.UserClient{}
	user, err := uclient.GetUserByPathphrase(pw)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	json.NewEncoder(w).Encode(user)
	return
}

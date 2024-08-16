package handlers

import (
	"encoding/json"
	"fmt"
	"halloween/internal/core/services"
	"halloween/internal/models"
	"net/http"
)

// PostCostume : Creates a new costume
func PostCostume(w http.ResponseWriter, r *http.Request) {
	var costume models.Costume
	err := json.NewDecoder(r.Body).Decode(&costume)
	if err != nil {
		ErrorBuilder(w, err, http.StatusBadRequest)
		return
	}
	cclient := &services.CostumeClient{}
	err = cclient.InsertCostume(costume)
	if err != nil {
		ErrorBuilder(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: fmt.Sprintf("created %s's costume as %s successfully", costume.Owner, costume.Description),
		Code:    http.StatusCreated,
	})
	return
}

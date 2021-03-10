package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type addMessageRequest struct {
	Message string
}

type addMessageResponse struct {
	Error string `json:"error,omitempty"`
}

func addMessage(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var request addMessageRequest
	err := decoder.Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(addMessageResponse{Error: "No message"})
		return
	}
	u := uuid.New()
	log.WithFields(log.Fields{
		"msg":  request.Message,
		"uuid": u,
	}).Debug("Got message")

	w.WriteHeader(http.StatusOK)
}

func main() {
	log.SetLevel(log.DebugLevel)
	router := mux.NewRouter()
	router.Path("/message/add").HandlerFunc(addMessage).Methods(http.MethodPost)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:8081",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

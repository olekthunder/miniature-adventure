package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// I don't want to parse configs for now
const LOGGING_SERVICE_ENDPOINT = "http://localhost:8082/log/add"

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
	u := uuid.New().String()
	log.WithFields(log.Fields{
		"msg":  request.Message,
		"uuid": u,
	}).Debug("Got message")

	// Don't wait for response
	go sendLog(u, request.Message)

	w.WriteHeader(http.StatusOK)
}

type logServiceRequest struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

func sendLog(uuid string, message string) {
	request := logServiceRequest{uuid, message}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(request)
	logReq, err := http.NewRequest(
		http.MethodPost,
		LOGGING_SERVICE_ENDPOINT,
		buf,
	)
	if err != nil {
		log.Error(err)
		return
	}
	client := &http.Client{}
	resp, err := client.Do(logReq)
	if err != nil {
		log.Error(err)
		return
	}
	defer resp.Body.Close()
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

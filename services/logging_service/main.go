package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var LOG_MAP = make(map[string]string)

type addLogRequest struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

type addLogResponse struct {
	Error string `json:"error,omitempty"`
}

func addLog(w http.ResponseWriter, r *http.Request) {
	var request addLogRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || len(request.UUID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(addLogResponse{Error: "Bad request"})
		return
	}
	LOG_MAP[request.UUID] = request.Message
	log.Info(request.Message)

	w.WriteHeader(http.StatusOK)
}

func listLogs(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	for _, v := range LOG_MAP {
		fmt.Fprintf(w, "%s\n", v)
	}
}

func main() {
	router := mux.NewRouter()
	logRouter := router.PathPrefix("/log").Subrouter()
	logRouter.Path("/add").HandlerFunc(addLog).Methods(http.MethodPost)
	logRouter.Path("/list").HandlerFunc(listLogs).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:8082",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// I don't want to parse configs for now
const LOG_ADD_ENDPOINT = "http://localhost:8082/log/add"
const LOG_LIST_ENDPOINT = "http://localhost:8082/log/list"
const MESSAGES_ENDPOINT = "http://localhost:8083/"

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
		LOG_ADD_ENDPOINT,
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

func index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		logResponse     string
		messageResponse string
	}{}
	g := errgroup.Group{}
	g.Go(func() error {
		resp, err := http.Get(LOG_LIST_ENDPOINT)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data.logResponse = string(respData)
		return nil
	})
	g.Go(func() error {
		resp, err := http.Get(MESSAGES_ENDPOINT)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		data.messageResponse = string(respData)
		return nil
	})
	err := g.Wait()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Message:")
	fmt.Fprintln(w, data.messageResponse)
	fmt.Fprintln(w, "Logs:")
	fmt.Fprintln(w, data.logResponse)
}

func main() {
	log.SetLevel(log.DebugLevel)
	router := mux.NewRouter()
	router.Path("/").HandlerFunc(index).Methods(http.MethodGet)
	router.Path("/message/add").HandlerFunc(addMessage).Methods(http.MethodPost)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:8081",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

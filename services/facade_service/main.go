package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

var LOGGING_SERVICE_ADDRS = strings.Split(os.Getenv("LOGGING_SERVICE_ADDRS"), ",")
const LOG_ADD_ENDPOINT = "http://%v/log/add"
const LOG_LIST_ENDPOINT = "http://%v/log/list"
var MESSAGES_SERVICE_ADDRS = []string{
	"messages_service1:8083",
	"messages_service2:8083",
	"messages_service3:8083",
}
const Q_NAME = "message_queue"


func init() {
	rand.Seed(time.Now().Unix())
}

type addMessageRequest struct {
	Message string
}

type addMessageResponse struct {
	Error string `json:"error,omitempty"`
}

type addMessageHandler struct {
	ch *amqp.Channel
}

func (amh *addMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	err = amh.ch.Publish(
		"",
		Q_NAME,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(request.Message),
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Published %v\n", request.Message)
	// Don't wait for response
	go sendLog(u, request.Message)

	w.WriteHeader(http.StatusOK)
}

func newAddMessageHandler(ch *amqp.Channel) *addMessageHandler {
	return &addMessageHandler{ch: ch}
}

type logServiceRequest struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

func sendLog(uuid string, message string) {
	request := logServiceRequest{uuid, message}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(request)
	logAddEndpoint := getLogAddEndpoint()
	logReq, err := http.NewRequest(
		http.MethodPost,
		logAddEndpoint,
		buf,
	)
	if err != nil {
		log.Error(err)
		return
	}
	client := &http.Client{}
	log.Printf("Sending log to %v", logAddEndpoint)
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
	logListEndpoint := getLogListEndpoint()
	messagesEndpoint := getMessagesEndpoint()
	g.Go(func() error {
		log.Printf("Querying %v", logListEndpoint)
		resp, err := http.Get(logListEndpoint)
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
		log.Printf("Querying %v", messagesEndpoint)
		resp, err := http.Get(messagesEndpoint)
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

func getLogServiceAddr() string {
	return LOGGING_SERVICE_ADDRS[rand.Intn(len(LOGGING_SERVICE_ADDRS))]
}

func getLogListEndpoint() string {
	return fmt.Sprintf(LOG_LIST_ENDPOINT, getLogServiceAddr())
}

func getLogAddEndpoint() string {
	return fmt.Sprintf(LOG_ADD_ENDPOINT, getLogServiceAddr())
}

func getMessagesServiceAddr() string {
	return MESSAGES_SERVICE_ADDRS[rand.Intn(len(MESSAGES_SERVICE_ADDRS))]
}

func getMessagesEndpoint() string {
	return fmt.Sprintf("http://%v/", getMessagesServiceAddr())
}

func rmqConnect() *amqp.Connection {
	for {
		conn, err := amqp.Dial("amqp://guest:guest@rmq:5672/")
		if err == nil {
			fmt.Println("Connected to rmq")
			return conn
		} else {
			fmt.Println("Failed to connect, retrying...")
			time.Sleep(time.Second * 2)
		}
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	conn := rmqConnect()
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	defer ch.Close()
	addMessageH := newAddMessageHandler(ch)
	router := mux.NewRouter()
	router.Path("/").HandlerFunc(index).Methods(http.MethodGet)
	router.Path("/message/add").Handler(addMessageH).Methods(http.MethodPost)

	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:8081",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

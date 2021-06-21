package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

var LOGGING_SERVICE_ADDRS = strings.Split(os.Getenv("LOGGING_SERVICE_ADDRS"), ",")
const LOG_ADD_ENDPOINT = "http://%v/log/add"
const LOG_LIST_ENDPOINT = "http://%v/log/list"
var MESSAGES_SERVICE_ADDRS = strings.Split(os.Getenv("MESSAGES_SERVICE_ADDRS"), ",")

func getEnvOrExit(name string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	} else {
		log.Fatalf("Env var %v is missing\n", name)
		return "" // to suppress compiler errs
	}
}

type appConfig struct {
	port int
	serviceID string
	serviceName string
	messageQueueName string
	consulAddr string
	rabbitURL string
}

func newAppConfig() *appConfig {
	cfg := new(appConfig)
	cfg.messageQueueName = getEnvOrExit("MESSAGE_QUEUE_NAME")
	if port, err := strconv.Atoi(getEnvOrExit("PORT")); err != nil {
		log.Fatalln(err)
	} else {
		cfg.port = port
	}
	cfg.serviceName = getEnvOrExit("SERVICE_NAME")
	cfg.serviceID = getEnvOrExit("SERVICE_ID")
	cfg.consulAddr = getEnvOrExit("CONSUL_ADDR")
	cfg.rabbitURL = getEnvOrExit("RABBIT_URL")
	return cfg
}

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
	cfg *appConfig
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
		amh.cfg.messageQueueName,
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

func newAddMessageHandler(cfg *appConfig, ch *amqp.Channel) *addMessageHandler {
	return &addMessageHandler{cfg: cfg, ch: ch}
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

func rmqConnect(cfg *appConfig) *amqp.Connection {
	for {
		conn, err := amqp.Dial(cfg.rabbitURL)
		if err == nil {
			fmt.Println("Connected to rmq")
			return conn
		} else {
			fmt.Println("Failed to connect, retrying...")
			time.Sleep(time.Second * 2)
		}
	}
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func initConsul(cfg *appConfig) *consul.Client {
	config := consul.DefaultConfig()
	config.Address = cfg.consulAddr
	consulClient, err := consul.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	registration := new(consul.AgentServiceRegistration)

	registration.ID = cfg.serviceID
	registration.Name = cfg.serviceName
	addr, err := os.Hostname()
	if err != nil {
		log.Fatalln(err)
	}
	registration.Address = addr
	registration.Port = cfg.port
	registration.Check = new(consul.AgentServiceCheck)
	registration.Check.HTTP = fmt.Sprintf("http://%s:%v/health", addr, registration.Port)
	registration.Check.Interval = "5s"
	registration.Check.Timeout = "30s"
	consulClient.Agent().ServiceRegister(registration)
	log.Infof("Registered %v!", cfg.serviceName)
	return consulClient
}

func lookupService(consulClient *consul.Client, name string) (string, error) {
	log.Infof("looking up %v", name)
	time.Sleep(time.Second  * 30)
	status, services, err := consulClient.Agent().AgentHealthServiceByName(name)
	if err != nil {
		log.Fatalln(err)
	}
	if status == consul.HealthCritical {
		log.Fatalln("Status critical")
	}
	log.Infof("Found %v", name)
	for _, srvc := range services {
		log.Info(fmt.Sprintf("%v:%v", srvc.Service.Address, srvc.Service.Port))
	}
	return "", nil
}

func main() {
	cfg := newAppConfig()
	consulClient := initConsul(cfg)
	go lookupService(consulClient, cfg.serviceName)
	log.SetLevel(log.DebugLevel)
	conn := rmqConnect(cfg)
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	defer ch.Close()
	addMessageH := newAddMessageHandler(cfg, ch)
	router := mux.NewRouter()
	router.Path("/").HandlerFunc(index).Methods(http.MethodGet)
	router.Path("/message/add").Handler(addMessageH).Methods(http.MethodPost)
	router.Path("/health").HandlerFunc(healthCheck).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("0.0.0.0:%v", cfg.port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Info("Running server..")
	log.Fatal(srv.ListenAndServe())
}

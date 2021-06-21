package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

const Q_NAME = "message_queue"

var messages = []string{} // mutable global storage

func getEnvOrExit(name string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	} else {
		log.Fatalf("Env var %v is missing\n", name)
		return "" // to suppress compiler errs
	}
}

type appConfig struct {
	port             int
	messageQueueName string
	rabbitURL        string
}

func newAppConfig() *appConfig {
	cfg := new(appConfig)
	cfg.messageQueueName = getEnvOrExit("MESSAGE_QUEUE_NAME")
	if port, err := strconv.Atoi(getEnvOrExit("PORT")); err != nil {
		log.Fatalln(err)
	} else {
		cfg.port = port
	}
	cfg.rabbitURL = getEnvOrExit("RABBIT_URL")
	return cfg
}

func index(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(messages)
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

func startConsumer(cfg *appConfig, ch *amqp.Channel) {
	_, err := ch.QueueDeclare(cfg.messageQueueName, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		fmt.Println("Consuming...")
		msgs, err := ch.Consume(
			cfg.messageQueueName,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Fatalln(err)
		}
		for m := range msgs {
			mstr := string(m.Body)
			fmt.Printf("Consumed %v\n", mstr)
			messages = append(messages, mstr)
		}
	}
}

func main() {
	cfg := newAppConfig()
	conn := rmqConnect(cfg)
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	go startConsumer(cfg, ch)
	r := mux.NewRouter()
	r.Path("/").HandlerFunc(index)
	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf("0.0.0.0:%v", cfg.port),
	}
	log.Fatal(srv.ListenAndServe())
}

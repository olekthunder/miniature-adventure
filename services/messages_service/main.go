package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

const Q_NAME = "message_queue"
var messages = []string{} // mutable global storage

func index(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(messages)
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

func startConsumer(ch *amqp.Channel) {
	_, err := ch.QueueDeclare(Q_NAME, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		fmt.Println("Consuming...")
		msgs, err := ch.Consume(
			Q_NAME,
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
	conn := rmqConnect()
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	go startConsumer(ch)
	r := mux.NewRouter()
	r.Path("/").HandlerFunc(index)
	srv := &http.Server{
		Handler: r,
		Addr: "0.0.0.0:8083",
	}
	log.Fatal(srv.ListenAndServe())
}
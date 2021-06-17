package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

var Q_NAME = "tq6"

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
	conn := rmqConnect()
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	defer ch.Close()
	_, err = ch.QueueDeclare(Q_NAME, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		msgs, err := ch.Consume(
			Q_NAME,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			continue
		}
		for m := range msgs {
			fmt.Printf("Consumed %v\n", string(m.Body))
		}
	}
}

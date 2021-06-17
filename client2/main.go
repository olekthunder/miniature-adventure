package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

const PROCESSING_Q = "pq2"
const RESPONSE_Q = "rq2"

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
	_, err = ch.QueueDeclare(PROCESSING_Q, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = ch.QueueDeclare(RESPONSE_Q, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	forever := make(chan bool)
	go func () {
		for i := 0; i < 10; i++ {
			msg := fmt.Sprintf("msg%v", i)
			err = ch.Publish(
				"",
				PROCESSING_Q,
				false,
				false,
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(msg),
				},
			)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("Published %v\n", msg)
			time.Sleep(time.Second)
		}
	}()

	go func () {
		for {
			msgs, err := ch.Consume(
				RESPONSE_Q,
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
				fmt.Printf("Consumed %v\n", string(m.Body))
			}
		}
	}()
	<-forever
}

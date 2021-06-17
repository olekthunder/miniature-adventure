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
			fmt.Println("Successfully Connected to our RabbitMQ Instance")
			return conn
		} else {
			fmt.Println("Failed to connect, retrying...")
			time.Sleep(time.Second * 2)
		}
	}
}

func processMsg(msg string, ch *amqp.Channel) {
	err := ch.Publish(
		"",
		RESPONSE_Q,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(fmt.Sprintf("%v_processed", msg)),
		},
	)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Published %v\n", msg)
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
	for {
		msgs, err := ch.Consume(
			PROCESSING_Q,
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
			processMsg(mstr, ch)
		}
	}
}

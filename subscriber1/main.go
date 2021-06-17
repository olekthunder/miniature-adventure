package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

const EXCHANGE_NAME = "pub1"

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
	err = ch.ExchangeDeclare(EXCHANGE_NAME, amqp.ExchangeFanout, false, false, false, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		log.Fatalln(err)
	}
	if err = ch.QueueBind(q.Name, "", EXCHANGE_NAME, false, nil); err != nil {
		log.Fatalln(err)
	}
	for {
		fmt.Printf("Consuming from %v\n", q.Name)
		msgs, err := ch.Consume(
			q.Name,
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

}

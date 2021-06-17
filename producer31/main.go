package main

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

const Q_NAME = "tq31"

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
	_, err = ch.QueueDeclare(Q_NAME, false, false, false, false, amqp.Table{"x-max-length": 5})
	if err != nil {
		log.Fatalln(err)
	}
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("msg%v", i)
		err = ch.Publish(
			"",
			Q_NAME,
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
	}

}

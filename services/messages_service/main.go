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
	consul "github.com/hashicorp/consul/api"
	"github.com/streadway/amqp"
)

var messages = []string{} // mutable global storage

func getEnvOrExit(name string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	} else {
		log.Fatalf("Env var %v is missing\n", name)
		return "" // to suppress compiler errs
	}
}

func getConsulKVConfig(cli *consul.Client, name string) string {
	val, _, err := cli.KV().Get(name, nil)
	if err != nil {
		log.Fatal(err)
	}
	if val == nil {
		log.Fatalf("KV %v is missing", name)
		return ""
	}
	return string(val.Value)
}

type appConfig struct {
	port             int
	messageQueueName string
	rabbitURL        string
	serviceID        string
	serviceName      string
	consulAddr       string
}

func newAppConfig() *appConfig {
	cfg := new(appConfig)
	cfg.serviceID = getEnvOrExit("SERVICE_ID")
	cfg.serviceName = getEnvOrExit("SERVICE_NAME")
	if port, err := strconv.Atoi(getEnvOrExit("PORT")); err != nil {
		log.Fatalln(err)
	} else {
		cfg.port = port
	}
	cfg.consulAddr = getEnvOrExit("CONSUL_ADDR")
	return cfg
}

func fillAppConfigWithConsulKV(cfg *appConfig, cli *consul.Client) {
	cfg.rabbitURL = getConsulKVConfig(cli, "rabbit/url")
	cfg.messageQueueName = getConsulKVConfig(cli, "rabbit/queue")
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
	return consulClient
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	cfg := newAppConfig()
	consulClient := initConsul(cfg)
	fillAppConfigWithConsulKV(cfg, consulClient)
	conn := rmqConnect(cfg)
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalln(err)
	}
	go startConsumer(cfg, ch)
	r := mux.NewRouter()
	r.Path("/").HandlerFunc(index)
	r.Path("/health").HandlerFunc(healthCheck)
	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf("0.0.0.0:%v", cfg.port),
	}
	log.Fatal(srv.ListenAndServe())
}

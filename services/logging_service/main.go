package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	consul "github.com/hashicorp/consul/api"
	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/logger"
	log "github.com/sirupsen/logrus"
)

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
	hazelcastAddr        string
	hazelcastClusterName string
	port                 int
	hazelcastMapName     string
	serviceID            string
	serviceName          string
	consulAddr           string
}

func newAppConfig() *appConfig {
	cfg := new(appConfig)
	cfg.serviceID = getEnvOrExit("SERVICE_ID")
	cfg.serviceName = getEnvOrExit("SERVICE_NAME")
	cfg.consulAddr = getEnvOrExit("CONSUL_ADDR")
	if port, err := strconv.Atoi(getEnvOrExit("PORT")); err != nil {
		log.Fatalln(err)
	} else {
		cfg.port = port
	}
	return cfg
}

func fillAppConfigWithConsulKV(cfg *appConfig, cli *consul.Client) {
	cfg.hazelcastAddr = getConsulKVConfig(cli, "hazelcast/addr")
	cfg.hazelcastClusterName = getConsulKVConfig(cli, "hazelcast/cluster")
	cfg.hazelcastMapName = getConsulKVConfig(cli, "hazelcast/map")
}

type addLogRequest struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

type addLogResponse struct {
	Error string `json:"error,omitempty"`
}

type addLogHandler struct {
	client *hazelcast.Client
}

func (l *addLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request addLogRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || len(request.UUID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(addLogResponse{Error: "Bad request"})
		return
	}
	mymap, err := l.client.GetMap("mymap")
	if err != nil {
		log.Fatal(err)
	}
	mymap.Set(request.UUID, request.Message)
	log.Infof("Adding %s:%s", request.UUID, request.Message)

	w.WriteHeader(http.StatusOK)
}

func newAddLogHandler(client *hazelcast.Client) *addLogHandler {
	return &addLogHandler{client: client}
}

type listLogsHandler struct {
	client *hazelcast.Client
	cfg    *appConfig
}

func (l *listLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mymap, err := l.client.GetMap(l.cfg.hazelcastMapName)
	if err != nil {
		log.Fatal(err)
	}

	m := map[string]string{}
	if entries, err := mymap.GetEntrySet(); err == nil {
		for _, e := range entries {
			m[e.Key.(string)] = e.Value.(string)
		}
	} else {
		fmt.Println(err)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

func newListLogsHandler(cfg *appConfig, client *hazelcast.Client) *listLogsHandler {
	return &listLogsHandler{cfg: cfg, client: client}
}

func newHazelcastClient(cfg *appConfig) *hazelcast.Client {
	config := hazelcast.NewConfig()
	config.ClusterConfig.Name = cfg.hazelcastClusterName
	if err := config.ClusterConfig.SetAddress(cfg.hazelcastAddr); err != nil {
		log.Fatal(err)
	}
	// Create client
	fmt.Println("Connecting...")
	config.LoggerConfig.Level = logger.OffLevel
	client, err := hazelcast.StartNewClientWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected!")
	return client
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

func main() {
	cfg := newAppConfig()
	consulClient := initConsul(cfg)
	fillAppConfigWithConsulKV(cfg, consulClient)
	client := newHazelcastClient(cfg)
	defer func() {
		client.Shutdown()
		fmt.Println("Shutdown.")
	}()
	router := mux.NewRouter()
	router.Path("/health").HandlerFunc(healthCheck)
	logRouter := router.PathPrefix("/log").Subrouter()
	addLogH := newAddLogHandler(client)
	logRouter.Path("/add").Handler(addLogH).Methods(http.MethodPost)
	listLogsH := newListLogsHandler(cfg, client)
	logRouter.Path("/list").Handler(listLogsH).Methods(http.MethodGet)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("0.0.0.0:%v", cfg.port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
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

type appConfig struct {
	hazelcastAddr string
	hazelcastClusterName string
	port int
	hazelcastMapName string
}

func newAppConfig() *appConfig {
	cfg := new(appConfig)
	cfg.hazelcastAddr = getEnvOrExit("HAZELCAST_ADDR")
	cfg.hazelcastClusterName = getEnvOrExit("HAZELCAST_CLUSTER_NAME")
	if port, err := strconv.Atoi(getEnvOrExit("PORT")); err != nil {
		log.Fatalln(err)
	} else {
		cfg.port = port
	}
	cfg.hazelcastMapName = getEnvOrExit("HAZELCAST_MAP_NAME")
	return cfg
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
	cfg *appConfig
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

func main() {
	cfg := newAppConfig()
	client := newHazelcastClient(cfg)
	defer func() {
		client.Shutdown()
		fmt.Println("Shutdown.")
	}()
	router := mux.NewRouter()
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

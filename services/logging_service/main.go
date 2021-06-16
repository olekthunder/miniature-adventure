package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/hazelcast/hazelcast-go-client"
	log "github.com/sirupsen/logrus"
)

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
}

func (l *listLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mymap, err := l.client.GetMap("mymap")
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

func newListLogsHandler(client *hazelcast.Client) *listLogsHandler {
	return &listLogsHandler{client: client}
}

func newHazelcastClient() *hazelcast.Client {
	config := hazelcast.NewConfig()
	config.ClusterConfig.Name = "dev"
	if err := config.ClusterConfig.SetAddress(os.Getenv("HAZELCAST_ADDR")); err != nil {
		log.Fatal(err)
	}
	// Create client
	fmt.Println("Connecting...")
	client, err := hazelcast.StartNewClientWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected!")
	return client
}

func main() {
	client := newHazelcastClient()
	defer func() {
		client.Shutdown()
		fmt.Println("Shutdown.")
	}()
	router := mux.NewRouter()
	logRouter := router.PathPrefix("/log").Subrouter()
	addLogH := newAddLogHandler(client)
	logRouter.Path("/add").Handler(addLogH).Methods(http.MethodPost)
	listLogsH := newListLogsHandler(client)
	logRouter.Path("/list").Handler(listLogsH).Methods(http.MethodGet)
	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:8082",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

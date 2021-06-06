package main

import (
	"log"
	"strconv"

	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/logger"
)

var HZ_ADDRS = []string{"hz1", "hz2", "hz3"}

func main() {
	// Create the configuration
	config := hazelcast.NewConfig()
	config.ClusterConfig.Name = "dev"
	config.LoggerConfig.Level = logger.OffLevel
	if err := config.ClusterConfig.SetAddress(HZ_ADDRS...); err != nil {
		log.Fatal(err)
	}
	log.Println("Connecting...")
	client, err := hazelcast.StartNewClientWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected!")
	// Retrieve a map.
	mymap, err := client.GetMap("mymap")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		mymap.Set(strconv.Itoa(i), i)
	}
	// Stop the client once you are done with it.
	client.Shutdown()
	log.Println("Shutdown.")
}

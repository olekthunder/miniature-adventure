package main

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/logger"
)

var HZ_ADDRS = []string{"hz1", "hz2", "hz3"}

type incrFuncType = func(*hazelcast.Map, string, int)

func main() {
	// Create the configuration
	config := hazelcast.NewConfig()
	config.ClusterConfig.Name = "dev"
	config.LoggerConfig.Level = logger.OffLevel
	if err := config.ClusterConfig.SetAddress(HZ_ADDRS...); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connecting...")
	client, err := hazelcast.StartNewClientWithConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected!")
	// Retrieve a map.
	mymap, err := client.GetMap("mymap")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		mymap.Set(strconv.Itoa(i), i)
	}

	runIncrementWorkers := func(mapName string, incr incrFuncType) {
		wg := new(sync.WaitGroup)
		for i := 0; i < 3; i++ {
			go createWorker(wg, config, i, mapName, incr)()
		}
		wg.Wait()
	}

	runIncrementWorkers("pessimistic", incrementWithPessimisticLock)
	printLine()
	runIncrementWorkers("nolock", incrementMapValue)
	printLine()
	runIncrementWorkers("optimistic", incremeptWithOptimisticLock)

	// Stop the client once you are done with it.
	client.Shutdown()
	fmt.Println("Shutdown.")
}

func printLine() {
	fmt.Println(strings.Repeat("-", 40))
}

func incrementMapValue(m *hazelcast.Map, key string, workerIdx int) {
	val, err := m.Get(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	if v, ok := val.(int64); !ok {
		fmt.Printf("%+v\n", v)
		fmt.Println(reflect.TypeOf(val))
		return
	} else {
		v += 1
		fmt.Printf("%v: Setting %v\n", workerIdx, v)
		if err := m.Set(key, v); err != nil {
			fmt.Printf("%v: %+v\n", workerIdx, err)
			return
		}
		fmt.Printf("%v: Set %v\n", workerIdx, v)
		time.Sleep(time.Millisecond * 100)
	}
}

func incrementWithPessimisticLock(m *hazelcast.Map, key string, workerIdx int) {
	defer func() {
		m.Unlock(key)
	}()
	if err := m.Lock(key); err != nil {
		fmt.Println(err)
		return
	}
	incrementMapValue(m, key, workerIdx)
}

func incremeptWithOptimisticLock(m *hazelcast.Map, key string, workerIdx int) {
	for {
		val, err := m.Get(key)
		if err != nil {
			fmt.Println(err)
			return
		}
		if v, ok := val.(int64); !ok {
			fmt.Printf("%+v\n", v)
			fmt.Println(reflect.TypeOf(val))
			return
		} else {
			newVal := v+1
			fmt.Printf("%v: Trying to replace %v with %v\n", workerIdx, v, newVal)
			replaced, err := m.ReplaceIfSame(key, v, newVal)
			if err != nil {
				fmt.Printf("%v: %+v\n", workerIdx, err)
				return
			}
			fmt.Printf("%v: Tried to replace %v with %v, result: %v\n", workerIdx, v, newVal, replaced)
			if replaced {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func createWorker(wg *sync.WaitGroup, config hazelcast.Config, workerIdx int, mapName string, incr incrFuncType) func() {
	wg.Add(1)
	return func() {
		fmt.Printf("Starting %v!\n", mapName)
		defer wg.Done()
		client, err := hazelcast.StartNewClientWithConfig(config)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Connected %v\n", workerIdx)
		pMap, err := client.GetMap(mapName)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%v: Got the map!\n", workerIdx)
		key := "1"
		pMap.PutIfAbsent(key, 0)
		for i := 0; i < 25; i++ {
			incr(pMap, key, workerIdx)
		}
	}
}

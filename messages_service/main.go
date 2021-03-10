package main

import (
	"log"
	"fmt"
	"net/http"
)

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "index")
}

func main() {
	http.HandleFunc("/", index)
	println("running")
	log.Fatal(http.ListenAndServe(":8080", nil))
	println("Stopping")
}
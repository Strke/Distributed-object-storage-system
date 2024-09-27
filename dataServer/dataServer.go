package main

import (
	"go-project/Scalable-distributed-system/dataServer/heartbeat"
	"go-project/Scalable-distributed-system/dataServer/locate"
	"go-project/Scalable-distributed-system/dataServer/objects"
	"log"
	"net/http"
	"os"
)

func main() {
	go heartbeat.StartHeartbeat()
	go locate.StartLocate()
	http.HandleFunc("/objects/", objects.Handler)
	err := http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

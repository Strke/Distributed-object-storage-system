package main

import (
	"go-project/Scalable-distributed-system/ApiServer/heartbeat"
	"go-project/Scalable-distributed-system/ApiServer/locate"
	"go-project/Scalable-distributed-system/ApiServer/objects"
	"go-project/Scalable-distributed-system/ApiServer/temp"
	"go-project/Scalable-distributed-system/ApiServer/versions"
	"log"
	"net/http"
	"os"
)

func main() {
	go heartbeat.ListenHeartbeat()
	http.HandleFunc("/objects/", objects.Handler)
	http.HandleFunc("/temp/", temp.Handler)
	http.HandleFunc("/locate/", locate.Handler)
	http.HandleFunc("/versions/", versions.Handler)
	err := http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

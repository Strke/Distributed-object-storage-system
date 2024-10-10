package main

import (
	"go-project/Scalable-distributed-system/dataServer/heartbeat"
	"go-project/Scalable-distributed-system/dataServer/locate"
	"go-project/Scalable-distributed-system/dataServer/objects"
	"go-project/Scalable-distributed-system/dataServer/temp"
	"log"
	"net/http"
	"os"
)

func main() {
	locate.CollectObjects() //将所有对象的存储位置读入内存，减少磁盘访问次数
	go heartbeat.StartHeartbeat()
	go locate.StartLocate()
	http.HandleFunc("/objects/", objects.Handler)
	http.HandleFunc("/temp/", temp.Handler)
	err := http.ListenAndServe(os.Getenv("LISTEN_ADDRESS"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

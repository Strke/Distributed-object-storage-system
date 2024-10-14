package locate

import (
	"encoding/json"
	"fmt"
	"go-project/Scalable-distributed-system/rabbitmq"
	"go-project/Scalable-distributed-system/rs"
	"go-project/Scalable-distributed-system/types"
	"os"
	"time"
)

func Locate(name string) (locateInfo map[int]string) {
	q := rabbitmq.New(os.Getenv("RABBITMQ_SERVER"))
	q.Publish("dataServers", name)
	c := q.Consume()
	go func() {
		time.Sleep(time.Second)
		q.Close()
	}()
	locateInfo = make(map[int]string)
	for i := 0; i < rs.ALL_SHARDS; i++ {
		msg := <-c
		if len(msg.Body) == 0 {
			return
		}
		var info types.LocateMessage
		json.Unmarshal(msg.Body, &info)
		locateInfo[info.Id] = info.Addr
	}
	return
}

func Exist(name string) bool {
	names := Locate(name)
	fmt.Println("names :", names)
	fmt.Println("names Size:", len(names))
	return len(Locate(name)) >= rs.DATA_SHARDS
}

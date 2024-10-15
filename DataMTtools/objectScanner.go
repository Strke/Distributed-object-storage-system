package DataMTtools

import (
	"go-project/Scalable-distributed-system/ApiServer/objects"
	"go-project/Scalable-distributed-system/ApiServer/utils"
	"go-project/Scalable-distributed-system/es"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	files, _ := filepath.Glob(os.Getenv("STORAGE_ROOT") + "/objects/*")
	for i := range files {
		hash := strings.Split(filepath.Base(files[i]), ".")[0]
		verify(hash)
	}
}

func verify(hash string) {
	log.Println("verify", hash)
	size, e := es.SearchHashSize(hash)
	if e != nil {
		log.Println(e)
		return
	}
	stream, e := objects.GetStream(hash, size)
	if e != nil {
		log.Print(e)
		return
	}
	d := utils.CalculateHash(stream)
	if d != hash {
		log.Print("object hash mismatch, calculated=%s, requested=%s", d, hash)
	}
	stream.Close()
}

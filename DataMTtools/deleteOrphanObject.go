package DataMTtools

import (
	"go-project/Scalable-distributed-system/es"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	files, _ := filepath.Glob(os.Getenv("STORAGE_ROOT") + "/object/*")

	for i := range files {
		hash := strings.Split(filepath.Base(files[i]), ".")[0]
		hashInMetadata, e := es.HasHash(hash)
		if e != nil {
			log.Print(e)
			return
		}
		if !hashInMetadata {
			del(hash)
		}
	}
}

func del(hash string) {
	log.Println("delete", hash)
	url := "http://" + os.Getenv("LISTEN_ADDRESS") + "/objects/" + hash
	request, _ := http.NewRequest("DELETE", url, nil)
	client := http.Client{}
	client.Do(request)
}

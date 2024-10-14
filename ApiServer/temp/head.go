package temp

import (
	"fmt"
	"go-project/Scalable-distributed-system/rs"
	"log"
	"net/http"
	"strings"
)

func head(w http.ResponseWriter, r *http.Request) {
	token := strings.Split(r.URL.EscapedPath(), "/")[2]
	stream, e := rs.NewRSResumablePutStreamFromToken(token)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	current := stream.CurrentSize()
	fmt.Println("start get size")
	if current == -1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Println("current:", current)
	w.Header().Set("content-length", fmt.Sprintf("%d", current))
}

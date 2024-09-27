package objects

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodGet {
		Get(w, r)
		return
	}
	if m == http.MethodPut {
		Put(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func Put(w http.ResponseWriter, r *http.Request) {
	file := os.Getenv("STORAGE_ROOT") + "/objects/" +
		strings.Split(r.URL.EscapedPath(), "/")[2]
	f, err := os.Create(file)
	if err != nil {
		log.Println(err, file)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	io.Copy(f, r.Body)
}

func Get(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(os.Getenv("STORAGE_ROOT") + "/objects/" +
		strings.Split(r.URL.EscapedPath(), "/")[2])
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

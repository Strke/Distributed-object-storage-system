package objects

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"go-project/Scalable-distributed-system/dataServer/locate"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//func Get(w http.ResponseWriter, r *http.Request) {
//	f, err := os.Open(os.Getenv("STORAGE_ROOT") + "/objects/" +
//		strings.Split(r.URL.EscapedPath(), "/")[2])
//	if err != nil {
//		log.Println(err)
//		w.WriteHeader(http.StatusNotFound)
//		return
//	}
//	defer f.Close()
//	io.Copy(w, f)
//}

func Get(w http.ResponseWriter, r *http.Request) {
	file := getFile(strings.Split(r.URL.EscapedPath(), "/")[2])
	if file == "" {

		w.WriteHeader(http.StatusNotFound)
		return
	}
	sendFile(w, file)
}

func getFile(name string) string {
	files, _ := filepath.Glob(os.Getenv("STORAGE_ROOT") + "/objects/" + name + ".*")
	if len(files) != 1 {
		return ""
	}
	file := files[0]
	h := sha256.New()
	sendFile(h, file)
	d := url.PathEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	hash := strings.Split(file, ".")[2]
	if d != hash {
		log.Println("object hash mismatch, remove", file)
		locate.Del(hash)
		os.Remove(file)
		return ""
	}
	return file
}

func sendFile(w io.Writer, file string) {
	f, e := os.Open(file)
	if e != nil {
		log.Print(e)
		return
	}
	defer f.Close()
	gzipStream, e := gzip.NewReader(f)
	if e != nil {
		log.Print(e)
		return
	}
	io.Copy(w, gzipStream)
	gzipStream.Close()
}

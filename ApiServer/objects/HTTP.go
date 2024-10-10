package objects

import (
	"fmt"
	"go-project/Scalable-distributed-system/ApiServer/heartbeat"
	"go-project/Scalable-distributed-system/ApiServer/locate"
	"go-project/Scalable-distributed-system/ApiServer/utils"
	"go-project/Scalable-distributed-system/objectstream"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodGet {
		get(w, r)
		return
	}
	if m == http.MethodPut {
		put(w, r)
		return
	}
	if m == http.MethodDelete {
		del(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

//func put(w http.ResponseWriter, r *http.Request) {
//	object := strings.Split(r.URL.EscapedPath(), "/")[2]
//	c, err := storeObject(r.Body, object)
//	if err != nil {
//		log.Println(err)
//	}
//	w.WriteHeader(c)
//}

func put(w http.ResponseWriter, r *http.Request) {
	hash := utils.GetHashFromHeader(r.Header)
	if hash == "" {
		log.Println("missing object hash in digest header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c, e := storeObject(r.Body, url.PathEscape(hash))
	if e != nil {
		log.Println(e)
		w.WriteHeader(c)
		return
	}
	if c != http.StatusOK {
		w.WriteHeader(c)
		return
	}

	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	size := utils.GetSizeFromHeader(r.Header)
	e = es.AddVersion(name, hash, size)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func storeObject(r io.Reader, object string) (int, error) {
	stream, err := putStream(object)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	io.Copy(stream, r)
	err = stream.Close()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
func putStream(object string) (*objectstream.PutStream, error) {
	server := heartbeat.ChooseRandomDataServer()
	if server == "" {
		return nil, fmt.Errorf("cannot find any dataServer")
	}
	return objectstream.NewPutStream(server, object), nil
}

//func get(w http.ResponseWriter, r *http.Request) {
//	object := strings.Split(r.URL.EscapedPath(), "/")[2]
//	stream, e := getStream(object)
//	if e != nil {
//		log.Println(e)
//		w.WriteHeader(http.StatusNotFound)
//		return
//	}
//	io.Copy(w, stream)
//}

func get(w http.ResponseWriter, r *http.Request) {
	name := strings.Split(r.URL.EscapedPath(), "/")
	versionId := r.URL.Query()["version"]
	version := 0
	var e error
	if len(versionId) != 0 {
		version, e = strconv.Atoi(versionId[0])
		if e != nil {
			log.Println(e)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	meta, e := es.GetMetadata(name, version)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if meta.Hash == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	object := url.PathEscape(meta.Hash)
	stream, e := getStream(object)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	io.Copy(w, stream)
}

func getStream(object string) (io.Reader, error) {
	server := locate.Locate(object)
	if server == "" {
		return nil, fmt.Errorf("oboject %s locate fall!", object)
	}
	return objectstream.NewGetStream(server, object)
}

func del(w http.ResponseWriter, r *http.Request) {
	name := strings.Split(r.URL.EscapedPath(), "/")[2]
	version, e := es.SearchAllVersion(name)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	e = es.PutMetadata(name, version.Version+1, 0, "")
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

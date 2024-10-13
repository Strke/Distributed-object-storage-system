package objects

import (
	"fmt"
	"go-project/Scalable-distributed-system/ApiServer/heartbeat"
	"go-project/Scalable-distributed-system/ApiServer/locate"
	"go-project/Scalable-distributed-system/ApiServer/utils"
	"go-project/Scalable-distributed-system/es"
	"go-project/Scalable-distributed-system/rs"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func put(w http.ResponseWriter, r *http.Request) {
	hash := utils.GetHashFromHeader(r.Header)
	if hash == "" {
		log.Println("missing object hash in digest header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	size := utils.GetSizeFromHeader(r.Header)
	fmt.Println(size)
	c, e := storeObject(r.Body, hash, size)
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
	e = es.AddVersion(name, hash, size)
	if e != nil {
		log.Println(e)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func storeObject(r io.Reader, hash string, size int64) (int, error) {
	fmt.Println("Start store")
	if locate.Exist(url.PathEscape(hash)) {
		fmt.Println("hash already exist. don`t store again")
		return http.StatusOK, nil
	}
	stream, err := putStream(url.PathEscape(hash), size)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}
	reader := io.TeeReader(r, stream)
	d := utils.CalculateHash(reader)
	if d != hash {
		stream.Commit(false)
		return http.StatusBadRequest, fmt.Errorf("object hash mismatch, calculated=%s, request=%s", d, hash)
	}
	fmt.Println("start commit")
	stream.Commit(true)
	return http.StatusOK, nil
}

func putStream(hash string, size int64) (*rs.RSPutStream, error) {
	servers := heartbeat.ChooseRandomDataServers(rs.ALL_SHARDS, nil)
	if len(servers) != rs.ALL_SHARDS {
		return nil, fmt.Errorf("cannot find enough dataServer")
	}
	return rs.NewRSPutStream(servers, hash, size)
}

package es

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Metadata struct {
	Name    string
	Version int
	Size    int64
	Hash    string
}

func getMetadata(name string, versionId int) (meta Metadata, e error) {
	url := fmt.Sprint("http://%s/metadata/objects/%s_%d/_source",
		os.Getenv("ES_SERVER"), name, versionId)
	r, e := http.Get(url)
	if e != nil {
		return
	}
	if r.StatusCode != http.StatusOK {
		e = fmt.Errorf("fail to get %s_%d: %d", name, versionId, r.StatusCode)
		return
	}
	result, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(result, &meta)
	return
}

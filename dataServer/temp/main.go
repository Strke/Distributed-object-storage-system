package temp

import (
	"encoding/json"
	"fmt"
	"go-project/Scalable-distributed-system/dataServer/locate"
	"go-project/Scalable-distributed-system/dataServer/objects"
	"io/ioutil"
	"net/http"
	"os"
)

type tempInfo struct {
	Uuid string
	Name string
	Size int64
}

func (t *tempInfo) writeToFile() error {
	f, e := os.Create(os.Getenv("STORAGE_ROOT") + "/temp/" + t.Uuid)
	if e != nil {
		return e
	}
	defer f.Close()
	b, _ := json.Marshal(t)
	f.Write(b)
	return nil
}
func readFromFile(uuid string) (*tempInfo, error) {
	f, e := os.Open(os.Getenv("STORAGE_ROOT") + "/temp/" + uuid)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	var info tempInfo
	json.Unmarshal(b, &info)
	return &info, nil
}

func commitTempObject(datFile string, tempinfo *tempInfo) {
	fmt.Println(datFile)
	fmt.Println(os.Getenv("STORAGE_ROOT") + "/objects/" + tempinfo.Name)
	os.Rename(datFile, os.Getenv("STORAGE_ROOT")+"/objects/"+tempinfo.Name)
	fmt.Println("rename success, prepare to add")
	locate.Add(tempinfo.Name)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodPut {
		put(w, r)
		return
	}
	if m == http.MethodGet {
		objects.Get(w, r)
		return
	}
	if m == http.MethodPost {
		post(w, r)
		return
	}
	if m == http.MethodPatch {
		patch(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

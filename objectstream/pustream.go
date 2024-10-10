package objectstream

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type PutStream struct {
	writer *io.PipeWriter
	c      chan error
}

func NewPutStream(server, object string) *PutStream {
	reader, writer := io.Pipe()
	c := make(chan error)
	go func() {
		request, _ := http.NewRequest("PUT", "http://"+server+"/objects/"+object, reader)
		fmt.Println("http://" + server + os.Getenv("STORAGE_ROOT") + "/objects/" + object)
		client := http.Client{}
		r, e := client.Do(request)
		if e == nil && r.StatusCode != http.StatusOK {
			fmt.Println(r)
			e = fmt.Errorf("dataServer return http code %d", r.StatusCode)
		}
		c <- e
	}()
	return &PutStream{writer, c}
}

func (w *PutStream) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}
func (w *PutStream) Close() error {
	w.writer.Close()
	return <-w.c
}

type TempPutStream struct {
	Server string
	Uuid   string
}

func NewTempPutStream(server, hash string, size int64) (*TempPutStream, error) {
	request, e := http.NewRequest("POST", "http://"+server+"/temp/"+hash, nil)
	if e != nil {
		return nil, e
	}
	request.Header.Set("size", fmt.Sprintf("%d", size))
	client := http.Client{}
	response, e := client.Do(request)
	if e != nil {
		return nil, e
	}
	uuid, e := ioutil.ReadAll(response.Body)
	if e != nil {
		return nil, e
	}
	return &TempPutStream{server, string(uuid)}, nil
}

func (w *TempPutStream) Write(p []byte) (n int, err error) {
	request, e := http.NewRequest("PATCH", "http://"+w.Server+"/temp/"+w.Uuid, strings.NewReader(string(p)))
	if e != nil {
		return 0, e
	}
	client := http.Client{}
	r, e := client.Do(request)
	if e != nil {
		return 0, e
	}
	if r.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("dataserver return http code %d", r.StatusCode)
	}
	return len(p), nil
}

func (w *TempPutStream) Commit(good bool) {
	method := "DELETE"
	if good {
		method = "PUT"
	}
	request, _ := http.NewRequest(method, "http://"+w.Server+"/temp/"+w.Uuid, nil)
	client := http.Client{}
	client.Do(request)
}

package objects

import (
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	m := r.Method
	if m == http.MethodGet {
		Get(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

//func Put(w http.ResponseWriter, r *http.Request) {
//	file := os.Getenv("STORAGE_ROOT") + "/objects/" +
//		strings.Split(r.URL.EscapedPath(), "/")[2]
//	f, err := os.Create(file)
//	if err != nil {
//		log.Println(err, file)
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	defer f.Close()
//	io.Copy(f, r.Body)
//}

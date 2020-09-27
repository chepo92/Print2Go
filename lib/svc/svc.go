package svc

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/tarm/serial"
	"gitlab.com/adrian_blx/gfeeder/lib/task"
	"os"
)

type Svc struct {
	sync.RWMutex
	srv        *http.Server
	storage    FileStorage
	task       *task.Task
	serialPort func() (*serial.Port, error)
}

type FileStorage interface {
	UploadFile(path, filename string, r io.ReadCloser) error
	ReadFile(path, filename string) (*os.File, error)
}

func New(srv *http.Server, store FileStorage, serial func() (*serial.Port, error)) *Svc {
	svc := &Svc{
		srv:        srv,
		storage:    store,
		serialPort: serial,
	}
	mux := http.NewServeMux()
	mux.Handle("/", svc)
	svc.srv.Handler = mux
	return svc
}

func (svc *Svc) Run() error {
	return svc.srv.ListenAndServe()
}

func (svc *Svc) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	if rq.URL.Path == "/api/version" {
		svc.versionReply(w, rq)
		return
	}
	if rq.URL.Path == "/api/files/local" && rq.Method == "POST" {
		svc.localUpload(w, rq)
		return
	}
	if rq.URL.Path == "/api/job" {
		svc.jobStatus(w, rq)
		return
	}
	if rq.URL.Path == "/" {
		svc.indexPage(w)
		return
	}
	http.Error(w, "unknown url", 404)
}

func jsonWrite(w http.ResponseWriter, msg interface{}) {
	pl, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, "marshalling error", 503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(pl)
}

func (svc *Svc) versionReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		API     string `json:"api"`
		Version string `json:"server"`
		Banner  string `json:"text"`
	}{
		API:     "0.1",
		Version: "0.20200918",
		Banner:  "OctoPrint compatible gfeeder api",
	}
	jsonWrite(w, reply)
}

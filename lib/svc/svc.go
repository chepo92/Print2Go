package svc

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"gitlab.com/adrian_blx/takoprint/lib/camera"
	"gitlab.com/adrian_blx/takoprint/lib/store"
	"gitlab.com/adrian_blx/takoprint/lib/task"
)

type Svc struct {
	sync.RWMutex
	srv        *http.Server
	storage    FileStorage
	camera     *camera.Camera
	task       *task.Task
	serialPort func() (io.ReadWriteCloser, error)
	// function we execute if hardware should be shut down.
	shutdown func()
}

type FileStorage interface {
	UploadFile(path, filename string, r io.ReadCloser) error
	ReadFile(path, filename string) (store.Stream, error)
}

func New(srv *http.Server, store FileStorage, serial func() (io.ReadWriteCloser, error), shutdown func()) *Svc {
	svc := &Svc{
		srv:        srv,
		storage:    store,
		serialPort: serial,
		shutdown:   shutdown,
		camera:     camera.New(),
		task:       task.New(),
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
	switch rq.URL.Path {
	case "/api/version":
		svc.versionReply(w, rq)
	case "/api/files/local":
		svc.localUpload(w, rq)
	case "/api/job/status":
		svc.jobStatus(w, rq)
	case "/api/job/cancel":
		svc.jobCancel(w)
	case "/api/gcode/action":
		svc.enqueueBuiltin(w, rq)
	case "/api/device/shutdown":
		svc.shutdown()
	case "/camera":
		svc.cameraPage(w)
	case "/camera/stream.mjpeg":
		svc.camera.WriteStream(w)
	case "/":
		svc.indexPage(w)
	default:
		http.Error(w, "unknown url", 404)
	}
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
		Banner:  "OctoPrint compatible takoprint api",
	}
	jsonWrite(w, reply)
}

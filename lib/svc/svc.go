package svc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"git.sr.ht/~adrian-blx/takoprint/lib/camera"
	"git.sr.ht/~adrian-blx/takoprint/lib/store"
	"git.sr.ht/~adrian-blx/takoprint/lib/task"
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
	case "/api/settings":
		svc.settingsReply(w)
	case "/api/login":
		svc.fakeLoginReply(w)
	case "/api/printer":
		svc.fakePrinterReply(w)
	case "/api/files/local":
		svc.localUpload(w, rq)
	case "/api/job":
		svc.apiJobStatus(w)
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
		fmt.Printf("Unknown URL: %s\n", rq.URL.Path)
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

func (svc *Svc) settingsReply(w http.ResponseWriter) {
	reply := struct {
		Appearance struct {
			Name string `json:"name"`
		} `json:"appearance"`
	}{
		Appearance: struct {
			Name string `json:"name"`
		}{

			Name: "takoprint",
		},
	}
	jsonWrite(w, reply)
}

func (svc *Svc) fakeLoginReply(w http.ResponseWriter) {
	reply := struct {
		Ext     bool   `json:"_is_external_client"`
		Session string `json:"session"`
	}{
		Session: "abadapi",
	}
	jsonWrite(w, reply)
}

type fakeFlags struct {
	Operational bool `json:"operational"`
	Ready       bool `json:"ready"`
	Error       bool `json:"error"`
	CoErr       bool `json:"closedOrError"`
	Pausing     bool `json:"pausing"`
	Paused      bool `json:"paused"`
	Printing    bool `json:"printing"`
	Cancelling   bool `json:"cancelling"`
}

func (svc *Svc) fakePrinterReply(w http.ResponseWriter) {
	reply := struct {
		State struct {
			Text  string    `json:"text"`
			Flags fakeFlags `json:"flags"`
		} `json:"state"`
	}{
		State: struct {
			Text  string    `json:"text"`
			Flags fakeFlags `json:"flags"`
		}{
			Text: "operational",
			Flags: fakeFlags{
				Operational: true,
				Ready:       true,
			},
		},
	}
	jsonWrite(w, reply)
}

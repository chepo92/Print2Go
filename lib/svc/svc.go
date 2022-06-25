package svc

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"git.sr.ht/~adrian-blx/takoprint/lib/camera"
	"git.sr.ht/~adrian-blx/takoprint/lib/store"
	"git.sr.ht/~adrian-blx/takoprint/lib/task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Svc struct {
	sync.RWMutex
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

func New(store FileStorage, serial func() (io.ReadWriteCloser, error), shutdown func()) *Svc {
	svc := &Svc{
		storage:    store,
		serialPort: serial,
		shutdown:   shutdown,
		camera:     camera.New(),
		task:       task.New(),
	}
	return svc
}

func (svc *Svc) Run(srv *http.Server) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Get("/", indexPage)
	// Takoprint api
	r.Get("/api/job/status", svc.jobStatus)
	r.Post("/api/job/cancel", svc.jobCancel)
	r.Post("/api/device/shutdown", svc.apiShutdown)
	r.Post("/api/files/local", svc.localUpload)
	r.Get("/api/gcode/action", svc.enqueueBuiltin)

	// Octoprint fake-compatibility
	r.Get("/api/version", svc.versionReply)
	r.Get("/api/settings", svc.settingsReply)
	r.Post("/api/login", svc.fakeLoginReply)
	r.Get("/api/printer", svc.fakePrinterReply)
	r.Get("/api/job", svc.apiJobStatus)

	// Camera support
	r.Get("/camera", svc.cameraPage)
	r.Get("/camera/stream.mjpeg", svc.camera.WriteStream)

	srv.Handler = r
	return srv.ListenAndServe()
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

func (svc *Svc) settingsReply(w http.ResponseWriter, r *http.Request) {
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

func (svc *Svc) fakeLoginReply(w http.ResponseWriter, r *http.Request) {
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
	Cancelling  bool `json:"cancelling"`
}

func (svc *Svc) fakePrinterReply(w http.ResponseWriter, r *http.Request) {
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

func (svc *Svc) apiShutdown(w http.ResponseWriter, r *http.Request) {
	svc.shutdown()
	jsonWrite(w, nil)
}

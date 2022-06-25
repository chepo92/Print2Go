package webapi

import (
	"encoding/json"
	"io"
	"net/http"

	"git.sr.ht/~adrian-blx/takoprint/camera"
	"git.sr.ht/~adrian-blx/takoprint/store"
	"git.sr.ht/~adrian-blx/takoprint/task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type WebApi struct {
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

func New(store FileStorage, serial func() (io.ReadWriteCloser, error), shutdown func()) *WebApi {
	wapi := &WebApi{
		storage:    store,
		serialPort: serial,
		shutdown:   shutdown,
		camera:     camera.New(),
		task:       task.New(),
	}
	return wapi
}

func (wapi *WebApi) Run(srv *http.Server) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Get("/", indexPage)
	// Takoprint api
	r.Get("/api/job/status", wapi.jobStatus)
	r.Post("/api/job/cancel", wapi.jobCancel)
	r.Post("/api/device/shutdown", wapi.apiShutdown)
	r.Post("/api/files/local", wapi.localUpload)
	r.Get("/api/gcode/action", wapi.enqueueBuiltin)

	// Octoprint fake-compatibility
	r.Get("/api/version", fakeVersionReply)
	r.Get("/api/settings", settingsReply)
	r.Post("/api/login", fakeLoginReply)
	r.Get("/api/printer", fakePrinterReply)
	r.Get("/api/job", wapi.apiJobStatus)

	// Camera support
	r.Get("/camera", cameraPage)
	r.Get("/camera/stream.mjpeg", wapi.camera.WriteStream)

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

func fakeVersionReply(w http.ResponseWriter, rq *http.Request) {
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

func settingsReply(w http.ResponseWriter, r *http.Request) {
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

func fakeLoginReply(w http.ResponseWriter, r *http.Request) {
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

func fakePrinterReply(w http.ResponseWriter, r *http.Request) {
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

func (wapi *WebApi) apiShutdown(w http.ResponseWriter, r *http.Request) {
	wapi.shutdown()
	jsonWrite(w, nil)
}

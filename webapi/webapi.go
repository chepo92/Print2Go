package webapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/chepo92/PrintAndGo/store"
	"github.com/chepo92/PrintAndGo/task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// WebApi is the macro structure of PrintAndGo, implements and manages the server and web API, handling storage, print tasks, serial port and camera.
type WebApi struct {
	storage FileStorage
	// camera     *camera.Camera // camera disabled for windows build
	task       *task.Task
	serialPort func() (io.ReadWriteCloser, error)
	motdFile   string
	// function we execute if hardware should be shut down.
	shutdown func()
}

type FileStorage interface {
	UploadFile(path, filename string, r io.ReadCloser) error
	ReadFile(path, filename string) (store.Stream, error)
}

// New creates a new WebApi instance. Starts a new task and returns the instance.
func New(store FileStorage, motdFile string, serial func() (io.ReadWriteCloser, error), shutdown func()) *WebApi {
	wapi := &WebApi{
		storage:    store,
		serialPort: serial,
		motdFile:   motdFile,
		shutdown:   shutdown,
		//camera:     camera.New(camdev), // camera disabled for windows build
		task: task.New(),
	}
	return wapi
}

// Run starts the web server and binds the routes.
// returns the http server handle
func (wapi *WebApi) Run(srv *http.Server) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Get("/", indexPage)
	// PrintAndGo api
	r.Get("/printandgo/job/status", wapi.pagJobStatus)
	r.Post("/printandgo/job/create", wapi.localUpload)
	r.Post("/printandgo/job/cancel", wapi.pagJobCancel)
	r.Post("/printandgo/device/shutdown", wapi.pagShutdown)
	r.Get("/printandgo/gcode/action", wapi.pagEnqueueBuiltin)

	// Octoprint fake-compatibility
	r.Get("/api/version", octoVersionReply)
	r.Get("/api/settings", octoSettingsReply)
	r.Post("/api/login", octoLoginReply)
	r.Get("/api/printer", octoPrinterReplyFake)
	r.Post("/api/printer/command", wapi.octoPrinterCommand)
	r.Get("/api/job", wapi.octoJobStatus)
	r.Post("/api/job", wapi.octoModifyJob)
	r.Post("/api/files/local", wapi.localUpload)

	// Camera support
	//r.Get("/camera", cameraPage) // camera disabled for windows build
	//r.Get("/camera/stream.mjpeg", wapi.camera.WriteStream) // camera disabled for windows build

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

func (wapi *WebApi) pagShutdown(w http.ResponseWriter, r *http.Request) {
	wapi.shutdown()
	jsonWrite(w, nil)
}

package webapi

import (
	"encoding/json"
	"io"
	"net/http"

	"git.sr.ht/~adrian-blx/takoprint/store"
	"git.sr.ht/~adrian-blx/takoprint/task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

func (wapi *WebApi) Run(srv *http.Server) error {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Get("/", indexPage)
	// Takoprint api
	r.Get("/takoprint/job/status", wapi.takoJobStatus)
	r.Post("/takoprint/job/create", wapi.localUpload)
	r.Post("/takoprint/job/cancel", wapi.takoJobCancel)
	r.Post("/takoprint/device/shutdown", wapi.takoShutdown)
	r.Get("/takoprint/gcode/action", wapi.takoEnqueueBuiltin)

	// Octoprint fake-compatibility
	r.Get("/api/version", octoVersionReply)
	r.Get("/api/settings", octoSettingsReply)
	r.Post("/api/login", octoLoginReply)
	r.Get("/api/printer", octoPrinterReplyFake)
	r.Post("/api/printer/command", octoPrinterCommand)
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

func (wapi *WebApi) takoShutdown(w http.ResponseWriter, r *http.Request) {
	wapi.shutdown()
	jsonWrite(w, nil)
}

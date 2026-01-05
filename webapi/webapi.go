package webapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/chepo92/PrintAndGo/serial"
	"github.com/chepo92/PrintAndGo/serial/serialmgr"
	"github.com/chepo92/PrintAndGo/store"
	"github.com/chepo92/PrintAndGo/task"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// WebApi is the macro structure of PrintAndGo, implements and manages the server and web API, handling storage, print tasks, serial port and camera.
type WebApi struct {
	storage FileStorage
	// camera     *camera.Camera // camera disabled for windows build
	task   *task.Task
	serial *serialmgr.SerialManager

	// SerialPortInfo struct {
	// 	Port     string
	// 	BaudRate int
	// }
	// motd file path
	motdFile string
	// function we execute if hardware should be shut down.
	shutdown func()
	Version  string
}

type FileStorage interface {
	UploadFile(path, filename string, r io.ReadCloser) error
	ReadFile(path, filename string) (store.Stream, error)
}

// New creates a new WebApi instance. Starts a new task and returns the instance.
func New(store FileStorage, motdFile string, openSerial func(serial.SerialConfig) (io.ReadWriteCloser, error), shutdown func()) *WebApi {

	wapi := &WebApi{
		storage:  store,
		serial:   serialmgr.New(openSerial),
		motdFile: motdFile,
		shutdown: shutdown,
		//camera:     camera.New(camdev), // camera disabled for windows build
		task: task.New(),
	}
	return wapi
}

// SetPortBaudRate sets the current port and baud rate
// This is used to update the UI.
// func (wapi *WebApi) SetPortBaudRate(port string, baud int) {
// 	wapi.serial.SetPort(port)
// 	wapi.serial.SetBaudRate(baud)
// }

// SetVersion sets the current version of PrintAndGo
func (wapi *WebApi) SetVersion(v string) {
	wapi.Version = v
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
	r.Post("/api/login", octoLoginReply)
	r.Get("/api/version", octoVersionReply)
	r.Get("/api/server", octoServerReply)                   // To be implemented
	r.Get("/api/connection", wapi.octoGetConnectionReply)   // To be implemented
	r.Post("/api/connection", wapi.octoPostConnectionReply) // To be implemented
	r.Post("/api/files/local", wapi.localUpload)
	r.Get("/api/job", wapi.octoJobStatus)
	r.Post("/api/job", wapi.octoModifyJob)
	r.Get("/api/languages", octoLanguagesReply) // To be implemented
	r.Get("/api/printer", octoPrinterReplyFake)
	r.Post("/api/printer/command", wapi.octoPrinterCommand)
	r.Get("/api/printerprofiles", octoPrinterProfilesReply) // To be implemented
	r.Get("/api/settings", octoSettingsReply)
	r.Get("/api/slicing", octoSlicingReply)                // To be implemented
	r.Get("/api/system/commands", octoSystemCommandsReply) // To be implemented
	r.Get("/api/timelapse", octoTimelapseReply)            // To be implemented
	r.Get("/api/access", octoAccessReply)                  // To be implemented
	r.Get("/api/util/test", octoUtilTestReply)             // To be implemented
	r.Get("/setup/wizard", octoSetupWizardReply)           // To be implemented

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

// func (wapi *WebApi) getSerial() (io.ReadWriteCloser, error) {
// 	var err error
// 	wapi.serialOnce.Do(func() {
// 		wapi.serial, err = wapi.serialPort()
// 	})
// 	return wapi.serial, err
// }

// func (wapi *WebApi) Close() {
// 	if wapi.serial != nil {
// 		wapi.serial.Close()
// 	}
// }

func octoStateFromSerial(s serialmgr.SerialState) string {
	switch s {
	case serialmgr.Disconnected:
		return "Closed"
	case serialmgr.Connecting:
		return "Connecting"
	case serialmgr.Connected:
		return "Operational"
	case serialmgr.Error:
		return "Error"
	default:
		return "Unknown"
	}
}

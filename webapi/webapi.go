package webapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/chepo92/Print2Go/serial"
	"github.com/chepo92/Print2Go/serial/serialmgr"
	store "github.com/chepo92/Print2Go/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	hwserial "go.bug.st/serial"

	"github.com/chepo92/Print2Go/job"
)

// WebApi is the macro structure of Print2Go, implements and manages the server and web API, handling storage, print tasks, serial port and camera.
type WebApi struct {
	storage FileStorage
	// camera     *camera.Camera // camera disabled for windows build

	serial     *serialmgr.SerialManager
	jobManager *job.JobManager

	selectedFile string

	// motd file path
	motdFile string
	// function we execute if hardware should be shut down.
	shutdown func()
	Version  string

	listenAddr string
}

type GcodeCommandRequest struct {
	Command string `json:"command"`
}

type FileStorage interface {
	UploadFile(path, filename string, r io.ReadCloser) error
	ReadFile(path, filename string) (store.Stream, error)
}

// New creates a new WebApi instance. Starts a new task and returns the instance.
func New(listenAddr string, store FileStorage, motdFile string, openSerial func(serial.SerialConfig) (hwserial.Port, error), shutdown func()) *WebApi {

	serialMgr := serialmgr.New(openSerial)

	wapi := &WebApi{
		listenAddr: listenAddr,
		storage:    store,
		serial:     serialMgr,
		motdFile:   motdFile,
		shutdown:   shutdown,
		jobManager: job.New(serialMgr),
		//camera:     camera.New(camdev), // camera disabled for windows build
		//task: task.New(),
	}

	wapi.initAutoConnect()

	wapi.serial.OnConnected = func() {
		go wapi.sendInstanceM117()
	}

	return wapi
}

func (wapi *WebApi) sendInstanceM117() {
	host, port, err := net.SplitHostPort(wapi.listenAddr)
	if err != nil {
		fmt.Printf("[WebAPI] Invalid listenAddr: %v\n", err)
		return
	}

	ip := resolveDisplayIP(host)
	time.Sleep(5 * time.Second)
	cmd := fmt.Sprintf("M117 %s:%s", ip, port)

	fmt.Printf("[WebAPI] Displaying instance on printer: %s\n", cmd)
	if err := wapi.serial.SendGcodePriority(cmd, true); err != nil {
		fmt.Printf("[WebAPI] Failed to send M117: %v\n", err)
	}
}

func resolveDisplayIP(host string) string {
	if host != "" && host != "0.0.0.0" && host != "::" {
		return host
	}

	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if v4 := ipnet.IP.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return "localhost"
}

// SetVersion sets the current version of Print2Go
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
	// Print2Go api
	r.Get("/print2go/job/status", wapi.pagJobStatus)
	r.Post("/print2go/job/create", wapi.localUpload)
	r.Post("/print2go/job/cancel", wapi.pagJobCancel)
	r.Post("/print2go/device/shutdown", wapi.pagShutdown)
	r.Get("/print2go/gcode/action", wapi.pagEnqueueBuiltin)

	// Octoprint-compatibility
	r.Post("/api/login", octoLoginReply)
	r.Get("/api/version", octoVersionReply)
	r.Get("/api/server", octoServerReply) // To be implemented
	r.Get("/api/connection", wapi.octoGetConnectionReply)
	r.Post("/api/connection", wapi.octoPostConnectionReply)
	r.Post("/api/files/local", wapi.localUpload)
	r.Get("/api/job", wapi.octoGetJobStatus)
	r.Post("/api/job", wapi.octoPostJob)
	r.Get("/api/languages", octoLanguagesReply) // To be implemented
	r.Get("/api/printer", wapi.octoPrinterReply)
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

func (wapi *WebApi) initAutoConnect() {
	go func() {
		time.Sleep(5 * time.Second) // dar tiempo a detectar puertos
		req := map[string]any{
			"command":     "connect",
			"port":        "AUTO",
			"baudrate":    115200,
			"autoconnect": true,
		}
		buf, _ := json.Marshal(req)
		http.Post("http://localhost/api/connection", "application/json", bytes.NewReader(buf))
	}()
}

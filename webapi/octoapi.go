package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chepo92/PrintAndGo/job"
	"github.com/chepo92/PrintAndGo/serial"
)

func (wapi *WebApi) octoGetJobStatus(w http.ResponseWriter, r *http.Request) {
	c := wapi.jobManager.Subscribe()
	defer wapi.jobManager.Unsubscribe(c)

	select {
	case <-time.After(5 * time.Second):
		v := wapi.jobManager.Snapshot()
		jsonWrite(w, buildOctoJobReply(v))
		return

	case v := <-c:
		jsonWrite(w, buildOctoJobReply(v))
		return
	}
}

func buildOctoJobReply(v job.JobStatus) any {
	state := "Operational"

	printTime := 0
	printTimeLeft := 0
	filepos := 0
	completion := 0.0

	if v.Active {
		state = "Printing"
		printTime = int(time.Since(v.StartTime).Seconds())
		completion = v.DonePercent / 100.0

		if v.DonePercent > 1 {
			printTimeLeft = int(
				float64(printTime) * (100.0 - v.DonePercent) / v.DonePercent,
			)
		}
	}

	return struct {
		Job struct {
			File struct {
				Name   string `json:"name"`
				Origin string `json:"origin"`
				Size   int64  `json:"size"`
				Date   int64  `json:"date"`
			} `json:"file"`
			EstimatedPrintTime int `json:"estimatedPrintTime"`
			Filament           map[string]struct {
				Length float64 `json:"length"`
				Volume float64 `json:"volume"`
			} `json:"filament"`
		} `json:"job"`
		Progress struct {
			Completion    float64 `json:"completion"`
			Filepos       int     `json:"filepos"`
			PrintTime     int     `json:"printTime"`
			PrintTimeLeft int     `json:"printTimeLeft"`
		} `json:"progress"`
		State string `json:"state"`
	}{
		Job: struct {
			File struct {
				Name   string `json:"name"`
				Origin string `json:"origin"`
				Size   int64  `json:"size"`
				Date   int64  `json:"date"`
			} `json:"file"`
			EstimatedPrintTime int `json:"estimatedPrintTime"`
			Filament           map[string]struct {
				Length float64 `json:"length"`
				Volume float64 `json:"volume"`
			} `json:"filament"`
		}{
			File: struct {
				Name   string `json:"name"`
				Origin string `json:"origin"`
				Size   int64  `json:"size"`
				Date   int64  `json:"date"`
			}{
				Name:   v.File,
				Origin: "local",
				Size:   v.FileSize,
				Date:   v.FileDate,
			},
			EstimatedPrintTime: v.EstimatedTime,
			Filament: map[string]struct {
				Length float64 `json:"length"`
				Volume float64 `json:"volume"`
			}{
				"tool0": {
					Length: v.FilamentLength,
					Volume: v.FilamentVolume,
				},
			},
		},
		Progress: struct {
			Completion    float64 `json:"completion"`
			Filepos       int     `json:"filepos"`
			PrintTime     int     `json:"printTime"`
			PrintTimeLeft int     `json:"printTimeLeft"`
		}{
			Completion:    completion,
			Filepos:       filepos,
			PrintTime:     printTime,
			PrintTimeLeft: printTimeLeft,
		},
		State: state,
	}
}

func (wapi *WebApi) octoPostJob(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		wapi.error(w, "read body failure")
		return
	}

	var rq struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(buf, &rq); err != nil {
		wapi.error(w, "json unmarshal error")
		return
	}

	switch rq.Command {

	case "start":
		// Debe existir un archivo seleccionado / listo para imprimir
		// enqueuePrint ya maneja stream + task
		// if err := wapi.enqueueLastUploaded(false); err != nil {
		// 	wapi.error(w, err.Error())
		// 	return
		// }

		w.WriteHeader(http.StatusNoContent)
		return

	case "cancel":
		fmt.Printf("Ocotapi cancel \n")
		//wapi.task.Cancel()
		w.WriteHeader(http.StatusNoContent)
		return

	default:
		wapi.error(w, "invalid command")
		return
	}
}

func octoVersionReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		API     string `json:"api"`
		Version string `json:"server"`
		Banner  string `json:"text"`
	}{
		API:     "0.0.2",
		Version: "0.0.2",
		Banner:  "OctoPrint compatible PrintAndGo api",
	}
	jsonWrite(w, reply)
}

func octoSettingsReply(w http.ResponseWriter, r *http.Request) {
	reply := struct {
		Appearance struct {
			Name string `json:"name"`
		} `json:"appearance"`
	}{
		Appearance: struct {
			Name string `json:"name"`
		}{

			Name: "PrintAndGo",
		},
	}
	jsonWrite(w, reply)
}

func octoLoginReply(w http.ResponseWriter, r *http.Request) {
	reply := struct {
		Ext     bool   `json:"_is_external_client"`
		Session string `json:"session"`
	}{
		Session: "abadapi",
	}
	jsonWrite(w, reply)
}

type octoPrinterReply struct {
	Temperature map[string]any `json:"temperature"`
	SD          struct {
		Ready bool `json:"ready"`
	} `json:"sd"`
	State struct {
		Text  string       `json:"text"`
		Flags printerFlags `json:"flags"`
	} `json:"state"`
}

type printerFlags struct {
	Operational   bool `json:"operational"`
	Paused        bool `json:"paused"`
	Printing      bool `json:"printing"`
	Cancelling    bool `json:"cancelling"`
	Pausing       bool `json:"pausing"`
	SdReady       bool `json:"sdReady"`
	Error         bool `json:"error"`
	Ready         bool `json:"ready"`
	ClosedOrError bool `json:"closedOrError"`
}

func (wapi *WebApi) octoPrinterReply(w http.ResponseWriter, r *http.Request) {

	connected := wapi.serial.IsConnected()
	job := wapi.jobManager.Snapshot()

	var text string
	flags := printerFlags{}

	switch {
	case !connected:
		text = "Closed"
		flags.ClosedOrError = true

	case job.Error:
		text = "Error"
		flags.Error = true
		flags.ClosedOrError = true

	case job.Active:
		text = "Printing"
		flags.Printing = true

	default:
		text = "Operational"
		flags.Operational = true
		flags.Ready = true
	}

	flags.SdReady = connected

	reply := octoPrinterReply{
		Temperature: map[string]any{}, // vacío pero presente
	}

	reply.SD.Ready = flags.SdReady
	reply.State.Text = text
	reply.State.Flags = flags

	jsonWrite(w, reply)
}

type Commands struct {
	//CmdArray []string `json:"commands"`
	Command  string   `json:"command"`
	Commands []string `json:"commands"`
}

// octoPrinterCommand handles G-code commands sent via the OctoPrint-compatible API endpoint
func (wapi *WebApi) octoPrinterCommand(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		wapi.error(w, "read body failure")
		return
	}

	fmt.Printf("[octoPrinterCommand] Body: %s\n", string(bodyBytes))

	var cmds Commands
	if err := json.Unmarshal(bodyBytes, &cmds); err != nil {
		wapi.error(w, "json unmarshal error")
		return
	}

	// Rechazar comandos multilinea en "command"
	if strings.Contains(cmds.Command, "\n") {
		wapi.error(w, "multiline command not allowed")
		return
	}

	// Normalizar comandos
	var cmdList []string
	switch {
	case cmds.Command != "":
		cmdList = []string{cmds.Command}
	case len(cmds.Commands) > 0:
		cmdList = cmds.Commands
	default:
		wapi.error(w, "no command provided")
		return
	}

	if len(cmdList) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Verificar conexión serial
	if !wapi.serial.IsConnected() {
		fmt.Printf("[octoPrinterCommand] Serial not connected\n")
		wapi.error(w, "serial not connected")
		return
	}

	fmt.Printf("[octoPrinterCommand] Sending G-code %s\n", cmdList)

	// FASE ACTUAL: envío directo, sin task, sin parser
	for _, c := range cmdList {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		if err := wapi.serial.SendLine(c); err != nil {
			wapi.error(w, err.Error())
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)

}

// To be implemented
// octoServerReply

// octoLanguagesReply
// octoPrinterProfilesReply
// octoSlicingReply
// octoSystemCommandsReply
// octoTimelapseReply
// octoAccessReply
// octoUtilTestReply
// octoSetupWizardReply

func octoServerReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		Version  string `json:"version"`
		Safemode string `json:"safemode"`
	}{
		Version:  "0.0.0",
		Safemode: "false",
	}
	jsonWrite(w, reply)
}

type ConnectionReply struct {
	Current struct {
		State          string `json:"state"`
		Port           string `json:"port"`
		Baudrate       int    `json:"baudrate"`
		PrinterProfile string `json:"printerProfile"`
	} `json:"current"`
	Options struct {
		Ports           []string `json:"ports"`
		Baudrates       []int    `json:"baudrates"`
		PrinterProfiles []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"printerProfiles"`
		PortPreference           string `json:"portPreference"`
		BaudratePreference       int    `json:"baudratePreference"`
		PrinterProfilePreference string `json:"printerProfilePreference"`
		Autoconnect              bool   `json:"autoconnect"`
	} `json:"options"`
}

func (wapi *WebApi) octoGetConnectionReply(w http.ResponseWriter, r *http.Request) {
	state := wapi.serial.GetState()
	cfg := wapi.serial.GetConfig()

	ports, err := serial.ListPorts()
	if err != nil {
		ports = []string{} // fallback seguro
	}

	reply := ConnectionReply{
		Current: struct {
			State          string `json:"state"`
			Port           string `json:"port"`
			Baudrate       int    `json:"baudrate"`
			PrinterProfile string `json:"printerProfile"`
		}{
			State:          octoStateFromSerial(state),
			Port:           cfg.Port,
			Baudrate:       cfg.BaudRate,
			PrinterProfile: "_default",
		},
		Options: struct {
			Ports           []string `json:"ports"`
			Baudrates       []int    `json:"baudrates"`
			PrinterProfiles []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"printerProfiles"`
			PortPreference           string `json:"portPreference"`
			BaudratePreference       int    `json:"baudratePreference"`
			PrinterProfilePreference string `json:"printerProfilePreference"`
			Autoconnect              bool   `json:"autoconnect"`
		}{
			Ports:     ports,
			Baudrates: []int{115200, 250000, 230400, 57600, 38400, 19200, 9600},
			PrinterProfiles: []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			}{
				{Name: "Default", ID: "_default"},
			},
			PortPreference:           cfg.Port,
			BaudratePreference:       cfg.BaudRate,
			PrinterProfilePreference: "_default",
			Autoconnect:              false,
		},
	}

	jsonWrite(w, reply)
}

type ConnectionCommand struct {
	Command  string `json:"command"`
	Port     string `json:"port,omitempty"`
	Baudrate int    `json:"baudrate,omitempty"`
}

func (wapi *WebApi) octoPostConnectionReply(w http.ResponseWriter, rq *http.Request) {
	var cmd ConnectionCommand

	if err := json.NewDecoder(rq.Body).Decode(&cmd); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	switch cmd.Command {
	case "connect":
		cfg := serial.SerialConfig{
			Port:     cmd.Port,
			BaudRate: cmd.Baudrate,
		}

		if err := wapi.serial.Connect(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

	case "disconnect":
		if err := wapi.serial.Disconnect(); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

	default:
		http.Error(w, "unknown command", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func octoLanguagesReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}
func octoPrinterProfilesReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}
func octoSlicingReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}
func octoSystemCommandsReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}

func octoTimelapseReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}
func octoAccessReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}
func octoUtilTestReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}

func octoSetupWizardReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
}

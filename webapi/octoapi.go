package webapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/chepo92/PrintAndGo/serial"
)

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
		wapi.task.Cancel()
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
		API:     "0.0.0",
		Version: "0.0.0",
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

func octoPrinterReplyFake(w http.ResponseWriter, r *http.Request) {
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
		wapi.error(w, "serial not connected")
		return
	}

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
// octoGetConnectionReply
// octoPostConnectionReply
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
	state := wapi.serial.State()
	cfg := wapi.serial.Config()

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

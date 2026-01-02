package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/chepo92/PrintAndGo/store/bufstore"
)

func (wapi *WebApi) octoModifyJob(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		wapi.error(w, "read body failure")
		return
	}
	defer r.Body.Close()

	rq := struct {
		Command string `json:"command"`
	}{}
	if err := json.Unmarshal(buf, &rq); err != nil {
		wapi.error(w, "json unmarshal error")
		return
	}

	// Currently we don't support anything else.
	if rq.Command != "cancel" {
		wapi.error(w, "invalid command")
		return
	}
	wapi.task.Cancel()
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

// octoPrinterCommand handles G-code commands sent via the OctoPrint-compatible API endpoint. we only support "commands" field in JSON body, with an array of strings, each string being a G-code command.
func (wapi *WebApi) octoPrinterCommand(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	// Read the entire body into a byte slice
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		wapi.error(w, "read body failure")
		fmt.Printf("Error reading request body: %v\n", err)
		return
	}
	// Convert the byte slice to a string (if needed)
	bodyString := string(bodyBytes)

	//fmt.Fprintf(w, "Request body received successfully!")

	// We have to do something with the received command here, parse, check and send to printer, wait for OK, etc.
	// For now we just print it to stdout.
	fmt.Printf("Received request body: %s\n", bodyString)

	// Unmarshalling JSON/ByteArray into a struct
	var cmds Commands
	if err := json.Unmarshal(bodyBytes, &cmds); err != nil {
		wapi.error(w, "json unmarshal error")
		fmt.Printf("JSON unmarshal error: %v\n", err)
		return
	}
	// Rechazar comandos multilinea en command
	if strings.Contains(cmds.Command, "\n") {
		wapi.error(w, "multiline command not allowed")
		return
	}
	// Normalizar a una lista única de comandos
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
	// Print received commands for debugging
	fmt.Println("Commands:")
	for _, c := range cmdList {
		fmt.Println(c)
	}

	// If there's an active task, inject commands directly into the running print.
	if !wapi.task.Done() && wapi.task.IsActive() {
		fmt.Println("Injecting commands into active task")
		for _, c := range cmdList {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if err := wapi.task.InjectGcode(c); err != nil {
				wapi.error(w, "error injecting gcode")
				fmt.Printf("InjectGcode failed: %v\n", err)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	fmt.Println("No active task, enqueuing commands as a new print job")
	// No active task: create a temporary stream and enqueue as a normal print job.
	code := []byte(strings.Join(cmdList, "\n") + "\n")
	instr := bufstore.New(code, "Arbitrary Commands")
	if err := wapi.enqueuePrint(instr, false); err != nil {
		wapi.error(w, "error executing internal gcode")
		fmt.Printf("enqueuePrint failed: %v\n", err)
		return
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

func (wapi *WebApi) octoGetConnectionReply(w http.ResponseWriter, rq *http.Request) {
	reply := ConnectionReply{
		Current: struct {
			State          string `json:"state"`
			Port           string `json:"port"`
			Baudrate       int    `json:"baudrate"`
			PrinterProfile string `json:"printerProfile"`
		}{
			State:          "To be implemented",
			Port:           wapi.SerialPortInfo.Port,
			Baudrate:       wapi.SerialPortInfo.BaudRate,
			PrinterProfile: "To be implemented",
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
			Ports:     []string{},
			Baudrates: []int{},
			PrinterProfiles: []struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			}{},
			PortPreference:           "To be defined",
			BaudratePreference:       0,
			PrinterProfilePreference: "To be defined",
			Autoconnect:              false,
		},
	}
	jsonWrite(w, reply)
}

func octoPostConnectionReply(w http.ResponseWriter, rq *http.Request) {
	reply := struct {
		TBI string `json:"tbi"`
	}{
		TBI: "To be implemented",
	}
	jsonWrite(w, reply)
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

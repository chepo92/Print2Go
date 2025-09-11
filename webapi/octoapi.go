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
		API:     "0.2",
		Version: "0.20220626",
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
	CmdArray []string `json:"commands"`
}

// octoPrinterCommand handles G-code commands sent via the OctoPrint-compatible API endpoint. we only support "commands" field in JSON body, with an array of strings, each string being a G-code command.
func (wapi *WebApi) octoPrinterCommand(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	// Read the entire body into a byte slice
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		fmt.Printf("Error reading request body: %v", err)
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
	err = json.Unmarshal(bodyBytes, &cmds)
	if err != nil {
		// Handle error
	}
	fmt.Println("Commands:")
	for i := 0; i < len(cmds.CmdArray); i++ {
		fmt.Println(cmds.CmdArray[i])
	}

	code := []byte(strings.Join(cmds.CmdArray, "\n") + "\n")
	instr := bufstore.New(code, "Arbitrary Commands")
	if err := wapi.enqueuePrint(instr, false); err != nil {
		wapi.error(w, "error executing internal gcode")
	}

	// code := []byte(strings.Join(pl, "\n") + "\n")
	// instr := bufstore.New(code, q)

	// if err := wapi.enqueuePrint(instr, false); err != nil {
	// 	wapi.error(w, "error executing internal gcode")
	// } else {
	// 	// Octoprint replies with 204 No Content on success.
	// 	w.WriteHeader(http.StatusNoContent)
	// }

	w.WriteHeader(http.StatusNoContent)

}

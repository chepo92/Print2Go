package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		Banner:  "OctoPrint compatible takoprint api",
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

			Name: "takoprint",
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

type Command struct {
	Command string `json:"command"`
}

func octoPrinterCommand(w http.ResponseWriter, r *http.Request) {

	// Declare a new Person struct.
	var cmd Command

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&cmd)
	if err != nil {
		fmt.Println("Error decoding JSON: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("Received request body + : %+v\n", cmd)

	w.WriteHeader(http.StatusNoContent)

}

func parseJSON(r *http.Request) {

	// Declare a new struct.
	var cmd Command

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&cmd)
	if err != nil {
		fmt.Println("Error decoding JSON: ", err)
		return
	}
	fmt.Printf("Received request body + : %+v\n", cmd)

}

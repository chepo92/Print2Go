package webapi

import (
	"fmt"
	"net/http"
)

func (wapi *WebApi) pagJobStatus(w http.ResponseWriter, rq *http.Request) {

	if wapi.jobManager == nil {
		jsonWrite(w, map[string]string{"error": "job manager not initialized"})
		return
	}
	js := wapi.jobManager.Snapshot()

	js.Motd = wapi.readMotd() // opcional
	jsonWrite(w, js)

}

func (wapi *WebApi) pagJobCancel(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[pagJobCancel] \n")
	if wapi.jobManager == nil {
		jsonWrite(w, map[string]string{"error": "job manager not initialized"})
		return
	}
	wapi.jobManager.Cancel()
	jsonWrite(w, nil)
}

func (wapi *WebApi) resetQueue(w http.ResponseWriter, r *http.Request) {

	wapi.serial.ResetQueues()

	w.WriteHeader(http.StatusNoContent)
}

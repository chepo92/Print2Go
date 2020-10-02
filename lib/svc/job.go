package svc

import (
	"net/http"
)

type jobStatus struct {
}

func (svc *Svc) jobStatus(w http.ResponseWriter, rq *http.Request) {
	svc.RLock()
	defer svc.RUnlock()

	if rq.Method == "GET" {
		svc.error(w, "Fixme")
		return
	}
	if rq.Method == "POST" && svc.task != nil {
		if rq.FormValue("cancel") == "true" {
			svc.task.Cancel()
			return
		}
	}
	svc.error(w, "bad request to job status endpoint")
}

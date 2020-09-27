package svc

import (
	"net/http"
)

type jobStatus struct {
	Job    struct{} `json:"job"`
	Status string   `json:"status"`
}

func (svc *Svc) jobStatus(w http.ResponseWriter, rq *http.Request) {
	svc.RLock()
	defer svc.RUnlock()

	if rq.Method == "GET" {
		if svc.task != nil {
			jsonWrite(w, jobStatus{
				Status: svc.task.Describe(),
			})
		}
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

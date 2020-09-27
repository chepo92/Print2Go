package svc

import (
	"net/http"
)

type jobStatus struct {
	Job    struct{} `json:"job"`
	Status string   `json:"status"`
	Buffer []string `json:"buffer"`
}

func (svc *Svc) jobStatus(w http.ResponseWriter, rq *http.Request) {
	svc.RLock()
	defer svc.RUnlock()

	if rq.Method == "GET" {
		if svc.task != nil {

			dbg := make([]string, 0)
			for _, v := range svc.task.LogBuffer() {
				if v == nil {
					continue
				}
				dbg = append(dbg, v.LastSent)
			}
			jsonWrite(w, jobStatus{
				Status: svc.task.Describe(),
				Buffer: dbg,
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

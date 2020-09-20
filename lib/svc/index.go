package svc

import (
	"net/http"
)

func (svc *Svc) indexPage(w http.ResponseWriter) {
	txt := "no print running"
	svc.Lock()
	if svc.task != nil {
		txt = svc.task.Describe()
	}
	svc.Unlock()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(txt))
}

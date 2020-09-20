package svc

import (
	"fmt"
	"net/http"
)

func (svc *Svc) log(f string, args ...interface{}) {
	fmt.Printf(f, args...)
}

func (svc *Svc) error(w http.ResponseWriter, msg string) {
	http.Error(w, msg, 503)
}

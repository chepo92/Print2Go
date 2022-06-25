package webapi

import (
	"fmt"
	"net/http"
)

func (wapi *WebApi) log(f string, args ...interface{}) {
	fmt.Printf(f+"\n", args...)
}

func (wapi *WebApi) error(w http.ResponseWriter, msg string) {
	http.Error(w, msg, 503)
}

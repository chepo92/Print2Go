package webapi

import (
	"fmt"
	"net/http"
	"os"
)

func (wapi *WebApi) log(f string, args ...interface{}) {
	fmt.Printf(f+"\n", args...)
}

func (wapi *WebApi) error(w http.ResponseWriter, msg string) {
	http.Error(w, msg, 503)
}

func (wapi *WebApi) readMotd() string {
	pl, err := os.ReadFile(wapi.motdFile)
	if err != nil {
		wapi.log("failed to read motd file %q: %v", wapi.motdFile, err)
	}
	return string(pl)
}

package webapi

import (
	"net/http"
)

const (
	maxUploadSize = 1024 * 1024 * 50
)

type uploadReplyLocal struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

type uploadReplyFiles struct {
	Local *uploadReplyLocal `json:"local"`
}

func (wapi *WebApi) localUpload(w http.ResponseWriter, rq *http.Request) {
	f, h, err := rq.FormFile("file")
	if err != nil {
		wapi.error(w, "failed to parse form data")
		return
	}

	if h.Size > maxUploadSize {
		wapi.error(w, "file upload exceeds limit")
		return
	}

	path := rq.FormValue("path")
	if err := wapi.storage.UploadFile(path, h.Filename, f); err != nil {
		wapi.error(w, "failed to store file")
		return
	}

	reply := struct {
		Files *uploadReplyFiles `json:"files"`
		Done  bool              `json:"done"`
	}{
		Files: &uploadReplyFiles{
			Local: &uploadReplyLocal{
				Origin: "local",
				Name:   h.Filename,
			},
		},
		Done: true,
	}

	jsonWrite(w, reply)

	if rq.FormValue("print") == "true" {
		shutdown := rq.FormValue("shutdown") == "true"

		wapi.log("Enqueueing %s, %s for printing after upload. Shutdown = %v", path, h.Filename, shutdown)
		instr, err := wapi.storage.ReadFile(path, h.Filename)
		if err != nil {
			wapi.log("failed to read file we just uploaded: %v", err)
			return
		}
		if err := wapi.enqueuePrint(instr, shutdown); err != nil {
			wapi.log("enqueueing failed: %v", err)
			return
		}
	}
}

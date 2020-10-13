package svc

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

func (svc *Svc) localUpload(w http.ResponseWriter, rq *http.Request) {
	f, h, err := rq.FormFile("file")
	if err != nil {
		svc.error(w, "failed to parse form data")
		return
	}

	if h.Size > maxUploadSize {
		svc.error(w, "file upload exceeds limit")
		return
	}

	path := rq.FormValue("path")
	if err := svc.storage.UploadFile(path, h.Filename, f); err != nil {
		svc.error(w, "failed to store file")
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

		svc.log("Enqueueing %s, %s for printing after upload. Shutdown = %v", path, h.Filename, shutdown)
		instr, err := svc.storage.ReadFile(path, h.Filename)
		if err != nil {
			svc.log("failed to read file we just uploaded: %v", err)
			return
		}
		if err := svc.enqueuePrint(instr, shutdown); err != nil {
			svc.log("enqueueing failed: %v", err)
			return
		}
	}
}

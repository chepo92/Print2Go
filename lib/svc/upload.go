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
	//	w.Header().Set("Location", "todo: management url")
	jsonWrite(w, reply)

	if rq.FormValue("print") == "true" {
		svc.log("Enqueueing %s, %s for printing after upload", path, h.Filename)
		svc.enqueuePrint(path, h.Filename)
	}
}

package webapi

import (
	"fmt"
	"net/http"
	"time"
)

const (
	// maxUploadSize is 50MB
	maxUploadSize = 1024 * 1024 * 50
)

type uploadReplyLocal struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

type uploadReplyFiles struct {
	Local *uploadReplyLocal `json:"local"`
}

// localUpload handles file uploads to the local storage.
// If the "print" form value is set to "true", it will enqueue the file for printing after upload.
// If the "shutdown" form value is set to "true", it will shutdown the hardware after printing, will do nothing if shutdown is nor set.
func (wapi *WebApi) localUpload(w http.ResponseWriter, rq *http.Request) {

	// Dump the request to a byte slice
	// requestDump, err := httputil.DumpRequest(rq, true) // 'true' includes the body
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("Error dumping request: %v", err), http.StatusInternalServerError)
	// 	return
	// }

	// fmt.Printf("--- Incoming Request ---\n%s\n", string(requestDump))
	// Necesario para acceder a multipart sin consumir body arbitrariamente
	err := rq.ParseMultipartForm(32 << 20) // 32MB max in memory
	if err != nil {
		wapi.error(w, fmt.Sprintf("Error parsing multipart: %v", err))
		return
	}

	fmt.Println("---- Multipart Form Fields ----")
	for key, vals := range rq.MultipartForm.Value {
		fmt.Printf("%s = %v\n", key, vals)
	}

	// show file info
	fileHeaders := rq.MultipartForm.File["file"]
	if len(fileHeaders) > 0 {
		fh := fileHeaders[0]
		fmt.Printf("file = { filename=%q, size=%d, header=%v }\n",
			fh.Filename, fh.Size, fh.Header)
	} else {
		fmt.Println("file = <none>")
	}

	f, h, err := rq.FormFile("file")
	if err != nil {
		wapi.error(w, "Failed to parse form data")
		return
	}
	defer f.Close()

	if h.Size > maxUploadSize {
		wapi.error(w, "File upload exceeds limit")
		return
	}
	// Optional path parameter
	path := rq.FormValue("path")
	wapi.log("Specified save path: %s", path)

	if err := wapi.storage.UploadFile(path, h.Filename, f); err != nil {
		wapi.error(w, "Failed to store file")
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

		instr, err := wapi.storage.ReadFile(path, h.Filename)
		if err != nil {
			wapi.log("Failed to read file we just uploaded: %v", err)
			return
		}
		wapi.log("Enqueueing: %s for printing after upload. Shutdown after print: %v", h.Filename, shutdown)

		fmt.Printf("calling jobManager.StartPrint from localUpload\n")
		// Start job using JobManager
		if err := wapi.jobManager.StartPrint(instr); err != nil {
			wapi.log("Failed to start print job: %v", err)
			return
		}

		// Optional: launch goroutine to shutdown after job finishes
		if shutdown {
			go func() {
				// Wait until job finishes
				for wapi.jobManager.Snapshot().Active {
					time.Sleep(time.Second)
				}
				wapi.log("Shutting down printer after job completion")
				wapi.shutdown()
			}()
		}
	}
}

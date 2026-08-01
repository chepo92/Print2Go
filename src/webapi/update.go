package webapi

import (
	"io"
	"net/http"
	"os"
	"os/exec"
)

const (
	updateBinaryPath = "/tmp/Print2Go.new"
	updateScriptPath = "/tmp/print2go-update.sh"
)

func (wapi *WebApi) pagUpdate(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "invalid multipart form",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "missing file",
		})
		return
	}
	defer file.Close()

	dst, err := os.OpenFile(
		updateBinaryPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)

	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "cannot create update file",
		})
		return
	}

	_, err = io.Copy(dst, file)
	dst.Close()

	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "copy failed",
		})
		return
	}

	err = os.Chmod(
		updateBinaryPath,
		0755,
	)

	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "chmod failed",
		})
		return
	}

	err = validateUpdateBinary(updateBinaryPath)

	if err != nil {
		os.Remove(updateBinaryPath)

		jsonWrite(w, map[string]string{
			"error": err.Error(),
		})
		return
	}

	err = createUpdateScript()

	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "cannot create updater",
		})
		return
	}

	err = exec.Command(
		"sh",
		updateScriptPath,
	).Start()

	if err != nil {
		jsonWrite(w, map[string]string{
			"error": "cannot start updater",
		})
		return
	}

	jsonWrite(w, map[string]string{
		"status": "updating",
	})

}

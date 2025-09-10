package webapi

import (
	"net/http"
)

func cameraPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html><head><title>PrintAndGo webcam</title></head>
<body bgcolor="#323232">
<center>
<img src="camera/stream.mjpeg">
</center>
</body>
`))
}

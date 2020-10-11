package svc

import (
	"net/http"
)

func (svc *Svc) cameraPage(w http.ResponseWriter) {
	w.Write([]byte(`<html><head><title>Takoprint webcam</title></head>
<body bgcolor="#323232">
<center>
<img src="camera/stream.mjpeg">
</center>
</body>
`))
}

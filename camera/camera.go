package camera

import (
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"sync"
	"time"
)

// Camera holds all connected feed consumers and controls the feed input.
type Camera struct {
	sync.RWMutex
	channels map[string]chan []byte
	feed     *feedReader
}

// New returns a new camera instance.
func New() *Camera {
	return &Camera{
		channels: make(map[string]chan []byte),
	}
}

// WriteStream writes a multipart (mjpeg) stream to the given response writer.
func (cam *Camera) WriteStream(w http.ResponseWriter, r *http.Request) {
	c := cam.feedChan()
	defer cam.closeFeed(c)

	mpw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", `multipart/x-mixed-replace;boundary=`+mpw.Boundary())
	for {
		select {
		case <-time.After(time.Second * 10):
			// something is wrong with the feed, probably.
			return
		case pl := <-c:
			if err := putPart(mpw, pl); err != nil {
				return
			}
		}
	}
}

// putPart writes a single multipart element to the writer.
func putPart(mpw *multipart.Writer, pl []byte) error {
	w, err := mpw.CreatePart(textproto.MIMEHeader{
		"Content-Type":   []string{"image/jpeg"},
		"Content-Length": []string{strconv.Itoa(len(pl))},
	})
	if err != nil {
		return err
	}
	if _, err := w.Write(pl); err != nil {
		return err
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

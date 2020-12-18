package camera

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"time"

	"git.sr.ht/~adrian-blx/takoprint/lib/ezcam"
	"github.com/pbnjay/pixfont"
)

const (
	camSleep = time.Second * 1
)

func RunPipe(dev string, w, h uint32) {
	zcam, err := ezcam.New(dev, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize webcam at %s: %v\n", dev, err)
		os.Exit(2)
	}

	s := make([]byte, 5)
	s[0] = 0x83 // magic
	for {
		pic, err := zcam.Grab()
		if err != nil {
			fmt.Fprintf(os.Stderr, "webcam failure: %v\n", err)
			os.Exit(2)
		}

		img := image.NewRGBA(image.Rect(0, 0, zcam.Width(), zcam.Height()))
		draw.Draw(img, img.Bounds(), pic, image.Point{0, 0}, draw.Src)
		drawTime(img)

		pl := &bytes.Buffer{}
		jpeg.Encode(pl, img, &jpeg.Options{Quality: 95})
		binary.BigEndian.PutUint32(s[1:], uint32(len(pl.Bytes())))
		if _, err := os.Stdout.Write(s); err != nil {
			os.Exit(1)
		}
		if nw, err := os.Stdout.Write(pl.Bytes()); err != nil || nw != len(pl.Bytes()) {
			os.Exit(1)
		}
		time.Sleep(camSleep)
	}
}

func drawTime(img draw.Image) {
	str := time.Now().Format(time.RFC3339)

	pixfont.DrawString(img, 10, img.Bounds().Dy()-15, str, color.RGBA{0, 0, 0, 255})
	pixfont.DrawString(img, 9, img.Bounds().Dy()-16, str, color.RGBA{255, 255, 0, 255})
}

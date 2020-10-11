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

	"github.com/pbnjay/pixfont"
)

func RunPipe(dev string, w, h int) {
	s := make([]byte, 5)
	s[0] = 0x83 // magic

	for {

		rect := image.Rect(0, 0, w, h)
		img := image.NewRGBA(rect)
		draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
		pixfont.DrawString(img, 10, img.Bounds().Dy()-15, fmt.Sprintf("%+v", time.Now()), color.Black)

		pl := &bytes.Buffer{}
		jpeg.Encode(pl, img, &jpeg.Options{Quality: 85})
		binary.BigEndian.PutUint32(s[1:], uint32(len(pl.Bytes())))
		if _, err := os.Stdout.Write(s); err != nil {
			os.Exit(1)
		}
		if nw, err := os.Stdout.Write(pl.Bytes()); err != nil || nw != len(pl.Bytes()) {
			os.Exit(1)
		}
		time.Sleep(time.Millisecond * 800)
	}
}

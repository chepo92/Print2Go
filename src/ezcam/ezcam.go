package ezcam

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/blackjack/webcam"
)

type Ezcam struct {
	cam    *webcam.Webcam
	width  uint32
	height uint32
}

type pick struct {
	fmt webcam.PixelFormat
	fs  webcam.FrameSize
}

func New(dev string, w, h uint32) (*Ezcam, error) {
	cam, err := webcam.Open(dev)
	if err != nil {
		return nil, err
	}

	p, err := bestFrameSize(cam, w, h)
	if err != nil {
		return nil, err
	}

	_, fw, fh, err := cam.SetImageFormat(p.fmt, p.fs.MaxWidth, p.fs.MaxHeight)
	if err != nil {
		return nil, err
	}

	cam.SetAutoWhiteBalance(true)
	cam.SetBufferCount(1) // we don't grab frames so often, so throw them away asap.
	cam.StartStreaming()
	return &Ezcam{
		width:  fw,
		height: fh,
		cam:    cam,
	}, nil
}

func (c *Ezcam) Stop() error {
	c.cam.StopStreaming()
	return c.cam.Close()
}

func (c *Ezcam) Width() int {
	return int(c.width)
}

func (c *Ezcam) Height() int {
	return int(c.height)
}

func (c *Ezcam) Grab() (image.Image, error) {
	if err := c.cam.WaitForFrame(1); err != nil {
		return nil, err
	}

	f, err := c.cam.ReadFrame()
	if err != nil {
		return nil, err
	}

	yuyv := image.NewYCbCr(image.Rect(0, 0, c.Width(), c.Height()), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		yuyv.Y[i*2] = f[ii]
		yuyv.Y[i*2+1] = f[ii+2]
		yuyv.Cb[i] = f[ii+1]
		yuyv.Cr[i] = f[ii+3]
	}
	return yuyv, nil
}

func bestFrameSize(cam *webcam.Webcam, w, h uint32) (*pick, error) {
	var fallback, picked pick

	for pf, n := range cam.GetSupportedFormats() {
		if !strings.Contains(n, "YUYV") {
			continue
		}
		for _, fs := range cam.GetSupportedFrameSizes(pf) {
			fallback = pick{fmt: pf, fs: fs}
			if fs.MaxWidth >= w && fs.MaxHeight >= h { // exact match or better.
				fmt.Fprintf(os.Stderr, "# fmt: found %v/%+v (want %d x %d)\n", n, fs, w, h)
				if picked.fs.MaxWidth == 0 || (picked.fs.MaxWidth > fs.MaxWidth && picked.fs.MaxHeight > fs.MaxHeight) {
					picked = pick{fmt: pf, fs: fs}
				}
			}
		}
	}

	if picked.fs.MaxWidth != 0 {
		return &picked, nil
	}
	if fallback.fs.MaxWidth != 0 {
		return &fallback, nil
	}
	return nil, fmt.Errorf("no suitable resolution found")
}

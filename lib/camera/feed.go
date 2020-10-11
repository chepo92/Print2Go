package camera

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// feedReader runs the subprocess and broadcasts new data to all camera consumers.
type feedReader struct {
	cam    *Camera
	ctx    context.Context
	cancel context.CancelFunc
}

// FeedChan returns a channel which will deliver image data.
func (cam *Camera) feedChan() chan []byte {
	c := make(chan []byte)

	cam.Lock()
	defer cam.Unlock()
	if cam.feed == nil {
		// no running subprocess, need to launch one.
		cam.feed = newFeedReader(cam)
		go func() {
			err := cam.feed.Start()
			fmt.Printf("feed exited with %v\n", err)
		}()
	}
	cam.channels[fmt.Sprintf("%p", c)] = c
	return c
}

// Closes a previously registered feed.
func (cam *Camera) closeFeed(c chan []byte) {
	cam.Lock()
	defer cam.Unlock()

	k := fmt.Sprintf("%p", c)
	if _, ok := cam.channels[k]; !ok {
		panic(fmt.Errorf("unregistering bad channel %v", c))
	}
	delete(cam.channels, k)

	if len(cam.channels) == 0 {
		// this was the last consumer, so we can kill the subprocess.
		go cam.feed.Stop()
		cam.feed = nil
	}
}

// newFeedReader returns an initialized instance.
func newFeedReader(cam *Camera) *feedReader {
	ctx, cancel := context.WithCancel(context.Background())
	return &feedReader{
		cam:    cam,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start launches a pipe and broadcasts the read data to all channels.
func (fr *feedReader) Start() error {
	cmd := exec.CommandContext(fr.ctx, os.Args[0], ":camera-pipe")
	defer cmd.Wait()

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// buffer which holds the marker, format: <1:magic><4:length>.
	s := make([]byte, 5)
	for {
		if _, err := io.ReadAtLeast(pipe, s, len(s)); err != nil {
			return err
		}
		if s[0] != 0x83 {
			return fmt.Errorf("invalid magic found: %X", s[0])
		}

		pllen := int(binary.BigEndian.Uint32(s[1:]))
		pl := make([]byte, pllen)
		if _, err := io.ReadAtLeast(pipe, pl, pllen); err != nil {
			return err
		}

		fr.cam.RLock()
		for _, c := range fr.cam.channels {
			c := c
			go func() {
				select {
				case c <- pl:
					// sent data
				default:
					fmt.Printf("dropping data for full channel %p\n", c)
				}
			}()
		}
		fr.cam.RUnlock()
	}
}

// Stop terminates a feed reader instance.
func (fr *feedReader) Stop() {
	fr.cancel()
}

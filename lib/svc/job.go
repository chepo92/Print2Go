package svc

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"
)

type jobStatus struct {
	File        string        `json:"file"`
	Done        float64       `json:"donePct"`
	Desc        string        `json:"description"`
	Cmd         string        `json:"lastcmd"`
	Reply       string        `json:"lastreply"`
	Active      bool          `json:"active"`
	Hash        uint32        `json:"fp"`
	Age         time.Duration `json:"ageSecs"`
	RunDuration string        `json:"runDuration"`
}

func (svc *Svc) jobStatus(w http.ResponseWriter, rq *http.Request) {
	nid, _ := strconv.Atoi(rq.FormValue("id"))
	id := uint32(nid)

	c := svc.task.Subscribe()
	defer svc.task.Unsubscribe(c)

	var js jobStatus
	for range []int{1, 2} {
		select {
		case <-time.After(time.Second * 20):
		case v := <-c:
			runDur := time.Now().Sub(v.Started)
			js = jobStatus{
				File:        v.File,
				Done:        v.Done,
				Desc:        v.Text,
				Cmd:         v.LastCommand,
				Reply:       v.LastReply,
				Active:      v.Active,
				Age:         runDur / time.Second,
				RunDuration: runDur.Round(time.Second).String(),
			}
			h := fnv.New32a()
			h.Write([]byte(fmt.Sprintf("%+v", js)))
			js.Hash = h.Sum32() & 0xEFFF
		}
		if js.Hash != id {
			break
		}
	}
	jsonWrite(w, js)
}

func (svc *Svc) apiJobStatus(w http.ResponseWriter) {
	c := svc.task.Subscribe()
	defer svc.task.Unsubscribe(c)
	select {
	case <-time.After(5 * time.Second):
	case v := <-c:
		// state strings need to map 1:1 with what clients expect (ever heard of enums?).
		state := "Operational"
		ptime := int(0)
		tleft := int(0)
		if v.Active {
			state = "Printing"
			ptime = int(time.Now().Sub(v.Started).Seconds())
			tleft = int(100 / (v.Done + 0.001) * float64(ptime))
		}
		res := struct {
			Progress struct {
				Completion float64 `json:"completion"`
				PrintTime  int     `json:"printTime"`
				Left       int     `json:"printTimeLeft"`
			} `json:"progress"`
			Job struct {
				File struct {
					Name string `json:"name"`
				} `json:"file"`
			} `json:"job"`
			State string `json:"state"`
		}{
			Progress: struct {
				Completion float64 `json:"completion"`
				PrintTime  int     `json:"printTime"`
				Left       int     `json:"printTimeLeft"`
			}{
				Completion: v.Done,
				PrintTime:  ptime,
				Left:       tleft,
			},
			Job: struct {
				File struct {
					Name string `json:"name"`
				} `json:"file"`
			}{
				File: struct {
					Name string `json:"name"`
				}{
					Name: v.Text,
				},
			},
			State: state,
		}
		jsonWrite(w, res)
		return
	}
	jsonWrite(w, nil)
}

func (svc *Svc) jobCancel(w http.ResponseWriter) {
	svc.task.Cancel()
	jsonWrite(w, nil)
}

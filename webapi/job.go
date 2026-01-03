package webapi

import (
	"fmt"
	"hash/fnv"
	"net/http"
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
	Motd        string        `json:"motd"`
	Status      string        `json:"status"`
	Error       bool          `json:"error"`
}

func (wapi *WebApi) pagJobStatus(w http.ResponseWriter, rq *http.Request) {

	v := wapi.task.Snapshot() //

	runDur := time.Since(v.Started)

	js := jobStatus{
		File:        v.File,
		Done:        v.DonePercent,
		Desc:        v.Text,
		Cmd:         v.LastCommand,
		Reply:       v.LastReply,
		Active:      v.Active,
		Age:         runDur / time.Second,
		RunDuration: runDur.Round(time.Second).String(),
		Error:       v.Error,
		Motd:        wapi.readMotd(),
	}

	js.Age = 0
	js.RunDuration = ""

	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%+v", js)))
	js.Hash = h.Sum32() & 0xEFFF

	jsonWrite(w, js)
}

func (wapi *WebApi) octoJobStatus(w http.ResponseWriter, r *http.Request) {
	c := wapi.task.Subscribe()
	defer wapi.task.Unsubscribe(c)
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
			tleft = int(100 / (v.DonePercent + 0.001) * float64(ptime))
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
				Completion: v.DonePercent,
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

func (wapi *WebApi) pagJobCancel(w http.ResponseWriter, r *http.Request) {
	wapi.task.Cancel()
	jsonWrite(w, nil)
}

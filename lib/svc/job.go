package svc

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"
)

type jobStatus struct {
	File   string  `json:"file"`
	Done   float64 `json:"donePct"`
	Desc   string  `json:"description"`
	Cmd    string  `json:"lastcmd"`
	Reply  string  `json:"lastreply"`
	Active bool    `json:"active"`
	Hash   uint32  `json:"fp"`
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
			js = jobStatus{
				File:   v.File,
				Done:   v.Done,
				Desc:   v.Text,
				Cmd:    v.LastCommand,
				Reply:  v.LastReply,
				Active: v.Active,
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

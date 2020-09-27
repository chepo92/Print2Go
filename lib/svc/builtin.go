package svc

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
)

var gCodes = map[string][]string{
	"z-axis": {
		"M117 z-axis lift",
		"M104 S0 ; turn off heat",
		"M140 S0 ; turn off bed",
		"G91     ; relative movement",
		"G1 Z20  ; go up 2cm",
		"G90     ; absolute movement",
	},
	"heat": {
		"M117 heating to 210 degree",
		"M104 S210 ; set hotend 210",
		"M109 S210 ; wait for 210",
	},
	"f-move": {
		"M117 moving filament",
		"M104 S210 ; set hotend 210",
		"M109 S210 ; wait for 210",
		"G1 E20    ; move 20mm of filament",
	},
	"reset": {
		"M104 S0          ; turn off temperature",
		"M140 S0          ; turn off heatbed",
		"M107             ; turn off fan",
		"G1 Z27.96 F600   ; Move print head up",
		"G1 X0 Y200 F3000 ; present print",
		"M84 X Y E        ; disable motors",
	},
}

func (svc *Svc) enqueueBuiltin(w http.ResponseWriter, rq *http.Request) {
	q := rq.FormValue("action")

	pl, ok := gCodes[q]
	if !ok {
		svc.error(w, "unknown internal gcode")
		return
	}

	code := ioutil.NopCloser(bytes.NewReader([]byte(strings.Join(pl, "\n") + "\n")))
	if err := svc.storage.UploadFile("internal", "gcode-"+q, code); err != nil {
		svc.error(w, "error writing gcode file")
		return
	}

	if err := svc.enqueuePrint("internal", "gcode-"+q); err != nil {
		svc.error(w, "error executing internal gcode")
	}
}

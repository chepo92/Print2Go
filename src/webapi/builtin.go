package webapi

import (
	"fmt"
	"net/http"
)

var gCodes = map[string][]string{
	"z-axis": {
		"M117 z-axis lift",
		"G91     ; relative movement",
		"G1 Z20  ; go up 2cm",
		"G90     ; absolute movement",
	},
	"heat": {
		"M117 heating to 210 degree",
		"M104 S210 ; set hotend 210",
		"M140 S60  ; set bed temp to 60",
	},
	"cooldown": {
		"M104 S0 ; Turn off hotend heater",
		"M140 S0 ; Turn off heated bed",
	},
	"E+10": {
		"M117 moving filament",
		"M104 S210 ; set hotend 210",
		"M109 S210 ; wait for 210",
		"G91 ; Set all axes to relative",
		"G1 E10 F500    ; move 10mm of filament",
		"G90     ; absolute movement",
	},
	"E-10": {
		"M117 moving filament",
		"M104 S210 ; set hotend 210",
		"M109 S210 ; wait for 210",
		"G91 ; Set all axes to relative",
		"G1 E-10 F500    ; move 10mm of filament backwards",
		"G90     ; absolute movement",
	},
	"home": {
		"G28 ; home all axes",
	},
	"disableXYMotors": {
		"M84 X Y; disable motors",
	},
	"reset": {
		"M104 S0          ; turn off temperature",
		"M140 S0          ; turn off heatbed",
		"M107             ; turn off fan",
		"G1 Z27.96 F600   ; Move print head up",
		"M84 X Y E        ; disable motors",
	},
}

func (wapi *WebApi) pagEnqueueBuiltin(w http.ResponseWriter, rq *http.Request) {

	q := rq.FormValue("action")

	pl, ok := gCodes[q]
	if !ok {
		wapi.error(w, "unknown internal gcode")
		return
	}

	if !wapi.serial.IsConnected() {
		wapi.error(w, "printer not connected")
		return
	}

	fmt.Printf("[builtin] executing action: %s\n", q)

	if err := wapi.serial.SendGcodeLines(pl); err != nil {
		wapi.error(w, "error executing internal gcode")
	}
}

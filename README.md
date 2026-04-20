# Print2Go

3D printer host written in Go language with "Octoprint" functionalities 

Convert any router with a USB port in a 3D printer host, with an Octoprint-like API

Print2Go is a lightweight and simple web based program written in Go to feed gcode to a 3d printer (aka gcode sender, 3d printer host)

It offers a convenient webinterface and mimics Octoprints upload API, meaning that common slicer software will be able to directly upload gcode to Print2Go.

It's based in the code of [Takoprint](https://git.sr.ht/~adrian-blx/takoprint) by [Adrian](https://github.com/adrian-bl)

## Features

- Written in Go and Multiplatform: Compile from source for your target device or use the release binaries, run it in your host device.
- Compatible with Windows and Linux operating systems. 
- Compatible with many target hardware, if your target is supported in Go, it should be compatible 
- Aimed for lightweight: Print2Go doesn't need a lot of resources is only a 12MB binary (still not optimized)
- Low hardware resource requirement: Will work well even on older/or low specs hardware
- Can run in OpenWRT. This means you can use almost any router with usb port that supports OpenWRT or even in the Creality Wifi Box
- Compatible with any 3D printer with usb port that has some version of Marlin firmware (or accepts standard gcode over serial)
- Octoprint emulation: Mimics the basic Octoprint API allowing for direct Gcode upload from various slicers (Cura, PrusaSlicer)
- Custom hooks: Print2Go can execute custom scripts after your print is finished (eg. to turn off your printer).
- Simple: Web User interface with just the minimum required operations for printing
- OctoPrint minimal API implementation: Send custom Gcode, and upload file for inmediate printing

## Brief story of the objective an purpose of this project

This project is related to [OctoWrt](https://github.com/shivajiva101/OctoWrt), and the addition of the Creality WifiBox (WB01) in [OpenWRT](https://github.com/openwrt/openwrt/pull/19686). OctoPrint is a great sofware and tool, but not very optimized for low resource hardware (lower than a raspberry pi 3, which was the original target hardware as far as I know) such as a router. I had previously contributed in a Octoprint-like (or mimic) firmware for embedded devices as ESP8266 or ESP32 see [WirelessPrinting](https://github.com/probonopd/WirelessPrinting), but there was missing sofware solution for the intermediate hardware between embedded devices and mini-pc. So it needed some leverage, then I found [Takoprint](https://git.sr.ht/~adrian-blx/takoprint), added some functions and connected the other projects (OctoWrt and OpenWRT) together.


## Screenshots

![webinterface](/img/print2go_home_gui.png)

![webinterface](/img/print2go_upload.png)

![webinterface](/img/print2go_printing.png)

![webinterface](/img/print2go_done.png)


##  Install, Run and Usage 

No install is required, just run the executables/binaries according to your platform/OS

### Run in host machine

In a command line or shell in windows or linux, run:
```
./Print2Go  -tty <host_device_path>
```

Note: In windows you can double click the .exe, but will use the default configuration 

Examples:
```
./Print2Go -tty COM3
./Print2Go -tty /dev/ttyUSB0
```

Print2Go is configured via flags. By default, Print2Go will listen on
`127.0.0.1:5001` and expect a printer on `/dev/ttyUSB0` (Linux) or COM3 (Windows):

```shell
$ ./Print2Go -h
Usage of ./Print2Go:
  -baud int
        baud rate of -port (default 115200)
  -gcode string
        file containing gcode
  -listen string
        ip:port to bind to (default "127.0.0.1:5001")
  -shutdown-script string
        script to execute to shutdown the printer (default "/usr/lib/Print2Go-shutdown.sh")
  -storage string
        path to store gcode in (default "/tmp/Print2Go")
  -tty string
        tty to use (default "/dev/ttyUSB0")
```

Note that Print2Go only listens on `127.0.0.1` by default. You can tell Print2Go to listen on
all interfaces by running it via:

```
$ ./Print2Go -listen ':5001'
```


### With docker

1. After build, first time run the container, run detached and interactive terminal
You need to configure Map the usb device from host to container --device=<host_device_path>:<container_device_path>
`docker run -p 5001:5001 -d -it --device=/dev/ttyUSB0:/dev/ttyUSB0 --name go-container go-builder-linux-img:1.0`

Note: Mapping usb devices from windows as host to linux involves configuring WSL2, not the aim of this guide
For dev or other purposes can be run without mapping usb device
`docker run -p 5001:5001 -d -it --name go-container go-builder-linux-img:1.0`

2. Run Print2Go
`docker exec -it go-container /app/Print2Go -tty /dev/ttyUSB0 -listen 0.0.0.0:5001`

Note: Print2Go uses default ip 127.0.0.1 which is local only (cannot access from outside container), on the other hand docker uses 0.0.0.0 for exposing ports and services outside container, so we specify `-listen 0.0.0.0`, the port 5001 is the default of octoprint and can be changed (but need to change the docker file if you want another port)

### Other commands: 

Start a previously run container, with interactive shell (useful for build/running commands in the container)
`docker start -a -i go-container`

interactive shell access
`docker exec -it go-container bash`

Excecute command in container 
`docker exec -it go-container /app/myapp -tty /dev/ttyUSB0`

Run in OpWRT
`./Print2Go -tty <USBdevice> -listen <localip:port>`
Example
`./Print2Go -tty /dev/ttyUSB0 -listen 192.168.8.155:5001`

## Development Build (Win/Linux)

See [Readme-dev.md](/Readme-dev.md)

## Post print Script

Print2Go can be configured to run a command after the print finished. Eg. Automatic shutdown
By default, `/usr/lib/takoprint-shutdown.sh` will be executed (can be configured using the `-shudtown-script` flag).

The script could then execute a command to turn off a 'smart' power plug.
I'm using a Sonoff device running [TASMOTA](https://tasmota.github.io/docs/) in my setup with the following shutdown script:

```shell
$ cat /usr/lib/takoprint-shutdown.sh
#!/bin/bash

# retry 3x as my wifi connection can be crappy:
for x in 1 2 3 ; do
        until wget -O - -q "http://192.168.2.230/cm?cmnd=Power%20OFF"
        do
                echo "retry poweroff..."
                sleep 3
        done
        sleep 1
done
```

# Octoprint API Coverage

Print2Go implements the minimum required for printing of the Octoprint [API](https://docs.octoprint.org/en/main/api/index.html)
Some are just the endpoint with hardcoded response and no logic or validation. 
The endpoints are coded in [webapi.go](/webapi/webapi.go) and implemented in [octoapi.go](/webapi/octoapi.go)

# Autostart 

Acording to [instructions](https://openwrt.org/docs/guide-developer/procd-init-scripts), there is an example file included in the repo to make the script autostart at boot as a service. See: [Print2Go-srv](/Print2Go-srv), modify it acordingly to your configuration and ip.


# Future features and ideas

- [ ] Improve UI/UX
- [ ] Persist config
- [ ] Print file in storage
- [ ] Disable controls when printer is not connected
- [ ] Telegram Notification integration
- [ ] Homeassistant integration
- [x] Send custom commands
- [ ] Test two or more printers on same device
- [ ] WebCamStream
- [ ] Implement other OctoPrint API endpoints
- [ ] Update serial library
- [ ] Spread the word
      - [ ] Hackaday
      - [ ] Instructables

# Licence

GPLv3. See [licence file](/LICENSE.txt)


# Acknowledgement

Adrian Ulrich - Takoprint
https://git.sr.ht/~adrian-blx/takoprint
# PrintAndGo

3D printer host written in Go language
Convert your router in a 3D printer host with an Octoprint-like API
PrintAndGo is a lightweight and simple web based program written in Go to feed gcode to a 3d printer (aka gcode sender, 3d printer host)
It offers a convenient webinterface and mimics Octoprints upload API, meaning that common slicer software will be able to directly upload gcode to PrintAndGo.

It's based in the code of [Takoprint](https://git.sr.ht/~adrian-blx/takoprint) by[Adrian](https://github.com/adrian-bl)

## Features

- Written in Go and Multiplatform: Compile from source for your target device or use the release binaries, run it in your host device. Compatible with Windows and Linux operating systems. Compatible with many target hardware, if your target is supported in Go, it should be compatible 
- Aimed for lightweight: PrintAndGo doesn't need a lot of resources is only a 12MB binary (still not optimized)
- Low hardware resource requirement: Will work well even on older/or low specs hardware
- Can run in OpenWRT. This means you can use almost any router with usb port that supports OpenWRT or even in the Creality Wifi Box
- Compatible with any 3D printer with usb port that has some version of Marlin firmware (or accepts standard gcode over serial)
- Octoprint emulation: Mimics the basic Octoprint API allowing for direct Gcode upload from various slicers (Cura, PrusaSlicer)
- Custom hooks: PrintAndGo can execute custom scripts after your print is finished (eg. to turn off your printer).
- Simple: Web User interface with just the minimum required operations for printing
- OctoPrint minimal API implementation: Send custom Gcode, and upload file for inmediate printing

- Brief story: this project is related to [OctoWrt](https://github.com/shivajiva101/OctoWrt), and the addition of the Creality WifiBox (WB01) in [OpenWRT](https://github.com/openwrt/openwrt/pull/19686). OctoPrint is a great sofware and tool, but not very optimized, as is written in python, which is also great but nowadays requires more resources than when it first started, I believe that current code will barely run in a raspberry pi 3 (which was the original target hardware as far as I know). I had previously contributed in a Octoprint-like (or mimic) firmware for embedded devices as ESP8266 or ESP32 see [WirelessPrinting](https://github.com/probonopd/WirelessPrinting), but there was missing sofware solution for the intermediate hardware between embedded devices and mini-pc (like a Raspberry pi). So it needed some leverage, then I found [Takoprint](https://git.sr.ht/~adrian-blx/takoprint), and connected everything together


## Screenshots

![webinterface](/img/printandgo_home_gui.png)

## Build (Win/Linux)

A reasonably recent version of the Go compiler is required to build PrintAndGo (v1.25+) as September 2025

```shell
$ git clone https://url.to.this/repo
$ cd PrintAndGo
$ go build -o ./builds/PrintAndGo.exe .   # in case of windows
$ go build -o ./builds/PrintAndGo .       # in case of linux
```

If you want to cross compile :

```shell
$ CGO_ENABLED=0 GOARCH=arm64 go build . # Example for a raspberry Pi 3
$ GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o ./builds/PrintAndGo . # Example for MIPS Little Endian
```

## Build with docker (usually for dev and cross compile)

1. First build the docker image configured in the docker file, this will build the PrintAndGo code too
`docker build -t go-builder-linux-img:1.0 .`

2. After first time build, can be run

`docker run --rm -v ./:/usr/src/app -w /usr/src/PrintAndGo -e GOOS=linux -e GOARCH=mips -e GOMIPS=softfloat go-builder-linux-img:1.0 go build -v`

3. After first run, can be re-started, with interactive terminal
`docker start -a -i go-container`

4. In the interactive terminal 
`go build -o ./builds/PrintAndGo .`

5. Once build, copy files back to host (eg. after cross compile, get the binaries)
`docker cp go-container:/app/builds/PrintAndGo ./builds/PrintAndGo`

6. If you need to update the source, copy files from host to container (eg. developed new code and need to compile in docker), better to delete previous files if made a lot of changes
`docker cp ./* go-builder:/app/*`


### Cross compile

Go can be cross compiled
Example for MIPS Little Endian
`GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build .`

1.Build docker image with updated files, no cache if anything changed (for dev), dockerfile and docker image can be modified with cross compile commands
`docker build --no-cache -t go-dev-linux-img:1.0 -f ./Dockerfile_builder .`

Examples
`docker run --rm -v ./:/usr/src/app -w /usr/src/PrintAndGo -e GOOS=linux -e GOARCH=mips -e GOMIPS=softfloat go-builder-linux-img:1.0 go build -v`

Examples
`docker run -d -it --name go-builder go-dev-linux-img:1.0`

### Virtual printer

There is a virtual printer code used for testing in /util from the original Takoprint repo. 


## Run 

### Run in host machine

In a shell in windows or linux
```
./PrintAndGo  -tty <host_device_path>
```

Examples:
```
./PrintAndGo -tty COM3
./PrintAndGo -tty /dev/ttyUSB0
```

PrintAndGo is configured via flags. By default, PrintAndGo will listen on
`127.0.0.1:5001` and expect a printer on `/dev/ttyUSB0`:

```shell
$ ./PrintAndGo -h
Usage of ./PrintAndGo:
  -baud int
        baud rate of -port (default 115200)
  -gcode string
        file containing gcode
  -listen string
        ip:port to bind to (default "127.0.0.1:5001")
  -shutdown-script string
        script to execute to shutdown the printer (default "/usr/lib/printandgo-shutdown.sh")
  -storage string
        path to store gcode in (default "/tmp/printandgo")
  -tty string
        tty to use (default "/dev/ttyUSB0")
```

Note that PrintAndGo only listens on `127.0.0.1` by default. You can tell PrintAndGo to listen on
all interfaces by running it via:

```
$ ./printandgo -listen ':5001'
```


### With docker

1. After build, first time run the container, run detached and interactive terminal
You need to configure Map the usb device from host to container --device=<host_device_path>:<container_device_path>
`docker run -p 5001:5001 -d -it --device=/dev/ttyUSB0:/dev/ttyUSB0 --name go-container go-builder-linux-img:1.0`

Note: Mapping usb devices from windows as host to linux involves configuring WSL2, not the aim of this guide
For dev or other purposes can be run without mapping usb device
`docker run -p 5001:5001 -d -it --name go-container go-builder-linux-img:1.0`

2. Run PrintAndGo
`docker exec -it go-container /app/PrintAndGo -tty /dev/ttyUSB0 -listen 0.0.0.0:5001`

Note: PrintAndGo uses default ip 127.0.0.1 which is local only (cannot access from outside container), on the other hand docker uses 0.0.0.0 for exposing ports and services outside container, so we specify `-listen 0.0.0.0`, the port 5001 is the default of octoprint and can be changed (but need to change the docker file if you want another port)

### Other commands: 

Start a previously run container, with interactive shell (useful for build/running commands in the container)
`docker start -a -i go-container`

interactive shell access
`docker exec -it go-container bash`

Excecute command in container 
`docker exec -it go-container /app/myapp -tty /dev/ttyUSB0`

Run in OpWRT
`./PrintAndGo -tty <USBdevice> -listen <localip:port>`
Example
`./PrintAndGo -tty /dev/ttyUSB0 -listen 192.168.8.155:5001`

### Webcam

Webcam support is WIP, but should mostly work - as long as your webcam shows up on `/dev/video0`.

Note that certain webcams are extremely power hungry and can cause stability issues on RBPI hardware:

The excessive power draw of some webcams can cause issues to the USB controller, resulting in 'lost gcode' replies
which will cause the print to stall. This is *not* a bug in PrintAndGo: If you suffer from this issue, get a better
power supply or/and replace your webcam (or find other ways to feed power to it, like using an active USB hub).

Note that PrintAndGo has an escape hatch for stalled prints: Sending a `USR1` signal to PrintAndGo TTY subprocess
should resume the print in most cases:

```shell
$ ps -ef|grep :serial-pipe   # first, find the subprocess
takopri+   351   301  0 09:01 ?        00:00:00 PrintAndGo -tty /dev/ttyUSB0 -baud 115200 :serial-pipe
$ kill -USR1 351  # send SIGUSR1
```

### Post print Script

PrintAndGo can be configured to run a command after the print finished. Eg. Automatic shutdown
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

PrintAndGo implements the minimum required for printing of the Octoprint [API](https://docs.octoprint.org/en/main/api/index.html)
Some are just the endpoint with hardcoded response and no logic or validation. 
The endpoints are coded in [webapi.go](/webapi/webapi.go) and implemented in [octoapi.go](/webapi/octoapi.go)

# Autostart 

Acording to [instructions](https://openwrt.org/docs/guide-developer/procd-init-scripts), there is an example file included in the repo to make the script autostart at boot as a service. See: [PrintAndGo-srv](/PrintAndGo-srv), modify it acordingly to your configuration and ip.


# Planned features/idea

- [] Telegram Notification integration
- [] Homeassistant integration
- [] Send custom commands
- [] WebCamStream
- [] Implement other OctoPrint API endpoints


# Licence

GPLv3. See [licence file](/LICENSE.txt)


# Acknowledgement

Adrian Ulrich - Takoprint
https://git.sr.ht/~adrian-blx/takoprint
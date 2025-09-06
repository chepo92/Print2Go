# PrintAndGo

PrintAndGo is a simple programm to feed gcode to a 3d printer.
It offers a convenient webinterface and mimics Octoprints upload API, meaning that common slicer software
will be able to directly upload gcode to PrintAndGo.

Its based in the code from Takoprint

## Features

- Written in Go: Just push a single binary to your host device.
- Speed: PrintAndGo doesn't need a lot of resources and will work well even on older hardware.
- Octoprint emulation: Mimics the basic Octoprint API allowing for direct Gcode upload from various slicers.
- Custom hooks: PrintAndGo can execute custom scripts after your print finished (eg. to turn off your printer).

## Screenshots

![webinterface]()

## Build (Win/Linux)

A reasonably recent version of the Go compiler is required to build takoprint.

```shell
$ git clone https://url.to.this/repo
$ cd PrintAndGo
$ go build PrintAndGo.go
```

If you want to cross compile (example for a raspberry Pi 3):

```shell
$ CGO_ENABLED=0 GOARCH=arm64 go build ./cmd/PrintAndGo.go
```

## Build with docker (usually for dev and cross compile)

1. First build the docker image configured in the docker file, this will build the PrintAndGo code too
`docker build -t go-builder-linux:1.0 .`

1.alt Build updated files, no cache if anything changed (for dev) 
`docker build -t go-builder-linux:1.0 . --no-cache`


## Quick Configuration for running

PrintAndGo is configured via flags. By default, PrintAndGo will listen on
`127.0.0.1:5001` and expect a printer on `/dev/ttyUSB0`:

```shell
$ ./PrintAndGo -h
Usage of ./takoprint:
  -baud int
        baud rate of -port (default 115200)
  -gcode string
        file containing gcode
  -listen string
        ip:port to bind to (default "127.0.0.1:5001")
  -shutdown-script string
        script to execute to shutdown the printer (default "/usr/lib/takoprint-shutdown.sh")
  -storage string
        path to store gcode in (default "/tmp/takoprint")
  -tty string
        tty to use (default "/dev/ttyUSB0")
```

Note that PrintAndGo only listens on `127.0.0.1` by default. You can tell PrintAndGo to listen on
all interfaces by running it via:

```
$ ./printandgo -listen ':5001'
```

## Run 

### Run in host machine

In a shell in windows or linux
```
./printandgo  -tty <host_device_path>
```

Examples:
```
./printandgo -tty COM3
./printandgo -tty /dev/ttyUSB0
```

### With docker

1. After build, first time run the container, run detached and interactive terminal
You need to configure Map the usb device from host to container --device=<host_device_path>:<container_device_path>
`docker run -p 5001:5001 -d -it --device COM3:/dev/ttyUSB0 --name linux-go-builder go-builder-linux:1.0`

2. Run PrintAndGo
`docker exec -it linux-go-builder /app/printandgo -tty /dev/ttyUSB0 -listen 0.0.0.0:5001`

Note: PrintAndGo uses default ip 127.0.0.1 which is local only (cannot access from outside container), on the other hand docker uses 0.0.0.0 for exposing ports and services outside container, so we specify `-listen 0.0.0.0`, the port 5001 is the default of octoprint and can be changed (but need to change the docker file if you want another port)

Other commands: 

Start a previously run container
`docker start -a -i linux-go-builder`

Excecute command in container 
`docker exec -it linux-go-builder /app/myapp -tty /dev/ttyUSB0`

interactive shell access
`docker exec -it linux-go-builder bash`



### Webcam

Webcam support is WIP, but should mostly work - as long as your webcam shows up on `/dev/video0`.

Note that certain webcams are extremely power hungry and can cause stability issues on RBPI hardware:

The excessive power draw of some webcams can cause issues to the USB controller, resulting in 'lost gcode' replies
which will cause the print to stall. This is *not* a bug in Takoprint: If you suffer from this issue, get a better
power supply or/and replace your webcam (or find other ways to feed power to it, like using an active USB hub).

Note that Takoprint has an escape hatch for stalled prints: Sending a `USR1` signal to Takoprints TTY subprocess
should resume the print in most cases:

```shell
$ ps -ef|grep :serial-pipe   # first, find the subprocess
takopri+   351   301  0 09:01 ?        00:00:00 takoprint -tty /dev/ttyUSB0 -baud 115200 :serial-pipe
$ kill -USR1 351  # send SIGUSR1
```

### Automatic shutdown

Takoprint can be configured to run a command after the print finished.
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


# Licence

GPLv3. See licence file


# Acknowledgement

Adrian Ulrich - Takoprint
https://git.sr.ht/~adrian-blx/takoprint
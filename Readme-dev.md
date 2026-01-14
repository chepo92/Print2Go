## Build (Win/Linux)

A reasonably recent version of the Go compiler is required to build Print2Go (v1.25+) as September 2025

0. Clone project and cd to repo
```shell
$ git clone https://github.com/chepo92/Print2Go.git
$ cd Print2Go
```

1. Compile 
```shell
$ go build -o ./builds/Print2Go.exe .   # in case you are running in and building for Windows
$ go build -o ./builds/Print2Go .       # in case you are running in and building for Linux
```

If you want to cross compile:

```shell
$ GOOS=linux GOARCH=amd64 go build -o ./builds/Print2Go-Linux . # Example for Linux 
$ CGO_ENABLED=0 GOARCH=arm64 go build -o ./builds/Print2Go-ARM64_RPI3 . # Example for a raspberry Pi 3
$ GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o ./builds/Print2Go-MIPSle . # Example for MIPS Little Endian
$ GOOS=windows GOARCH=amd64  go build -o ./builds/Print2Go-Win64.exe . # Example for Windows 64 bit
$ GOOS=windows GOARCH=386  go build -o ./builds/Print2Go-Win32.exe . # Example for Windows 32 bit
```

## Build with docker (usually for dev and cross compile)

0. Clone project and cd to repo
```shell
$ git clone https://github.com/chepo92/Print2Go.git
$ cd Print2Go
```

1. First build the docker image configured in the docker file, this will build the Print2Go code too
`docker build -t go-builder:1.0 .`

2. After first time build, can be run

`docker run --rm -v ./:/usr/src/app -w /usr/src/Print2Go -e GOOS=linux -e GOARCH=mips -e GOMIPS=softfloat go-builder:1.0 go build -v`

3. After first run, can be re-started, with interactive terminal
`docker start -a -i go-builder`

4. In the interactive terminal (IT), build the Print2Go project (use a the cross compile command from above if the case): 
`go build -o ./builds/Print2Go .`

5. Once build, copy files back to host (eg. after cross compile, get the binaries), run i nthe host: 
Copy one file
`docker cp go-builder:/app/builds/Print2GoLinux ./builds/Print2GoLinux`
Copy all folder
`docker cp go-builder:/app/builds/ .`

6. If you need to update the source, copy files from host to container (eg. developed new code and need to compile in docker), better to delete previous files if made a lot of changes
In the container (IT), remove existing app files: 
`cd /app/ && rm -r *`
Then in the **host**, copy to container: 
`docker cp ./ go-builder:/app/`
Then, back in the container, you can build (step 4) and copy back to host (step 5)

### Cross compile

Go can be cross compiled
Example for MIPS Little Endian
`GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o ./builds/Print2Go_mipsle .`

## Build with docker
1.Build docker image with updated files, no cache if anything changed (for dev), dockerfile and docker image can be modified with cross compile commands
`docker build --no-cache -t go-dev-linux-img:1.0 -f ./Dockerfile_builder .`

Examples
`docker run --rm -v ./:/usr/src/app -w /usr/src/Print2Go -e GOOS=linux -e GOARCH=mips -e GOMIPS=softfloat go-builder-linux-img:1.0 go build -v`

Examples
`docker run -d -it --name go-builder go-dev-linux-img:1.0`


### Virtual printer

There is a virtual printer code used for testing in /util from the original Takoprint repo. 


## Webcam

Webcam support is WIP, but should mostly work - as long as your webcam shows up on `/dev/video0`.

Note that certain webcams are extremely power hungry and can cause stability issues on RBPI hardware:

The excessive power draw of some webcams can cause issues to the USB controller, resulting in 'lost gcode' replies
which will cause the print to stall. This is *not* a bug in Print2Go: If you suffer from this issue, get a better
power supply or/and replace your webcam (or find other ways to feed power to it, like using an active USB hub).

Note that Print2Go has an escape hatch for stalled prints: Sending a `USR1` signal to Print2Go TTY subprocess
should resume the print in most cases:

```shell
$ ps -ef|grep :serial-pipe   # first, find the subprocess
takopri+   351   301  0 09:01 ?        00:00:00 Print2Go -tty /dev/ttyUSB0 -baud 115200 :serial-pipe
$ kill -USR1 351  # send SIGUSR1
```
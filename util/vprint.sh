#!/bin/sh

socat PTY,link=/tmp/virtual-tty,raw,echo=0 PTY,link=/tmp/virtual-tty-2,raw,echo=0 &

go run vprinter.go

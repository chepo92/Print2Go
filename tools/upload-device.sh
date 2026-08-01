#!/bin/bash

set -e


if [ -z "$1" ]; then
    echo "Usage: $0 <device_ip>"
    exit 1
fi


DEVICE_IP=$1
BINARY="dist/Print2Go_mipsle"


if [ ! -f "$BINARY" ]; then
    echo "Binary not found: $BINARY"
    exit 1
fi


echo "Uploading to ${DEVICE_IP}..."


curl.exe \
-v \
-F "file=@${BINARY}" \
"http://${DEVICE_IP}:5001/print2go/update"


echo
echo "Done: ${DEVICE_IP}"
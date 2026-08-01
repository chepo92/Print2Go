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


HTTP_CODE=$(curl.exe \
    -s \
    -o /tmp/print2go-update-response.txt \
    -w "%{http_code}" \
    -F "file=@${BINARY}" \
    "http://${DEVICE_IP}:5001/print2go/update"
)


echo "HTTP ${HTTP_CODE}"


if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "204" ]; then

    echo "Update failed on ${DEVICE_IP}"
    cat /tmp/print2go-update-response.txt
    exit 1

fi


cat /tmp/print2go-update-response.txt

echo
echo "Done: ${DEVICE_IP}"
#!/bin/bash

set -e


DEVICES_FILE="devices.txt"


if [ ! -f "$DEVICES_FILE" ]; then
    echo "Missing $DEVICES_FILE"
    exit 1
fi


./tools/build-mips.sh

while IFS= read -r IP || [ -n "$IP" ]
do

    IP=$(echo "$IP" | tr -d '\r')
    # skip empty lines
    [ -z "$IP" ] && continue

    echo
    echo "============================="
    echo "Updating ${IP}"
    echo "============================="


    if ./tools/upload-device.sh "$IP"; then
        echo "SUCCESS ${IP}"
    else
        echo "FAILED ${IP}"
    fi


done < "$DEVICES_FILE"


echo
echo "Fleet update finished"
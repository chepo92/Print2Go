#!/bin/bash

set -e


if [ -z "$1" ]; then
    echo "Usage: $0 <device_ip>"
    exit 1
fi


DEVICE_IP=$1


./tools/build-mips.sh


./tools/upload-device.sh ${DEVICE_IP}
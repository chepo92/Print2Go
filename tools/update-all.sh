#!/bin/bash

set -e


if [ $# -eq 0 ]; then
    echo "Usage:"
    echo "$0 <ip1> <ip2> ..."
    exit 1
fi


./tools/build-mips.sh


for IP in "$@"
do

    echo
    echo "============================="
    echo "Updating ${IP}"
    echo "============================="

    if ./tools/upload-device.sh ${IP}; then
        echo "SUCCESS ${IP}"
    else
        echo "FAILED ${IP}"
    fi

done


echo
echo "All updates finished"
#!/bin/bash

set -e


CONTAINER="print2go-dev"
IMAGE="print2go-builder"

DEVICE_IP="192.168.1.112"
UPDATE_URL="http://${DEVICE_IP}:5001/print2go/update"


echo "== Checking docker container =="


if [ "$(docker ps -aq -f name=^${CONTAINER}$)" = "" ]; then

    echo "Container does not exist, creating..."

    docker run -d \
        --name ${CONTAINER} \
        -v "${PWD}:/workspace" \
        -w /workspace/src \
        ${IMAGE} \
        sleep infinity

fi


if [ "$(docker ps -q -f name=^${CONTAINER}$)" = "" ]; then

    echo "Starting container..."

    docker start ${CONTAINER}

fi



echo "== Building MIPS binary =="


docker exec ${CONTAINER} sh -c \
"GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
go build -o ../dist/Print2Go_mipsle"



echo "== Uploading firmware =="


curl.exe \
-v \
-F "file=@dist/Print2Go_mipsle" \
${UPDATE_URL}



echo
echo "== Update complete =="
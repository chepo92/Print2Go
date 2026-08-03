#!/bin/bash

set -e

CONTAINER="print2go-dev"

echo "== Building MIPS =="

docker exec ${CONTAINER} sh -c \
"GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
go build -o ../dist/Print2Go_mipsle"

echo "== Build finished =="

ls -lh dist/Print2Go_mipsle
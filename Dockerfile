# Choose our Go base image for compiling
FROM golang:1.25

# Select architecture, OS, and ARM version if applicable
# ARG allows passing variables to the Dockerfile at build time
# ARG BUILD_OS=linux
# ARG BUILD_ARCH=amd64
# ARG BUILD_ARM=7

# Set Go environment variables
# ENV GOOS=$BUILD_OS
# ENV GOARCH=$BUILD_ARCH
# ENV GOARM=$BUILD_ARM

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container
COPY . .

# Compile
# RUN go build .

RUN go build -v -o /usr/local/bin/app/ ./...

CMD ["app"]
# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker's layer caching
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 disables CGO, creating a statically linked binary
# GOOS=linux ensures the binary is built for a Linux environment
# -o specifies the output file name
RUN CGO_ENABLED=0 GOOS=linux go build -o Print2Go .


# Stage 2: Create the final, minimal image
FROM alpine:latest

# Set the working directory in the final image
WORKDIR /app

# Install necessary runtime dependencies (e.g., CA certificates for HTTPS)
RUN apk --no-cache add ca-certificates tzdata

# Copy the built binary from the builder stage
COPY --from=builder /app/Print2Go .

# Expose the port your application listens on (if applicable)
EXPOSE 5001

# Set the entrypoint command to run the application
#CMD ["/app/myapp "]
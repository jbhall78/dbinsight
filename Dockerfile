# Use a suitable base image
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum if they exist
COPY go.mod go.sum ./

COPY go-mysql/ .

# Download dependencies (cache layer)
RUN go mod download

# Copy the source code
COPY . .

# Build the proxy binary
WORKDIR /app/cmd/proxy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build . 

# Create a smaller final image (multi-stage build)
FROM alpine:latest

WORKDIR /app

# Copy only the necessary files from the builder stage
RUN ls -l /app
COPY --from=builder /app/cmd/proxy/proxy /app/dbinsight-proxy
COPY --from=builder /app/data/config/proxy.yaml /app/data/config/proxy.yaml
RUN ls -l /app

# Set the entrypoint to run the proxy
ENTRYPOINT ["/app/dbinsight-proxy"]

# Expose the port if needed
EXPOSE 3306

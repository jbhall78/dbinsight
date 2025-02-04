# syntax=docker/dockerfile:1.4

# Use a suitable base image
FROM golang:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum if they exist
COPY go.mod go.sum ./

# Copy the forked library (crucial to do this BEFORE copying your code)
COPY go-mysql/ /app/go-mysql/

# Copy your application source code (cmd and data)
COPY cmd/ /app/cmd/
COPY data/ /app/data/

# Copy the source code
COPY cmd/ .
COPY data/ .

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

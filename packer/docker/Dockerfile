FROM golang:1.18-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o proxy ./cmd/proxy/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/proxy /app/proxy
COPY data /app/data  # Copy the data directory
CMD ["./proxy"]

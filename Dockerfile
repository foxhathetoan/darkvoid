# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/darkvoid ./cmd/api

# Stage 2: Runtime
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/darkvoid .

# Uploads directory for local storage provider
RUN mkdir -p /app/uploads

EXPOSE 8080

ENTRYPOINT ["/app/darkvoid"]

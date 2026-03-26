# ---------- Build Stage ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api ./cmd/api
RUN go build -o worker ./cmd/worker

# ---------- Runtime Stage ----------
FROM alpine:latest

WORKDIR /root/

RUN apk add --no-cache git

COPY --from=builder /app/api .
COPY --from=builder /app/worker .

COPY .jennings .jennings

EXPOSE 8080
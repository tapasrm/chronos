# Stage 1: Build Go app
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache build-base gcc sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o chronos main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache sqlite

WORKDIR /app
COPY --from=builder /app/chronos .
COPY --from=builder /app/storage ./storage
COPY --from=builder /app/backup ./backup
COPY --from=builder /app/cronmgr ./cronmgr

EXPOSE 8080
CMD ["./chronos"]


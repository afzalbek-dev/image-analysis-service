# Multi-stage Docker build for Image Analysis Service
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/server /app/server
COPY .env.example /app/.env.example

EXPOSE 8080

USER nobody:nobody

ENTRYPOINT ["/app/server"]

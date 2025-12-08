# Stage 1: Builder
FROM golang:1.24.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY migrations /app/migrations/
RUN go mod download
COPY . .
RUN go build -o gophkeeper ./cmd/server

# Stage 2: Final
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gophkeeper .
COPY --from=builder /app/go.mod .
COPY --from=builder /app/.env .
COPY --from=builder /app/migrations ./migrations
CMD ["./gophkeeper"]
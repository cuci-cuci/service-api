## Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./cmd/server

## Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["sh", "-c", "echo 'ENV CHECK: DATABASE_URL set='$(test -n \"$DATABASE_URL\" && echo yes || echo no) && echo 'ENV CHECK: JWT_SECRET set='$(test -n \"$JWT_SECRET\" && echo yes || echo no) && ./server"]

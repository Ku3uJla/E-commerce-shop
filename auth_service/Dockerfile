# --- build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service ./cmd/app

# --- runtime stage ---
FROM alpine:3.20

RUN adduser -D -u 10001 app

COPY --from=builder /out/auth-service /usr/local/bin/auth-service

USER app

EXPOSE 8080

ENTRYPOINT ["auth-service"]
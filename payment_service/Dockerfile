# --- build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/payment-service ./cmd/app

# --- runtime stage ---
FROM alpine:3.20

RUN adduser -D -u 10001 app

COPY --from=builder /out/payment-service /usr/local/bin/payment-service

USER app

EXPOSE 8083

ENTRYPOINT ["payment-service"]
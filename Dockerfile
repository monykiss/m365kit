FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/klytics/m365kit/cmd/version.Version=${VERSION}" \
    -o /bin/kit .

# --- Runtime ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/kit /usr/local/bin/kit

# Create non-root user
RUN adduser -D -h /home/kit kit
USER kit
WORKDIR /home/kit

# Default config directory
RUN mkdir -p /home/kit/.kit

ENTRYPOINT ["kit"]
CMD ["--help"]

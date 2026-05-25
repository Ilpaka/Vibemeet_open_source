# syntax=docker/dockerfile:1.6

# --- Build stage ----------------------------------------------------------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# CGO toolchain + native libs for the WebRTC media path (libvpx, opus).
RUN apk add --no-cache git gcc g++ pkgconfig libvpx-dev opus-dev

# Cache module downloads independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 \
    GOOS=linux \
    CGO_CFLAGS="-I/usr/include" \
    CGO_LDFLAGS="-L/usr/lib"

RUN go build -trimpath -ldflags="-s -w" -o server ./cmd/server

# --- Runtime stage --------------------------------------------------------
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata netcat-openbsd wget libvpx opus

# Some libvpx releases bump SONAME; provide a compatibility symlink so the
# binary keeps loading when Alpine ships a newer library version.
RUN cd /usr/lib && \
    if ls libvpx.so.* 1> /dev/null 2>&1 && [ ! -f libvpx.so.9 ]; then \
        LATEST=$(ls -1 libvpx.so.* | head -1) && \
        ln -sf "$(basename "$LATEST")" libvpx.so.9 && \
        ln -sf "$(basename "$LATEST")" libvpx.so; \
    fi && \
    ldconfig /etc/ld.so.conf.d || true

WORKDIR /app
COPY --from=builder /app/server .
COPY docker-entrypoint.sh .
RUN sed -i 's/\r$//' docker-entrypoint.sh && chmod +x docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]

# Stage 1: Build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /vizhi-server ./cmd/vizhi-server

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 1000 vizhi \
    && adduser -S -u 1000 -G vizhi vizhi

COPY --from=builder /vizhi-server /usr/local/bin/vizhi-server

RUN mkdir -p /data/uploads /data/tls \
    && chown -R vizhi:vizhi /data

USER vizhi
EXPOSE 8443

ENTRYPOINT ["vizhi-server"]

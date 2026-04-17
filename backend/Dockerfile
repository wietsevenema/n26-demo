FROM golang:1.25 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates procps && \
    rm -rf /var/lib/apt/lists/*

# Create a non-privileged user
RUN useradd -m next26
USER next26

COPY --from=builder /app/server /server
CMD ["/server"]

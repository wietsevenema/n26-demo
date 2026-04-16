# Use the official Golang image to build the binary.
FROM golang:1.25 as builder

WORKDIR /app

# Copy the source code.
COPY main.go ./

# Since we are in a sub-directory, we'll initialize a module if needed, 
# but the easiest way is to use the parent's module if they were in the same tree.
# For simplicity, we'll initialize a local module here.
RUN go mod init cleanup && \
    go get cloud.google.com/go/firestore google.golang.org/api/iterator && \
    go mod tidy

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server

# Use a small alpine image for the runtime.
FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary from the builder.
COPY --from=builder /app/server /server

# Run the web service on container startup.
CMD ["/server"]

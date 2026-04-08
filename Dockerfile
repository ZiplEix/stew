# STEP 1: Build Stage
FROM golang:1.25-bookworm AS builder

# Install system dependencies
RUN apt-get update && apt-get install -y wget ca-certificates dpkg

# Install TinyGo 0.40.1 as explicitly requested by USER
RUN wget https://github.com/tinygo-org/tinygo/releases/download/v0.40.1/tinygo_0.40.1_amd64.deb && \
    dpkg -i tinygo_0.40.1_amd64.deb && \
    rm tinygo_0.40.1_amd64.deb

WORKDIR /src

# Copy full repository for building both the CLI and the documentation
COPY . .

# 1. Build the Stew CLI from the root
RUN go build -o /usr/local/bin/stew .

# 2. Prepare the documentation project
WORKDIR /src/doc

# Ensure all dependencies are clean and in sync for the build
RUN go mod download
RUN stew clean
RUN stew compile
RUN stew generate
RUN go mod tidy

# 3. Build the final server binary
RUN go build -ldflags="-w -s" -o server .

# STEP 2: Runner Stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates libc6-compat

WORKDIR /app

# Copy the server binary and the generated static assets (including wasm)
COPY --from=builder /src/doc/server /app/server
COPY --from=builder /src/doc/static /app/static

# Documentation server uses 8080 by default
EXPOSE 8080

CMD ["./server"]

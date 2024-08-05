# This file is used to build the docker image for the Go Agent's GitHub Action tests
# Default go version if no arguments passed in
ARG GO_VERSION=1.20

# Takes in go version
FROM golang:${GO_VERSION} as builder

# Set working directory and run go mod tidy
WORKDIR /app
# Copy source code files
COPY . .

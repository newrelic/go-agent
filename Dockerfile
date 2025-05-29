# This file is used to build the docker image for the Go Agent's GitHub Action tests
ARG GO_VERSION

# Takes in go version
FROM golang:${GO_VERSION:-1.24}

# Set working directory and run go mod tidy
WORKDIR /usr/src/app

# Avoid "fatal: detected dubious ownership in repository at 'usr/src/app/'" error
# when running git commands inside container with host volume mounted:
RUN git config --global --add safe.directory /usr/src/app/
CMD ["bash"]

ARG ALPINE_VERSION=3.8
ARG GO_VERSION=1.11.4

# base image
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as lint

RUN apk add --no-cache \
  curl \
  git \
  make \
  coreutils
RUN go get github.com/alecthomas/gometalinter && gometalinter --install

WORKDIR /go/src/github.com/docker/cnab-to-oci
ENV CGO_ENABLED=0

COPY . .

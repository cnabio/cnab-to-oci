ARG ALPINE_VERSION=3.10
ARG GO_VERSION=1.13.0

# build image
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as build

ARG DOCKERCLI_VERSION=19.03.1
ARG DOCKERCLI_CHANNEL=stable

ARG BUILDTIME
ARG COMMIT
ARG TAG
ARG GOPROXY

RUN apk add --no-cache \
  bash \
  make \
  git \
  curl \
  util-linux \
  coreutils \
  build-base

# Fetch docker cli to run a registry container for e2e tests
RUN curl -Ls https://download.docker.com/linux/static/${DOCKERCLI_CHANNEL}/x86_64/docker-${DOCKERCLI_VERSION}.tgz | tar -xz

WORKDIR /go/src/github.com/docker/cnab-to-oci
COPY . .
RUN make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG bin/cnab-to-oci &&\
  make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG build-e2e-test

# e2e image
FROM alpine:${ALPINE_VERSION} as e2e

# copy all the elements needed for e2e tests from build image
COPY --from=build /go/docker/docker /usr/bin/docker
COPY --from=build /go/src/github.com/docker/cnab-to-oci/bin/cnab-to-oci /usr/bin/cnab-to-oci
COPY --from=build /go/src/github.com/docker/cnab-to-oci/e2e /e2e
COPY --from=build /go/src/github.com/docker/cnab-to-oci/e2e.test /e2e/e2e.test

# Run end-to-end tests
CMD ["e2e/run.sh"]

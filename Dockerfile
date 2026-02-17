ARG ALPINE_VERSION=3.22
ARG GO_VERSION=1.25.0

# build image
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build

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

WORKDIR /go/src/github.com/cnabio/cnab-to-oci
COPY . .
RUN make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG bin/cnab-to-oci &&\
  make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG build-e2e-test

# e2e image
FROM alpine:${ALPINE_VERSION} AS e2e

RUN apk add --no-cache docker

# copy all the elements needed for e2e tests from build image
COPY --from=build /go/src/github.com/cnabio/cnab-to-oci/bin/cnab-to-oci /usr/bin/cnab-to-oci
COPY --from=build /go/src/github.com/cnabio/cnab-to-oci/e2e /e2e
COPY --from=build /go/src/github.com/cnabio/cnab-to-oci/e2e.test /e2e/e2e.test

# Run end-to-end tests
CMD ["e2e/run.sh"]

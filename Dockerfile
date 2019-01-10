ARG ALPINE_VERSION=3.8
ARG GO_VERSION=1.11.4

# build image
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as build

ARG DOCKERCLI_VERSION=18.03.1-ce
ARG DOCKERCLI_CHANNEL=edge
ARG DOCKER_APP_VERSION=cnab-dockercon-preview

ARG BUILDTIME
ARG COMMIT
ARG TAG

RUN apk add --no-cache \
  bash \
  make \
  git \
  curl \
  util-linux \
  coreutils \
  build-base

# Fetch docker cli to run a registry container for e2e tests
RUN curl -Ls https://download.docker.com/linux/static/$DOCKERCLI_CHANNEL/x86_64/docker-$DOCKERCLI_VERSION.tgz | tar -xz

# Fetch docker-app to build a CNAB from an application template
RUN curl -Ls https://github.com/docker/app/releases/download/$DOCKER_APP_VERSION/docker-app-linux.tar.gz | tar -xz
RUN git clone https://github.com/docker/app

WORKDIR /go/src/github.com/docker/cnab-to-oci
COPY . .
RUN make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG bin/cnab-to-oci &&\
    make BUILDTIME=$BUILDTIME COMMIT=$COMMIT TAG=$TAG build-e2e-test

# e2e image
FROM alpine:${ALPINE_VERSION} as e2e

# copy all the elements needed for e2e tests from build image
COPY --from=build /go/docker/docker /usr/bin/docker
COPY --from=build /go/docker-app-linux /usr/bin/docker-app
COPY --from=build /go/app/examples /examples
COPY --from=build /go/src/github.com/docker/cnab-to-oci/bin/cnab-to-oci /usr/bin/cnab-to-oci
COPY --from=build /go/src/github.com/docker/cnab-to-oci/e2e /e2e
COPY --from=build /go/src/github.com/docker/cnab-to-oci/e2e.test /e2e/e2e.test

# Run end-to-end tests
CMD ["e2e/run.sh"]

all: binary test

binary:
	go build github.com/docker/cnab-to-oci/cmd/cnab-to-oci

.PHONY: test
test:
	go test ./...
module github.com/cnabio/cnab-to-oci

go 1.13

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78
	github.com/Masterminds/semver v1.5.0
	github.com/Microsoft/go-winio v0.4.14
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5
	github.com/beorn7/perks v1.0.1
	github.com/cnabio/cnab-go v0.8.2-beta1
	github.com/containerd/containerd v1.3.0
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808
	github.com/docker/cli v0.0.0-20191017083524-a8ff7f821017
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20181229214054-f76d6a078d88
	github.com/docker/docker-credential-helpers v0.6.3
	github.com/docker/go v1.5.1-1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82
	github.com/docker/go-units v0.4.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.0
	github.com/gorilla/mux v1.7.3
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v0.1.1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/prometheus/procfs v0.0.5
	github.com/qri-io/jsonpointer v0.1.0
	github.com/qri-io/jsonschema v0.1.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190830141801-acfa387b8d69
	google.golang.org/genproto v0.0.0-20190522204451-c2c4e71fbf69
	google.golang.org/grpc v1.22.1
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309

[![Documentation](https://godoc.org/github.com/docker/cnab-to-oci/remotes?status.svg)](http://godoc.org/github.com/docker/cnab-to-oci/remotes)

# CNAB to OCI

The intent of CNAB to OCI is to propose a library for sharing a CNAB bundle on an OCI or Docker registry.

## Getting Started

To get and build the project:

```sh
$ go get -u github.com/docker/cnab-to-oci
$ cd $GOPATH/src/github.com/docker/cnab-to-oci
$ make
```
By now you should have a binary of the project in the `bin` folder. To run it, execute:

```sh
$ bin/cnab-to-oci --help
```

### Prerequisites

```
- Make
- Golang 1.9+
- Git
```

### Installing

For installing, make sure your `$GOPATH/bin` makes part of the `$PATH`

```sh
$ make install
```

This will build and install `cnab-to-oci` into `$GOPATH/bin`

### Usage
The `cnab-to-oci` binary is a demonstration tool to `push` and `pull` a bundle to a registry. It comes with 3 commands: `push`, `pull` and `fixup`.

#### Pull

```sh
$ bin/cnab-to-oci pull <docker-ref>
```

#### Push

```sh
$ bin/cnab-to-oci push <bundle file>
```

#### Fixup
The fixup command resolves all the digest references from a registry and patches the bundle.json with them.

```sh
$ bin/cnab-to-oci fixup bundle.json
```

Where `bundle.json` is the bundle file to be fixed up. The result of a successful execution has as default output file `fixed-bundle.json`

### Example

Example of an OCI Index Descriptor sent to the registry

```json
{
  "schemaVersion": 1,
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 315,
      "annotations": {
        "io.cnab.type": "config"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:196d12cf6ab19273823e700516e98eb1910b03b17840f9d5509f03858484d321",
      "size": 506,
      "annotations": {
        "io.cnab.type": "invocation"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:6bb891430fb6e2d3b4db41fd1f7ece08c5fc769d8f4823ec33c7c7ba99679213",
      "size": 507,
      "annotations": {
        "io.cnab.component_name": "image-1",
        "io.cnab.original_name": "nginx:2.12",
        "io.cnab.type": "component"
      }
    }
  ],
  "annotations": {
    "io.cnab.keywords": "[\"keyword1\",\"keyword2\"]",
    "io.cnab.runtime_version": "v1.0.0-WD",
    "io.docker.app.format": "cnab",
    "io.docker.type": "app",
    "org.opencontainers.image.authors": "[{\"name\":\"docker\",\"email\":\"docker@docker.com\",\"url\":\"docker.com\"}]",
    "org.opencontainers.image.description": "description",
    "org.opencontainers.image.title": "my-app",
    "org.opencontainers.image.version": "0.1.0"
  }
}
```

## Development

### Running the tests

```sh
$ make test
```

### Running the e2e tests

```sh
$ make e2e
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Maintainers

See also the list of [maintainers](MAINTAINERS) who participated in this project.

## Contributors

See also the list of [contributors](https://github.com/docker/cnab-to-oci/graphs/contributors) who participated in this project.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

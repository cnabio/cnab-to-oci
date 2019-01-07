[![Documentation](https://godoc.org/github.com/docker/cnab-to-oci/remotes?status.svg)](http://godoc.org/github.com/docker/cnab-to-oci/remotes)

# CNAB to OCI

The intent of CNAB to OCI is to propose a library for sharing a CNAB using an
OCI or Docker registry.

## Getting Started

To get and build the project:

```console
$ go get -u github.com/docker/cnab-to-oci
$ cd $GOPATH/src/github.com/docker/cnab-to-oci
$ make
```

You should now have a `cnab-to-oci` binary the `bin/` folder. To run it,
execute:

```console
$ bin/cnab-to-oci --help
```

### Prerequisites

- Make
- Golang 1.9+
- Git

### Installing

For installing, make sure your `$GOPATH/bin` is part of your `$PATH`.

```console
$ make install
```

This will build and install `cnab-to-oci` into `$GOPATH/bin`.

### Usage

The `cnab-to-oci` binary is a demonstration tool to `push` and `pull` a CNAB
to a registry. It has three commands: `push`, `pull` and `fixup` which are
described in the following sections.

#### Push

The `push` command packages a `bundle.json` file into an OCI image index
(falling back to a Docker manifest if the registry does not support this) and
pushes this to the registry. As part of this process, [`fixup`](#fixup) process
is implicitly run.

```console
$ bin/cnab-to-oci push examples/helloworld-cnab/bundle.json --target myhubusername/repo
Ensuring image cnab/helloworld:0.1.1 is present in repository docker.io/myhubusername/repo
Image is not present in repository
Mounting descriptor sha256:bbffe37bb3899b1384bf1483cdcff44bd148d52078b4655e69cd23d534ea043d with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 203)
Mounting descriptor sha256:e280b57a032b8bb2ab45f26ea67f42b5d47fd5aca2dd63c5bcdbbd1f753f20b7 with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 370)
Mounting descriptor sha256:8e3ba11ec2a2b39ab372c60c16b421536e50e5ce64a0bc81765c2e38381bcff6 with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 2206542)
Mounting descriptor sha256:58e6f39290459b6563b348052b2a1a8cf2a44fac19a80ae0da36c82a32f151f8 with media type application/vnd.docker.container.image.v1+json (size: 2135)
Copying descriptor sha256:a59a4e74d9cc89e4e75dfb2cc7ea5c108e4236ba6231b53081a9e2506d1197b6 with media type application/vnd.docker.distribution.manifest.v2+json (size: 942)
{"errors":[{"code":"MANIFEST_INVALID","message":"manifest invalid","detail":{}}]}

Pushed successfully, with digest "sha256:6cabd752cb01d2efb9485225baf7fc26f4322c1f45f537f76c5eeb67ba8d83e0"
```

**Note:** `cnab-to-oci` does not push images from your docker daemon image store.
Make sure all your invocation images are already present on a registry before pushing your bundle.

**Note:** The `MANIFEST_INVALID` error in the above case is because the Docker Hub
does not currently support the OCI image index type.

**Note**: When using the Docker Hub, no tag will show up in the Hub interface.
The artifact must be referenced by its SHA - see [`pull`](#pull).

#### Pull

The `pull` command is used to fetch a CNAB packaged as an OCI image index or
Docker manifest from a registry. This must be done using the digest returned by
the [`push`](#push) command. By default the output is saved to `pulled.json`.

```console
$ bin/cnab-to-oci pull myhubusername/repo@sha256:6cabd752cb01d2efb9485225baf7fc26f4322c1f45f537f76c5eeb67ba8d83e0

$ cat pulled.json
{
 "name": "helloworld",
 "version": "0.1.1",
 "description": "A short description of your bundle",
 "keywords": [
  "helloworld",
  "cnab",
  "tutorial"
 ],
 "maintainers": [
  {
   "name": "Jane Doe",
   "email": "jane.doe@example.com",
   "url": "https://example.com"
  }
 ],
 "invocationImages": [
  {
   "imageType": "docker",
   "image": "myhubusername/repo@sha256:a59a4e74d9cc89e4e75dfb2cc7ea5c108e4236ba6231b53081a9e2506d1197b6",
   "size": 942,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json"
  }
 ],
 "images": null,
 "parameters": null,
 "credentials": null
}
```

#### Fixup

The `fixup` command resolves all the image digest references (for the
_invocationImages_ as well as the _images_ in the `bundle.json`) from the
relevant registries and pushes them to the _target_ repository to ensure they're
available to anyone who has access to the CNAB in the target repository. A
patched `bundle.json` is saved by default to `fixed-bundle.json`

```console
$ bin/cnab-to-oci fixup examples/helloworld-cnab/bundle.json --target myhubusername/repo
Ensuring image cnab/helloworld:0.1.1 is present in repository docker.io/myhubusername/repo
Image is not present in repository
Mounting descriptor sha256:bbffe37bb3899b1384bf1483cdcff44bd148d52078b4655e69cd23d534ea043d with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 203)
Mounting descriptor sha256:e280b57a032b8bb2ab45f26ea67f42b5d47fd5aca2dd63c5bcdbbd1f753f20b7 with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 370)
Mounting descriptor sha256:8e3ba11ec2a2b39ab372c60c16b421536e50e5ce64a0bc81765c2e38381bcff6 with media type application/vnd.docker.image.rootfs.diff.tar.gzip (size: 2206542)
Mounting descriptor sha256:58e6f39290459b6563b348052b2a1a8cf2a44fac19a80ae0da36c82a32f151f8 with media type application/vnd.docker.container.image.v1+json (size: 2135)
Copying descriptor sha256:a59a4e74d9cc89e4e75dfb2cc7ea5c108e4236ba6231b53081a9e2506d1197b6 with media type application/vnd.docker.distribution.manifest.v2+json (size: 942)

$ cat fixed-bundle.json
{
  "name": "helloworld",
  "version": "0.1.1",
  "description": "A short description of your bundle",
  "keywords": [
    "helloworld",
    "cnab",
    "tutorial"
  ],
  "maintainers": [
    {
      "name": "Jane Doe",
      "email": "jane.doe@example.com",
      "url": "https://example.com"
    }
  ],
  "invocationImages": [
    {
      "imageType": "docker",
      "image": "myhubusername/repo@sha256:a59a4e74d9cc89e4e75dfb2cc7ea5c108e4236ba6231b53081a9e2506d1197b6",
      "size": 942,
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json"
    }
  ],
  "images": null,
  "parameters": null,
  "credentials": null
}
```

**Note:** In the above example, the invocation image reference now matches the
target repository.

### Example

The following is an example of an OCI image index sent to the registry.

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

```console
$ make test
```

### Running the e2e tests

```console
$ make e2e
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of
conduct, and the process for submitting pull requests to us.

## Maintainers

See also the list of [maintainers](MAINTAINERS) who participated in this
project.

## Contributors

See also the list of
[contributors](https://github.com/docker/cnab-to-oci/graphs/contributors) who
participated in this project.

## License

This project is licensed under the Apache 2 License - see the [LICENSE](LICENSE)
file for details.

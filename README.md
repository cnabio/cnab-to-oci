[![Documentation](https://godoc.org/github.com/docker/cnab-to-oci/remotes?status.svg)](http://godoc.org/github.com/docker/cnab-to-oci/remotes)

# CNAB to OCI

The intent of CNAB to OCI is to propose a reference implementation for sharing a
CNAB using an OCI or Docker registry.

[Jump to the example](#example).

## Rationale for this approach

Goals:
- Package the information from a CNAB [`bundle.json`](https://github.com/deislabs/cnab-spec/blob/master/101-bundle-json.md) into a format that can be stored in container registries.
- Require no or only minor changes to the [OCI specification](https://github.com/opencontainers/image-spec).
    - Major changes would take long to get approved.
    - Anything that diverges from the current specification will require coordination with registries to ensure compatibility.
- Store all container images required for the CNAB in the same repository and reference them from the same manifest.
    - If a user can access the CNAB, they can access all the parts needed to install it.
    - Moving a CNAB from one repository to another is atomic.
- Ensure that registries can reason over these CNABs.
    - Provide enough information for registries to understand how to present these artifacts.

Non-goals:
- A perfectly clean solution.
    - The authors acknowledge that there is a tension between getting something working today and the ideal solution.

### Selection of OCI index

The CNAB specification references a
[list of invocation images](https://github.com/deislabs/cnab-spec/blob/master/101-bundle-json.md#invocation-images)
and a
[map of other images](https://github.com/deislabs/cnab-spec/blob/master/101-bundle-json.md#the-image-map).
An [OCI index](#what-is-an-oci-index) is already used for handling multiple
images so this was seen as a natural fit.

The only disadvantage of the OCI index is its lack of a top-level
mechanism for communicating type which may make the artifacts more difficult for
registries to understand. The authors propose overcoming this using
[annotations](#annotations).

### Annotations

[Annotations](https://github.com/opencontainers/image-spec/blob/master/annotations.md)
are an optional part of the OCI specification. They can be included in the
top-level of an OCI index, at the top-level of a manifest, or as part of a
descriptor.

While they are generally unrestricted key-value pairs, [some guidance is given
for keys by the OCI](https://github.com/opencontainers/image-spec/blob/master/annotations.md#pre-defined-annotation-keys).

Formalising some common annotations across various artifact types will make them
useful for registries to use to better understand the artifacts.

### The future

There is a clear trend towards more types of artifacts being stored in container
registries. While it's too early to predict exactly what will be stored in
registries, a couple of observations can be made.

Just as how OCI indices evolved from a need, it's likely that the OCI will
specify new schemas and media types once there are several new artifacts being
stored in registries. This needs to be done, in part, with hindsight so that the
specification captures the requirements for storing all artifacts.

It's likely that other artifacts will also want to reference multiple images
and/or artifacts. This means that the approach of using an OCI index for a
single object is likely to continue to be valid and useful.

Agreeing upon and using several common annotations will provide a solid
foundation for future specification work. If Helm, CNAB, and whatever comes next
find annotations that work well, these can be promoted to fields of the next OCI
specification.

## FAQ

### What is CNAB?

CNAB stands for Cloud Native Application Bundle. It aims to be the equivalent of
a deb (or MSI) package but for all things Cloud Native. See
[this site](https://cnab.io) for more.

### What is an OCI index?

A container image is presented by a registry as a
[manifest](https://github.com/opencontainers/image-spec/blob/master/manifest.md).
Each manifest is platform specific which means that in order to use an image on
multiple platforms, one needs to fetch the correct manifest for that platform.

Initially this was solved by indicating the platform as part of the tag, e.g.:
`myimage:tag-<platform>`. This is undesirable for base images used on multiple
platforms as it requires platform specific code. As such a manifest list was
added where multiple manifests could be presented behind the same code.

The client can fetch the manifest list (or
[OCI index](https://github.com/opencontainers/image-spec/blob/master/image-index.md))
and match its platform to those presented so that it gets the correct image
manifest. Registries are content addressable so the manifest can be found using
the digest.

An example of this is the `golang:alpine` image, note that a Docker manifest
list is the older version of an OCI index and they serve the same purpose:

```console
$ DOCKER_CLI_EXPERIMENTAL=enabled docker manifest inspect golang:alpine
{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1365,
         "digest": "sha256:9ba4afd1011b9151c3967651538b600f19e48eff2ddde987feb2b72ab2c0bb69",
         "platform": {
            "architecture": "amd64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1572,
         "digest": "sha256:89cc8193f7abc4237b0df2417e0b9fa61687017cd507456b21241d9ea4d94dd3",
         "platform": {
            "architecture": "arm",
            "os": "linux",
            "variant": "v6"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1572,
         "digest": "sha256:6803d818bb3dd6edfccbe35b70477483fc75ed11d925e80e4af443f737146328",
         "platform": {
            "architecture": "arm64",
            "os": "linux",
            "variant": "v8"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1572,
         "digest": "sha256:923201b72b9dcf9e96290f9f171c34a9c743047d707afe69d9c167d430607db7",
         "platform": {
            "architecture": "386",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1572,
         "digest": "sha256:1782df1f6fa4ede250547f7aa491d3d11fa974bc62a1b9b0e493f07c3ba4430f",
         "platform": {
            "architecture": "ppc64le",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 1572,
         "digest": "sha256:d50b32798b5e99eb046ee557c567c83d25e39cbaf42f1d4f24af708644a69123",
         "platform": {
            "architecture": "s390x",
            "os": "linux"
         }
      }
   ]
}
```

## Getting started

Clone the project into your GOPATH. You can then build it using:

```console
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
  "schemaVersion": 2,
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 285,
      "annotations": {
        "io.cnab.manifest.type": "config"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:196d12cf6ab19273823e700516e98eb1910b03b17840f9d5509f03858484d321",
      "size": 506,
      "annotations": {
        "io.cnab.manifest.type": "invocation"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:6bb891430fb6e2d3b4db41fd1f7ece08c5fc769d8f4823ec33c7c7ba99679213",
      "size": 507,
      "annotations": {
        "io.cnab.component.name": "image-1",
        "io.cnab.manifest.type": "component"
      }
    }
  ],
  "annotations": {
    "io.cnab.keywords": "[\"keyword1\",\"keyword2\"]",
    "io.cnab.runtime_version": "v1.0.0",
    "org.opencontainers.artifactType": "application/vnd.cnab.manifest.v1",
    "org.opencontainers.image.authors": "[{\"name\":\"docker\",\"email\":\"docker@docker.com\",\"url\":\"docker.com\"}]",
    "org.opencontainers.image.description": "description",
    "org.opencontainers.image.title": "my-app",
    "org.opencontainers.image.version": "0.1.0"
  }
}
```

The first manifest in the manifest list references the CNAB configuration. An
example of this follows:

```json
{
  "schemaVersion": 2,
  "config": {
    "mediaType": "application/vnd.cnab.config.v1+json",
    "digest": "sha256:4bc453b53cb3d914b45f4b250294236adba2c0e09ff6f03793949e7e39fd4cc1",
    "size": 578
  },
  "layers": []
}
```

Subsequent manifests in the manifest list are standard OCI images.

This example proposes two OCI specification and registry changes:
1. It proposes the addition of an `org.opencontainers.artifactType` annotation to be included in the OCI specification.
1. It requires that registries support the `application/vnd.cnab.config.v1+json` media type for a config type.

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

# Mind The Gap

`mindthegap` provides utilities to manage air-gapped image bundles, both
creating image bundles and seeding images from a bundle into an existing
image registry.

## Building

Build the CLI using `make build-snapshot` that will output binary into
`dist/mindthegap_$(GOOS)_$(GOARCH)/mindthegap` and put it in `$PATH`.

## Usage

### Creating an image bundle

```shell
mindthegap create image-bundle --images-file <path/to/images.yaml> \
  --platform <platform> [--platform <platform> ...] \
  --output-file <path/to/output.tar>
```

See the [example images.yaml](images-example.yaml) for the structure of the
images config file.

Platform can be specified multiple times. Supported platforms:

```plain
linux/amd64
linux/arm64
windows/amd64
windows/arm64
```

All images in the images config file must support all the requested platforms.

The output file will be a tarball that can be seeded into a registry,
or that can be untarred and used as the storage directory for a Docker registry
served via `registry:2`.

### Pushing an image bundle

```shell
mindthegap push image-bundle --image-bundle <path/to/images.tar> \
  --to-registry <registry.address> \
  [--to-registry-insecure-skip-tls-verify]
```

All images in the image bundle tar file will be pushed to the target docker registry.

## How does it work?

`mindthegap` starts up a [Docker registry](https://docs.docker.com/registry/)
and then uses [`skopeo`](https://github.com/containers/skopeo) to copy the
specified images for all specified platforms into the running registry. The
resulting registry storage is then tarred up, resulting in a tarball of the
specified images.

The resulting tarball can be loaded into a running Docker registry, or
be used as the initial storage for running your own registry from via Docker
or in a Kubernetes cluster.

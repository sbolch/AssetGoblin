---
title: Docker
weight: 9
url: /docs/container.html
---

# Docker

The container package is `ghcr.io/sbolch/assetgoblin`, with Linux amd64 and arm64
images. The container workflow publishes `edge` from `main`, version and
major/minor tags from `v*` release tags, and `latest` from stable version tags.
Pull requests build without publishing. Tags are available after the workflow
successfully publishes them; use a version or digest for repeatable deployments.

## Run

```sh
docker run --rm -p 8080:8080 \
  --mount type=bind,src="$PWD/assets",dst=/app/assets,readonly \
  --mount type=bind,src="$PWD/public",dst=/app/public,readonly \
  --mount type=volume,src=assetgoblin-cache,dst=/cache \
  ghcr.io/sbolch/assetgoblin:latest
```

Create the host asset and public directories before running. The default command
starts the server on port 8080. ImageMagick, vips, FFmpeg and CA certificates are
included. The image runs as UID/GID `10001:10001`; mounted files must be readable by
that user, and any custom cache directory must be writable. The default working
directory is `/app`, so the standard `assets/img` and `public` paths work unchanged.

## Configuration and cache

Mount a configuration file at `/app/config.json` (or another supported configuration
filename). For example, add this option to the run command:

```sh
--mount type=bind,src="$PWD/config.json",dst=/app/config.json,readonly
```

The configuration and transformation caches live under `/cache/assetgoblin`.
A named cache volume preserves them across container replacements. After changing
configuration, clear the gob cache using the same volume before restarting:

```sh
docker run --rm \
  --mount type=volume,src=assetgoblin-cache,dst=/cache \
  ghcr.io/sbolch/assetgoblin:latest -clear-gob
```

Passing arguments replaces the default command. To use custom server arguments,
include `-serve -plugins-dir /plugins` explicitly. If you change the configured
port, adjust the published container port too. See [configuration](configuration.md)
and [CLI options](cli.md) for application settings.

## Optional plugins

Copy selected scripts into a flat host directory using the
[plugin deployment instructions](../plugins/README.md#deployment), then mount it:

```sh
--mount type=bind,src="$PWD/runtime-plugins",dst=/plugins,readonly
```

The default command discovers scripts in `/plugins` at startup. No example
plugins are enabled in the image. FFmpeg supports the audio, video and QOI
examples; QOA additionally requires `qoaconv`, which is not bundled. Build a derived
image with that tool to use QOA. See the [plugin catalog](../plugins/README.md)
for each plugin’s requirements. Deploy trusted scripts; their tools execute with
the container user’s permissions.

## Build locally

```sh
docker build -t assetgoblin:local .
docker run --rm -p 8080:8080 assetgoblin:local
```

The multi-stage Dockerfile compiles the Go binary and keeps the compiler and
sources out of the runtime image. Optional build arguments `VERSION`, `BUILD_TIME`
and `GIT_COMMIT` populate the same metadata as binary releases. Update containers
by pulling a new image and recreating them, rather than using `-update` inside an
existing container.

## GitHub Container Registry

The workflow authenticates with `GITHUB_TOKEN` and `packages: write`; no separate
registry secret is needed. Images carry an OCI source label linking them to the
repository. After the first publication, set the package’s visibility to **Public**
if anonymous pulls are desired: newly published packages default to private.
Private pulls require a classic personal access token with `read:packages`:

```sh
printf '%s' "$CR_PAT" | docker login ghcr.io -u YOUR_USERNAME --password-stdin
```

See [GitHub’s Container Registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
for access controls and authentication.

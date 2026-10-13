---
title: Home
weight: 1
---

[![AssetGoblin](assets/header.png)](https://github.com/sbolch/AssetGoblin)

# AssetGoblin

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="assets/mascot/welcome.png">
    <img src="assets/mascot/welcome.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

[![Latest Release](https://img.shields.io/github/v/release/sbolch/AssetGoblin.svg)](https://github.com/sbolch/AssetGoblin/releases)

Serve static files and transform images through presets or URL parameters.
Optional Lua plugins add audio, video and extra formats without rebuilding the server.

## Requirements

 - Go 1.27.2 or newer (for building from source)
 - [ImageMagick](https://imagemagick.org) and/or [vips](https://www.libvips.org)
 - [FFmpeg](https://ffmpeg.org/download.html) (optional, for media transformations and QOI)

## Quick start

Download a binary from [Releases](https://github.com/sbolch/AssetGoblin/releases),
or build from source:

```sh
go build -o assetgoblin .
./assetgoblin -serve
```

Put images in `assets/img` and static files in `public`, relative to your working
directory. Request `/img/640/photo.webp` to resize and convert an image, or
`/path/to/file` to serve a file from `public`. The default port is `8080`.

For a container deployment, see the [Docker guide](docs/container.md).

## Documentation

- [Configuration](docs/configuration.md): defaults, presets, lookup paths and caching.
- [Images](docs/images.md): URL formats and transformation options.
- [Request security](docs/security.md): rate limiting and signed URLs.
- [Command-line usage](docs/cli.md): server and maintenance flags.
- [Plugin API](docs/plugins.md): deployment, callbacks, tools, codecs and inheritance.
- [Example plugins](plugins/README.md): audio, video, QOI and QOA setup.
- [Docker](docs/container.md): container images, mounts and GHCR publishing.
- [Development](docs/development.md): checks and documentation publishing.

The same documentation is available as a [GitHub Pages site](https://assetgoblin.sbolch.hu/).

## Acknowledgements

AssetGoblin is built with the following open-source projects. Thank you to their
authors and contributors.

| | Project | Used for |
| --- | --- | --- |
|  | [Charm terminal utilities](https://github.com/charmbracelet/x/tree/main/term) | Terminal detection |
| <img src="https://raw.githubusercontent.com/go-fuego/fuego/main/static/fuego.svg" width="64" alt="" loading="lazy" decoding="async"> | [Fuego](https://github.com/go-fuego/fuego) | HTTP server and routing |
| 🥒 | [Gocuke](https://github.com/regen-network/gocuke) | Gherkin feature tests |
|  | [GopherLua](https://github.com/yuin/gopher-lua) | Lua plugin runtime |
| <img src="https://gohugo.io/images/hugo-logo-wide.svg" width="64" alt="" loading="lazy" decoding="async"> | [Hugo](https://gohugo.io/) | Documentation site generation |
| <img src="https://github.com/user-attachments/assets/d13bbe1a-d2b2-4d18-9302-419a0bc3f579" width="64" alt="" loading="lazy" decoding="async"> | [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and tables |
| <img src="https://github.com/user-attachments/assets/acae9193-2974-41f3-808d-2d433f5ada5e" width="64" alt="" loading="lazy" decoding="async"> | [Viper](https://github.com/spf13/viper) | Configuration loading |

## Sponsors

<a href="https://gazmag.hu" target="_blank"><img src="https://gazmag.hu/icon/logo-long.svg" alt="GazMag" height="50"></a>

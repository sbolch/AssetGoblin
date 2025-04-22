![AssetGoblin](assets/header.png)

# AssetGoblin

[![Latest Release](https://img.shields.io/github/v/release/sbolch/AssetGoblin.svg)](https://github.com/sbolch/AssetGoblin/releases)

> Serve static files or dynamically manipulated images with ease

## Requirements

 - [ImageMagick](https://imagemagick.org) and/or [vips](https://www.libvips.org)
 - config.json file inside working directory formatted like below

```json
{
  "port": "8080",
  "public_dir": "public",
  "secret": "",
  "rate_limit": {
    "limit": 0,
    "ttl": "1m"
  },
  "image": {
    "formats": ["avif", "jpeg", "jpg", "png", "tiff", "webp"],
    "presets": {
      "lg": "960",
      "lg2x": "1920",
      "sm": "640",
      "sm2x": "1280"
    },
    "path": "/img/",
    "directory": "assets/img",
    "cache_dir": "cache",
    "avif_through_vips": false
  }
}
```
> [!NOTE]
> During its first run AssetGoblin encodes the config in [gob](https://pkg.go.dev/encoding/gob) format.
> If you want to modify the configuration, edit your json file, then delete the gob file, so it can re-encode it.

> [!NOTE]
> Avif through vips is disabled by default because that encoding is really slow at the moment.
> If you want to use avif files, you must have ImageMagick installed.

## URLs

### Images

Resizes image's width (while preserving original ratio) according to the queried preset and transforms it to
the queried format, e.g. if you have your settings.json just like the one above and a *path/to/image.png* file
inside */path/to/workdir/assets/img*, you can do any of the following:

```
https://localhost:8080/img/lg/path/to/image.webp
https://localhost:8080/img/lg2x/path/to/image.png
https://localhost:8080/img/sm/path/to/image.jpg
https://localhost:8080/img/sm2x/path/to/image.avif
```

### Static files

```
https://localhost:8080/path/to/file
```

## Rate limiter

soon...

## Token

soon...

## Flags

### -help

List flags

### -update

Update to latest version

### -version

Print version info

## Sponsors

<a href="https://gazmag.hu" target="_blank"><img src="https://gazmag.hu/icon/logo-long.svg" alt="GazMag" height="50"></a>

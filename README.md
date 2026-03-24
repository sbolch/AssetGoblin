[![AssetGoblin](assets/header.png)](https://github.com/sbolch/AssetGoblin)

# AssetGoblin

[![Latest Release](https://img.shields.io/github/v/release/sbolch/AssetGoblin.svg)](https://github.com/sbolch/AssetGoblin/releases)

> Serve static files or dynamically manipulated images with ease

## Requirements

 - [ImageMagick](https://imagemagick.org) and/or [vips](https://www.libvips.org)
 
## Configuration

You can modify any settings in JSON, TOML, YAML, HCL, envfile formats.
You only need to define the ones you want to override.

### Default settings as JSON

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
    "presets": {},
    "path": "/img/",
    "directory": "assets/img",
    "cache_dir": "<OS default cache>/assetgoblin/img",
    "avif_through_vips": false
  }
}
```

### Presets configuration

Define custom presets in your config file:

```json
{
  "image": {
    "presets": {
      "lg": {"width": 960},
      "lg2x": {"width": 1920},
      "sm": {"width": 640},
      "sm2x": {"width": 1280},
      "thumbnail": {"width": 200, "height": 200, "fit": "cover", "rotate": 90, "flip": "horizontal", "filters": ["grayscale", "blur"]}
    }
  }
}
```

- `width` (required): Target width in pixels
- `height` (optional): Target height in pixels. Use `0` for auto (preserves aspect ratio)
- `fit` (optional): `contain` (default) or `cover`
- `rotate` (optional): Rotation in degrees (0, 90, 180, 270)
- `flip` (optional): `horizontal`, `vertical`, or `both`
- `crop` (optional): Crop region (`top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`, `bottom-right`)
- `brightness` (optional): Brightness adjustment (-100 to 100)
- `contrast` (optional): Contrast adjustment (-100 to 100)
- `gamma` (optional): Gamma adjustment (0.1 to 10.0)
- `filters` (optional): Array of filters to apply in order (`grayscale`, `sepia`, `blur`, `sharpen`, `negate`, `normalize`, `equalize`, `contrast`, `edge`, `emboss`, `charcoal`, `solarize`, `paint`, `oil`, `sketch`, `vignette`)

Config file lookup order:
- Current working directory (`./config.*`)
- Linux: `$XDG_CONFIG_HOME/assetgoblin` (usually `~/.config/assetgoblin`) and `/etc/assetgoblin`
- macOS: `~/Library/Application Support/assetgoblin` and `/Library/Application Support/assetgoblin`
- Windows: `%APPDATA%\\assetgoblin` and `%ProgramData%\\assetgoblin`

If `image.cache_dir` is not set, AssetGoblin defaults to `<OS cache dir>/assetgoblin/img`:
- Linux: `$XDG_CACHE_HOME/assetgoblin/img` (usually `~/.cache/assetgoblin/img`)
- macOS: `~/Library/Caches/assetgoblin/img`
- Windows: `%LocalAppData%\\assetgoblin\\img`

> [!NOTE]
> For optimization, during its first run AssetGoblin encodes the config in [gob](https://pkg.go.dev/encoding/gob) format and stores it at `<OS cache dir>/assetgoblin/config.gob`.
> If you modify the config file, delete the gob file (or run `-clear-gob`) so it can re-encode it.

> [!NOTE]
> Avif through vips is disabled by default because that encoding is really slow at the moment.
> If you want to use avif files, you must have ImageMagick installed.

## URLs

### Images

Resizes image's width (while preserving original ratio) according to the queried preset and transforms it to the
queried format, e.g. if you have the default settings and a *path/to/image.png* file inside */path/to/workdir/assets/img*,
you can use any of the following URL formats:

```
# Preset-based (using configured presets)
https://localhost:8080/img/lg/path/to/image.webp
https://localhost:8080/img/lg2x/path/to/image.png
https://localhost:8080/img/sm/path/to/image.jpg
https://localhost:8080/img/sm2x/path/to/image.avif

# Direct width (height auto-calculated to preserve aspect ratio)
https://localhost:8080/img/640/path/to/image.jpg
https://localhost:8080/img/1920/path/to/image.png

# Direct width x height (with fit parameter)
https://localhost:8080/img/640x480/path/to/image.jpg
https://localhost:8080/img/800x600/path/to/image.png?fit=cover
```

- `fit=contain` (default): Image is resized to fit within the dimensions while preserving aspect ratio
- `fit=cover`: Image is resized to cover the entire dimensions, cropping excess (center-aligned)
- `rotate`: Rotation in degrees (0, 90, 180, 270)
- `flip`: Flip image (`horizontal`, `vertical`, `both`)
- `crop`: Crop region (`top-left`, `top`, `top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`, `bottom-right`)
- `brightness`: Brightness adjustment (-100 to 100)
- `contrast`: Contrast adjustment (-100 to 100)
- `gamma`: Gamma adjustment (0.1 to 10.0)
- `filter`: Apply filter (comma-separated: `?filter=grayscale,blur`)
  - `grayscale`: Convert to grayscale
  - `sepia`: Apply sepia tone
  - `blur`: Apply blur
  - `sharpen`: Sharpen image
  - `negate`, `invert`: Invert colors
  - `normalize`: Normalize image levels
  - `equalize`: Histogram equalization
  - `contrast`: Increase contrast
  - `edge`: Detect edges
  - `emboss`: Emboss effect
  - `charcoal`: Charcoal drawing effect
  - `solarize`: Solarize effect
  - `paint`: Oil painting effect
  - `oil`: Oil painting effect (stronger)
  - `sketch`: Sketch effect
  - `vignette`: Vignette effect

### Static files

```
https://localhost:8080/path/to/file
```

## Rate limiter

The rate limiter is a simple token bucket algorithm that limits the number of requests to a given path.

The rate limit is defined in the config file under the `rate_limit` key. The `limit` key defines the maximum number of
requests allowed within the time-to-live (TTL) period, which is defined by the `ttl` key. The TTL is specified in a
format that can be parsed by Go's `time.ParseDuration` function.
If you don't need it, just set the limit to 0 (unlimited).

## Token

The token is a hash string that is used to validate the request.
It's generated from a secret and the request path and is used to prevent unauthorized access to the server.

The secret is defined in the config file under the `secret` key. If you don't want to use tokens, just leave it empty.

If you use it, you must use the same secret to generate the token in your requests.
You have to pass the token as get parameter in the URL.

You can generate the token like this:

In Go:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func generateToken(secret, path string) string {
    hasher := hmac.New(sha256.New, []byte(secret))
    hasher.Write([]byte(path))
    return hex.EncodeToString(hasher.Sum(nil))
}
```

In PHP:

```php
$token = hash_hmac('sha256', $path, $secret);
```

You can definitely use other languages, but the algorithm is the same.

## Usage

### -help

List flags

### -serve

Run the server

### -clear-gob

Delete the cached gob config file from the OS default cache directory.

### -config

Print the effective config as a table, including the used config file location, whether it was loaded from gob cache, and the gob cache file path.

### -update

Update to latest version

### -version | -v

Print version info

## Sponsors

<a href="https://gazmag.hu" target="_blank"><img src="https://gazmag.hu/icon/logo-long.svg" alt="GazMag" height="50"></a>

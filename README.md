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
> If you want to modify the configuration, edit your config file, then delete the gob file, so it can re-encode it.

> [!NOTE]
> Avif through vips is disabled by default because that encoding is really slow at the moment.
> If you want to use avif files, you must have ImageMagick installed.

## URLs

### Images

Resizes image's width (while preserving original ratio) according to the queried preset and transforms it to the
queried format, e.g. if you have the default settings and a *path/to/image.png* file inside */path/to/workdir/assets/img*,
you can do any of the following:

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

### -update

Update to latest version

### -version

Print version info

## Sponsors

<a href="https://gazmag.hu" target="_blank"><img src="https://gazmag.hu/icon/logo-long.svg" alt="GazMag" height="50"></a>

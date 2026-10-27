---
title: Configuration
weight: 2
url: /docs/configuration.html
---

# Configuration

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/configuration.png">
    <img src="../assets/mascot/configuration.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

You can modify any settings in JSON, TOML, YAML, HCL, envfile formats.
You only need to define the ones you want to override.

## Default settings as JSON

```json
{
  "port": "8080",
  "public_dir": "public",
  "python_executable": "",
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

## Python interpreter

Set `python_executable` to select the interpreter for Python plugins, including a
virtual environment's Python:

```json
{
  "python_executable": "/opt/assetgoblin-venv/bin/python"
}
```

The default empty value searches for `python3`, then `python`, on PATH. Python
3.10+ is required only when Python plugins are present. Virtual environments need
no shell activation. This setting is included in the cached configuration; after
changing it, run `assetgoblin -clear-gob` and restart the server. `-config` displays
the configured value.

## Presets configuration

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

Preset fields use the same [image transformation options](images.md#transformation-options)
as URL queries, with `filters` as an array instead of the `filter` query string.
`width` is required; `height = 0` preserves the aspect ratio. Missing `fit` defaults
to `contain`; omitted or zero `gamma` defaults to `1` (no correction).

## File lookup and caching

Config file lookup order:

- Current working directory (`./config.*`)
- Linux: `$XDG_CONFIG_HOME/assetgoblin` (usually `~/.config/assetgoblin`) and `/etc/assetgoblin`
- macOS: `~/Library/Application Support/assetgoblin` and `/Library/Application Support/assetgoblin`
- Windows: `%APPDATA%\\assetgoblin` and `%ProgramData%\\assetgoblin`

If `image.cache_dir` is not set, AssetGoblin defaults to `<OS cache dir>/assetgoblin/img`:

- Linux: `$XDG_CACHE_HOME/assetgoblin/img` (usually `~/.cache/assetgoblin/img`)
- macOS: `~/Library/Caches/assetgoblin/img`
- Windows: `%LocalAppData%\\assetgoblin\\img`

> For faster startup, during its first run AssetGoblin encodes the config in [gob](https://pkg.go.dev/encoding/gob) format and stores it at `<OS cache dir>/assetgoblin/config.gob`.
> If you modify the config file, delete the gob file (or run `-clear-gob`) so it can re-encode it.

Each uncached load starts from fresh defaults, so settings removed from a file
return to their defaults. A failed load leaves the existing configuration unchanged.

AVIF encoding through vips is disabled by default. Use ImageMagick for AVIF output,
or enable `image.avif_through_vips` explicitly.

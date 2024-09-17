<div style="text-align: center;"><img src="logo.svg" alt="AssetGoblin" width="256"></div>

# AssetGoblin

> Simply manipulate images or serve static files

## Requirements

 - [ImageMagick](https://imagemagick.org) and/or [vips](https://www.libvips.org)
 - config.json file inside working directory formatted like below

```json
{
  "port": "8080",
  "public_dir": "public",
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
> *During its first run AssetGoblin encodes the config in [gob](https://pkg.go.dev/encoding/gob) format.*
> *If you want to modify the configuration, edit your json file, then delete the gob file, so it can re-encode it.*

> [!NOTE]
> *Avif through vips is disabled as default because that encoding is really slow at the moment.*
> *If you want to use avif files, you must have Imagick.*

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

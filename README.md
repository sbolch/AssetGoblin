# Asset manager written in Go

> Simply manipulate images or serve static files

## Requirements

 - Imagick and/or vips
 - assets directory inside working directory
 - config.json file inside working directory formatted like below

```json
{
  "port": "8080",
  "image": {
    "formats": ["avif", "jpeg", "jpg", "png", "tiff", "webp"],
    "presets": {
      "lg": "960",
      "lg2x": "1920",
      "sm": "640",
      "sm2x": "1280"
    }  
  }
}
```
> [!NOTE]
> *During its first run the program encodes the config in gob format. If you want to modify the configuration,
> edit your json file, then delete the gob file, so it can re-encode it.*

## URLs

### Images

Resizes image's width (while preserving original ratio) according to the queried preset and transforms it to
the queried format, eg. if you have the settings.json above and a *path/to/image.png* file inside
*/path/to/workdir/assets/img*, you can do any of the following:

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

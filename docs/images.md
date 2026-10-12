---
layout: default
title: Images
nav_order: 3
---

# Images

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/images.png">
    <img src="../assets/mascot/images.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

Resizes an image’s width (while preserving original ratio) according to the queried preset and transforms it to the
queried format, e.g. if you have the default settings and a *path/to/image.png* file inside */path/to/workdir/assets/img*,
you can use any of the following URL formats:

```
# Preset-based (using configured presets)
http://localhost:8080/img/lg/path/to/image.webp
http://localhost:8080/img/lg2x/path/to/image.png
http://localhost:8080/img/sm/path/to/image.jpg
http://localhost:8080/img/sm2x/path/to/image.avif

# Direct width (height auto-calculated to preserve aspect ratio)
http://localhost:8080/img/640/path/to/image.jpg
http://localhost:8080/img/1920/path/to/image.png

# Direct width x height (with fit parameter)
http://localhost:8080/img/640x480/path/to/image.jpg
http://localhost:8080/img/800x600/path/to/image.png?fit=cover
```

## Transformation options

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

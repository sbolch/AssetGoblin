---
layout: default
title: QOI
nav_order: 3
parent: Example plugins
---

# QOI image codec plugin

Deploy `assetgoblin-qoi.lua` using the [shared instructions](../README.md#deployment). Install FFmpeg on PATH with its `qoi` encoder
and decoder (`ffmpeg -encoders` and `ffmpeg -decoders` list available codecs).
The core image backend still requires ImageMagick or libvips.

The script registers QOI input/output on the existing image route. No separate
route or configured format-list change is needed. For example:

```text
/img/320/photo.qoi
/img/160x160/photo.qoi?fit=cover&rotate=90
/img/320/photo.png
```

The first two requests encode a native or QOI source as QOI after core resizing
and transforms. The last request can decode a QOI-only source to PNG. Native
source formats are tried in their configured order, then plugin formats in sorted
order. Sources live in the configured image directory. QOI responses use `image/qoi`.

The script owns the QOI/PNG FFmpeg commands. See the [image codec API](../../docs/plugins.md#extending-core-image-formats)
for the PNG bridge, source discovery and cache behavior.

[QOI reference](https://github.com/phoboslab/qoi).

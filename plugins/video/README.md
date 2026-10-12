---
layout: default
title: Video
nav_order: 2
parent: Example plugins
---

# Video plugin

Serves `/v/` from `assets/video`, relative to the server working
directory. Edit `directory` in `assetgoblin-video.lua` to change the source root.
See [deployment](../README.md#deployment) for copying scripts and restarting.
The script owns its formats, validation, codecs and FFmpeg arguments.

## Transformations

See [shared media parameters](../README.md#shared-media-parameters) for FFmpeg,
clip duration, start offset, volume and output selection.

Video supports temporal trimming, pixel cropping, aspect-preserving downscaling,
and MP4/H.264 or WebM/VP8 conversion:

```text
/v/movie.mp4?start=2&duration=8&crop=640:360:0:0&width=320&height=180&format=mp4
/v/movie.mp4?duration=10&format=webm
```

`crop` is `width:height:x:y` in source pixels. Crop runs before resizing.
`width`/`height` define a bounding box, must be even, and cannot exceed 1280x720.
Output is capped at 30 fps.
No upscaling occurs. Transformation inputs: MP4, MOV, WebM, MKV.

Shared staging, caching, size/deadline limits and HTTP behavior are documented
in the [tool execution API](../../docs/plugins.md#generic-external-tool-capability).

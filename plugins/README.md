---
title: Example plugins
weight: 7
url: /plugins/README.html
---

# Example plugins

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/media.png">
    <img src="../assets/mascot/media.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

These examples ship with AssetGoblin but are optional, independently deployed
Python scripts. Their format rules and tool arguments live in the scripts.

| Plugin | Extends | Tool dependency | Documentation |
| --- | --- | --- | --- |
| Audio | Host route API (`/a/`) | FFmpeg for transformations | [Audio](audio/README.md) |
| Video | Host route API (`/v/`) | FFmpeg for transformations | [Video](video/README.md) |
| QOI | Core image codecs (`/img/`) | FFmpeg with QOI support | [QOI](qoi/README.md) |
| QOA | Audio plugin (`/a/`) | FFmpeg and reference `qoaconv` | [QOA](qoa/README.md) |

## Deployment

From the repository root, copy selected scripts into one flat directory:

```sh
mkdir -p runtime-plugins
cp plugins/audio/assetgoblin-audio.py runtime-plugins/
cp plugins/video/assetgoblin-video.py runtime-plugins/
cp plugins/qoi/assetgoblin-qoi.py runtime-plugins/
cp plugins/qoa/assetgoblin-qoa.py runtime-plugins/
go run . -serve -plugins-dir runtime-plugins
```

For a built executable, copy selected Python files beside it and restart with `-serve`.
Discovery is nonrecursive: pointing at this catalog directory does not load scripts
inside its subdirectories. QOA requires the audio script in the same discovery
directory. Removing QOA restores the base audio plugin. Tool dependencies resolve
on PATH when a transformation runs. Install Python 3.10+ on PATH, or use
`-python` to select an interpreter (including a virtual environment).

Asset directories are relative to the server working directory, not this example
folder. Edit each route's `directory` in its script if needed.
See the [host API](../docs/plugins.md) for image codecs, inheritance and process pipelines.

## Shared media parameters

The audio and video scripts require FFmpeg only for modified requests. With no
transformation parameters they stream the original file. Their shared options are:

| Parameter | Accepted values |
| --- | --- |
| `duration` | Required for modifications; greater than 0 and at most 60 seconds. |
| `start` | Clip offset, 0..86400 seconds; default 0. |
| `volume` | Audio gain, 0..2; default 1. |
| `format` | Output format; see each plugin’s supported formats. |

Both scripts request one codec/filter thread and restrict FFmpeg protocols.
Invalid parameters return 400; conversion failures return a logged 500.
The [host execution reference](../docs/plugins.md#generic-external-tool-capability)
explains the shared worker, size/deadline limits, staging and cache maintenance.

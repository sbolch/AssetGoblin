---
title: Audio
weight: 1
url: /plugins/audio/README.html
---

# Audio plugin

Serves `/a/` from `assets/audio`, relative to the server working
directory. Edit `directory` in `assetgoblin-audio.py` to change the source root.
See [deployment](../README.md#deployment) for copying scripts and restarting.
The script owns its formats, validation, codecs and FFmpeg arguments.

## Transformations

See [shared media parameters](../README.md#shared-media-parameters) for FFmpeg,
clip duration, start offset, volume and output selection.

Audio supports temporal cropping/trimming and MP3, WAV, OGG/Vorbis conversion:

```text
/a/song.wav?start=5&duration=10&format=mp3&volume=0.5
/a/song.mp3?duration=20&format=ogg
```

Transformation inputs: MP3, WAV, OGG, FLAC, M4A. FFmpeg must include `libmp3lame`
and `libvorbis` encoders. On macOS, Homebrew’s standard `ffmpeg` omits `libvorbis`;
install `ffmpeg-full` and add its binaries to PATH:

```sh
brew install ffmpeg-full
export PATH="$(brew --prefix ffmpeg-full)/bin:$PATH"
```

Use `ffmpeg -encoders` to inspect the available encoders.

## QOA extension

Deploy the [QOA extension](../qoa/README.md) alongside this script to add QOA
input/output while retaining this handler’s trimming, volume and validation.

Shared staging, caching, size/deadline limits and HTTP behavior are documented
in the [tool execution API](../../docs/plugins.md#generic-external-tool-capability).

---
title: QOA
weight: 4
url: /plugins/qoa/README.html
---

# QOA audio extension

Deploy `assetgoblin-qoa.py` **together with**
`assetgoblin-audio.py` in the same flat discovery directory and restart AssetGoblin.
This extension keeps `/a/` and the audio plugin's configured directory. It adds
QOA input/output while reusing the parent's trim, volume and validation logic.
The base audio plugin continues to work independently when this extension is absent.

Install FFmpeg and the [QOA reference converter](https://github.com/phoboslab/qoa)
`qoaconv` on PATH. A pinned Unix build, using a C compiler and curl:

```sh
mkdir qoa-tool
cd qoa-tool
curl -fsSLO https://raw.githubusercontent.com/phoboslab/qoa/bc0589710ff6aa2f50e08fb4d0a815d567bc0f3e/qoa.h
curl -fsSLO https://raw.githubusercontent.com/phoboslab/qoa/bc0589710ff6aa2f50e08fb4d0a815d567bc0f3e/qoaconv.c
cc qoaconv.c -std=gnu99 -O3 -lm -o qoaconv
# Install qoaconv into a directory on PATH.
```

On Windows, build the same converter with a compatible C toolchain as `qoaconv.exe`
and add its directory to PATH. No compilation of AssetGoblin or the Python script is
needed. The minimal reference build converts QOA and PCM WAV; FFmpeg handles the
base audio plugin's other formats.

```text
/a/song.wav?duration=10&format=qoa&volume=0.5
/a/song.qoa?start=2&duration=5&format=mp3
/a/song.qoa?duration=5&format=qoa
/a/song.qoa
```

Modified requests follow the [shared media parameters](../README.md#shared-media-parameters). QOA input is decoded to WAV, the parent's FFmpeg plan trims/converts it,
and QOA output is encoded from PCM WAV. A modified QOA input without `format`
defaults to WAV. Original QOA files stream unchanged without tool execution,
including HEAD and byte ranges. QOA output uses the explicit `audio/qoa` header.

The extension composes up to three conversion stages through the
[inheritance and pipeline API](../../docs/plugins.md#extending-another-plugin).
It requests bitexact PCM WAV output for compatibility with qoaconv’s reader.
See [audio options](../audio/README.md) for the inherited transformation rules.

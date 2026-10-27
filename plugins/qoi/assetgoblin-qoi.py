"""QOI input/output through PNG; resizing stays in the core image service."""


def plan(codec, extension):
    return {
        "tool": "ffmpeg", "extension": extension,
        "args": ["-nostdin", "-hide_banner", "-loglevel", "error", "-y",
                 "-threads", "1", "-protocol_whitelist", "file,pipe", "-i", "{input}",
                 "-frames:v", "1", "-c:v", codec, "-threads", "1", "{output}"],
    }


plugin = {
    "api_version": 1,
    "tools": ["ffmpeg"],
    "image_codecs": {
        "qoi": {
            "mime_type": "image/qoi",
            "decode": lambda request: plan("png", "png"),
            "encode": lambda request: plan("qoi", "qoi"),
        },
    },
}

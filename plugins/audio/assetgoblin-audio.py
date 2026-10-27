"""Audio policy and FFmpeg arguments; Go supplies staging, caching and serving."""
import math

FORMATS = {"mp3", "wav", "ogg"}
OPERATIONS = {"start", "duration", "format", "volume"}


def bad(message):
    return {"status": 400, "body": message}


def number(query, key, default=None):
    try:
        value = float(query[key]) if key in query else default
        return value if value is not None and math.isfinite(value) else None
    except ValueError:
        return None


def handle(request):
    query = request["query"]
    if not OPERATIONS.intersection(query):
        return {"file": request["path"]}
    duration = number(query, "duration")
    if duration is None or not 0 < duration <= 60:
        return bad("Transformations require duration in seconds (>0 and <=60)")
    start = number(query, "start", 0)
    volume = number(query, "volume", 1)
    if start is None:
        return bad("start must be numeric")
    if volume is None:
        return bad("volume must be numeric")
    if not 0 <= start <= 86400:
        return bad("start must be 0..86400 seconds")
    if not 0 <= volume <= 2:
        return bad("volume must be 0..2")
    extension = request["path"].rsplit(".", 1)[-1].lower()
    output_format = query.get("format", extension if extension in FORMATS else "mp3")
    if output_format not in FORMATS:
        return bad("Unsupported audio output format")
    demuxer = {"mp3": "mp3", "wav": "wav", "ogg": "ogg", "flac": "flac", "m4a": "mov"}.get(extension)
    if demuxer is None:
        return bad("Unsupported input format")
    args = [
        "-nostdin", "-hide_banner", "-loglevel", "error", "-y",
        "-max_alloc", "67108864", "-threads", "1", "-filter_threads", "1",
        "-protocol_whitelist", "file,pipe", "-f", demuxer,
        "-ss", str(start), "-i", "{input}", "-t", str(duration),
        "-map_metadata", "-1", "-map_chapters", "-1",
        "-map", "0:a:0", "-vn", "-af", "volume=" + str(volume), "-ar", "48000",
    ]
    if output_format == "mp3":
        args += ["-c:a", "libmp3lame", "-b:a", "128k"]
    elif output_format == "ogg":
        args += ["-c:a", "libvorbis", "-q:a", "3"]
    else:
        args += ["-c:a", "pcm_s16le"]
    args += ["-threads", "1", "-fs", "33554432", "{output}"]
    return {"file": request["path"], "process": {"tool": "ffmpeg", "args": args, "extension": output_format}}


plugin = {
    "api_version": 1,
    "tools": ["ffmpeg"],
    "routes": [{"path": "/a/", "directory": "assets/audio", "handle": handle}],
}

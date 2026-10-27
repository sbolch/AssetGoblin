"""Video policy and FFmpeg arguments; Go supplies staging, caching and serving."""
import math
import re

FORMATS = {"mp4", "webm"}
OPERATIONS = {"start", "duration", "format", "volume", "width", "height", "crop"}


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
    output_format = query.get("format", extension if extension in FORMATS else "mp4")
    if output_format not in FORMATS:
        return bad("Unsupported video output format")
    dimensions = {}
    for key, maximum in (("width", 1280), ("height", 720)):
        if key in query:
            value = number(query, key)
            if value is None or not 0 <= value <= maximum or value % 2:
                return bad(key + " must be even and within 1280x720")
            dimensions[key] = int(value)
    filters = []
    if "crop" in query:
        if not re.fullmatch(r"[0-9]+:[0-9]+:[0-9]+:[0-9]+", query["crop"]):
            return bad("crop must be width:height:x:y (positive dimensions, maximum 4096)")
        width, height, x, y = map(int, query["crop"].split(":"))
        if not 0 < width <= 4096 or not 0 < height <= 4096 or x > 4096 or y > 4096:
            return bad("crop must be width:height:x:y (positive dimensions, maximum 4096)")
        filters.append(f"crop={width}:{height}:{x}:{y}")
    demuxer = {"mp4": "mov", "mov": "mov", "webm": "matroska", "mkv": "matroska"}.get(extension)
    if demuxer is None:
        return bad("Unsupported input format")
    args = [
        "-nostdin", "-hide_banner", "-loglevel", "error", "-y",
        "-max_alloc", "67108864", "-threads", "1", "-filter_threads", "1",
        "-protocol_whitelist", "file,pipe", "-f", demuxer,
        "-ss", str(start), "-i", "{input}", "-t", str(duration),
        "-map_metadata", "-1", "-map_chapters", "-1",
        "-map", "0:v:0", "-map", "0:a:0?", "-sn", "-dn",
    ]
    width = dimensions.get("width") or 1280
    height = dimensions.get("height") or 720
    filters += [f"scale=w='min(iw,{width})':h='min(ih,{height})':force_original_aspect_ratio=decrease:force_divisible_by=2", "setsar=1"]
    args += ["-vf", ",".join(filters), "-af", "volume=" + str(volume), "-pix_fmt", "yuv420p", "-r", "30"]
    if output_format == "mp4":
        args += ["-c:v", "libx264", "-preset", "ultrafast", "-crf", "28", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart"]
    else:
        args += ["-c:v", "libvpx", "-deadline", "realtime", "-cpu-used", "8", "-b:v", "1M", "-c:a", "libvorbis", "-q:a", "3"]
    args += ["-threads", "1", "-fs", "33554432", "{output}"]
    return {"file": request["path"], "process": {"tool": "ffmpeg", "args": args, "extension": output_format}}


plugin = {
    "api_version": 1,
    "tools": ["ffmpeg"],
    "routes": [{"path": "/v/", "directory": "assets/video", "handle": handle}],
}

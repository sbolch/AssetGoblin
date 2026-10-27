"""Compose QOA conversion with the audio plugin's validation and FFmpeg logic."""


def extend(parent):
    parent["tools"].append("qoaconv")
    route = parent["routes"][0]
    original = route["handle"]

    def handle(request):
        input_qoa = request["path"].lower().endswith(".qoa")
        output_qoa = request["query"].get("format") == "qoa"
        if not input_qoa and not output_qoa:
            return original(request)
        if not {"start", "duration", "format", "volume"}.intersection(request["query"]):
            return {"file": request["path"], "headers": {"Content-Type": "audio/qoa"}}
        adapted = dict(request)
        adapted["query"] = dict(request["query"])
        if output_qoa:
            adapted["query"]["format"] = "wav"
        if input_qoa:
            adapted["path"] = request["path"].rsplit(".", 1)[0] + ".wav"
        result = original(adapted)
        if "process" not in result:
            return result
        stages = []

        def convert(extension):
            stages.append({"tool": "qoaconv", "args": ["{input}", "{output}"], "extension": extension})

        if input_qoa:
            convert("wav")
        # qoaconv expects standard PCM WAV without metadata extensions.
        if output_qoa:
            result["process"]["args"][0:0] = ["-fflags", "+bitexact"]
        stages.append(result.pop("process"))
        if output_qoa:
            convert("qoa")
            result["headers"] = {"Content-Type": "audio/qoa"}
        result["file"] = request["path"]
        result["processes"] = stages
        return result

    route["handle"] = handle
    return parent


plugin = {"api_version": 1, "extends": "audio", "extend": extend}

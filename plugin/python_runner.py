"""Private stdio adapter, embedded in AssetGoblin; no installed SDK required."""
import hashlib
import json
import sys
import types

# Keep plugin print() calls off the protocol stream.
protocol = sys.stdout
sys.stdout = sys.stderr
sys.dont_write_bytecode = True
callbacks = {}


def encode(value):
    if callable(value):
        key = str(len(callbacks))
        callbacks[key] = value
        return {"$callback": key}
    if isinstance(value, dict):
        if "$callback" in value:
            raise ValueError("$callback is reserved for the plugin adapter")
        if not all(isinstance(key, str) for key in value):
            raise ValueError("manifest keys must be strings")
        return {key: encode(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [encode(item) for item in value]
    return value


def emit(value):
    text = json.dumps(value, allow_nan=False, separators=(",", ":"))
    if len(text.encode("utf-8")) > 4 * 1024 * 1024:
        raise ValueError("Python response exceeds 4 MiB")
    protocol.write(text + "\n")
    protocol.flush()


def load(filename, catalog, stack):
    if filename in stack or len(stack) >= 16:
        raise ValueError("cyclic or excessively deep plugin inheritance")
    with open(filename, "rb") as source:
        content = source.read(1024 * 1024 + 1)
    if len(content) > 1024 * 1024:
        raise ValueError("script exceeds 1048576 bytes")
    module_name = "assetgoblin_plugin_" + hashlib.sha256(filename.encode()).hexdigest()
    module = types.ModuleType(module_name)
    module.__file__ = filename
    sys.modules[module_name] = module
    exec(compile(content, filename, "exec"), module.__dict__)
    manifest = module.__dict__.get("plugin")
    if not isinstance(manifest, dict):
        raise ValueError("script must define a plugin dictionary")
    if type(manifest.get("api_version")) is not int or manifest["api_version"] != 1:
        raise ValueError("unsupported api_version (expected 1)")
    parents = []
    revision = hashlib.sha256(content).hexdigest()
    if "extends" in manifest:
        name = manifest["extends"]
        if not isinstance(name, str) or name not in catalog:
            raise ValueError("missing parent plugin or invalid extends name")
        parent_file = catalog[name]
        if not parent_file.endswith(".py"):
            raise ValueError("cross-language inheritance is not supported")
        parent, ancestors, parent_revision = load(parent_file, catalog, stack + [filename])
        extend = manifest.get("extend")
        if not callable(extend):
            raise ValueError("extension needs an extend(parent) function")
        manifest = extend(parent)
        if not isinstance(manifest, dict):
            raise ValueError("extend must return a plugin dictionary")
        parents = ancestors + [parent_file]
        revision = hashlib.sha256((revision + parent_revision).encode()).hexdigest()
    return manifest, parents, revision


try:
    if sys.version_info < (3, 10):
        raise RuntimeError("Python 3.10 or newer is required")
    init = json.loads(sys.stdin.readline())
    manifest, parents, revision = load(init["filename"], init["catalog"], [])
    # Include the selected runtime version so changing Python invalidates caches.
    revision = hashlib.sha256((revision + sys.version).encode()).hexdigest()
    emit({"value": encode(manifest), "parents": parents, "revision": revision})
except Exception as error:
    emit({"error": str(error)[:16384]})
    sys.exit(1)

for line in sys.stdin:
    try:
        message = json.loads(line)
        result = callbacks[message["callback"]](message["request"])
        emit({"value": result})
    except Exception as error:
        emit({"error": str(error)[:16384]})

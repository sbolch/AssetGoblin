---
title: Plugin API
weight: 6
url: /docs/plugins.html
---

# Plugin API

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/plugins.png">
    <img src="../assets/mascot/plugins.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

## Runtime plugins

AssetGoblin loads `assetgoblin-*.py` files beside its executable when `-serve`
starts. Deploy plugins independently by copying their files there and restarting
AssetGoblin. The executable needs no rebuild. Install **Python 3.10+** on PATH;
AssetGoblin looks for `python3`, then `python`. Set `python_executable` in [configuration](configuration.md#python-interpreter) to select a specific
interpreter, including a virtual environment's Python. Shell activation is not
needed. The container image includes Python and virtual-environment support.

See the [example catalog](../plugins/README.md) for deployment commands. With
`go run`, specify `-plugins-dir` because its executable lives in a temporary directory.
Discovery is nonrecursive and sorted by filename: place scripts in one flat directory.
On Windows, copy scripts beside `assetgoblin.exe` and configure `python_executable` if the interpreter
is not available on PATH. Unrelated files are ignored; an empty directory is valid.
Malformed scripts, unsupported API versions, invalid routes, and route/codec conflicts
fail startup atomically. CLI commands such as `-version` do not execute plugins.

Each Python script defines a `plugin` dictionary with `api_version = 1` and
nonempty `routes` or `image_codecs`, or declares `extends` and `extend(parent)`.
Callbacks are ordinary Python functions receiving dictionaries. A private runner
is embedded in AssetGoblin; no Python SDK package is required.

## Writing a plugin

Route prefixes must be unique, start/end with `/`, and cannot replace `/`.
A file-serving plugin can simply declare a directory:

```python
plugin = {
    "api_version": 1,
    "routes": [{"path": "/downloads/", "directory": "assets/downloads"}],
}
```

For custom logic, provide `handle(request)`:

```python
def greet(request):
    return {
        "status": 200,
        "headers": {"Content-Type": "text/plain; charset=utf-8"},
        "body": "Hello, " + request["query"].get("name", "visitor"),
    }

plugin = {
    "api_version": 1,
    "routes": [{"path": "/greet/", "handle": greet}],
}
```

The request has `method`, `path` (relative to the prefix), `full_path`, `query`
(the first value for each parameter), and `headers` (lowercase keys).
GET and HEAD are supported; other methods return 405. HEAD runs the callback but
suppresses its response body. Callback results must be JSON-compatible dictionaries;
callable values are allowed only in manifests.

Response `status` defaults to 200; `body` and `headers` are optional. Status codes
must be integers from 200 to 599. Header names/values are validated; framing headers
such as `Content-Length` are managed by Go. Status 204/304 cannot have a body.
A `file` response must have status 200 and no body. Files must be beneath the route's
`directory`, resolved against the startup working directory. Go streams files with
HEAD, byte ranges and conditional request support. Directory listings, traversal
and symlinks escaping the root are rejected. Signing and rate limiting wrap every route.

## Generic external-tool capability

A trusted plugin declares tools and returns a `process` response for an input file.
The host knows only the executable name, argument array, output extension and file
placeholders; plugins own query validation and tool-specific policy.

```python
def convert(request):
    return {
        "file": request["path"],
        "process": {
            "tool": "my-converter",
            "extension": "txt",
            "args": ["--uppercase", "--input", "{input}", "--output", "{output}"],
        },
    }

plugin = {
    "api_version": 1,
    "tools": ["my-converter"],
    "routes": [{"path": "/custom/", "directory": "assets/custom", "handle": convert}],
}
```

Tool names must be declared, be simple names (no paths), and resolve on PATH.
Arguments go directly to the executable without a shell. Arguments exactly equal
to `{input}` and `{output}` become staged input and temporary output paths; both
are required. Extensions are 1..16 ASCII alphanumeric characters. Argument arrays
allow up to 256 strings and 16 KiB total. Tools write their results to the output
path; tool stdout/stderr are discarded rather than buffered.

Processing uses one shared worker and a 10-second deadline including worker wait.
Inputs must be regular files up to 128 MiB; outputs must be nonempty and smaller
than 32 MiB. Go monitors output growth, atomically publishes successful results,
and removes temporary files after completion or failure. Staging uses disk so tools
can seek without accumulating whole files in the host's memory.

Results are cached in `<OS user cache>/assetgoblin/plugins`, keyed by plugin revision,
source path/size/modification time, tool, arguments and output extension. Python
revisions include the script, its `extends` ancestors and the interpreter version.
Script edits invalidate caches after restart. Changes to imported packages, external
scripts or tool binaries require clearing the cache; these are not automatically hashed.
Cache hits skip tool execution and retain HEAD, Range and conditional request support.
Periodically clear old cache files; there is no automatic disk quota.

A response can specify either one `process` or 1..8 `processes`, never both,
together with `file`. Each stage receives the previous stage's output as `{input}`.
The entire pipeline shares the 10-second deadline, including worker waits.

## Extending core image formats

Register `image_codecs` to add formats to the core image route. Each key is a
lowercase alphanumeric extension (1..16 characters). Its value supplies an image
`mime_type` and `decode(request)`/`encode(request)` functions. Each receives
`{"path": source_identity}` and returns a process plan using declared tools.
Decode must produce `extension = "png"`; encode must produce the registered extension.
Go supplies staged files through `{input}` and `{output}`.

The core decodes plugin formats to PNG, applies existing presets and transforms,
and encodes the result when requested. Registered formats automatically participate
in source discovery and output validation. Native formats retain their configured
order; plugin revisions participate in image caching.

The [QOI example](../plugins/qoi/README.md) demonstrates registration.
`/img/320/photo.qoi` can resize a native source; `/img/320/photo.png` can decode a
QOI-only source. The host has no QOI-specific implementation.

## Extending another plugin

Name the parent without `assetgoblin-` and `.py`:

```python
def extend(parent):
    original = parent["routes"][0]["handle"]
    def handle(request):
        result = original(request)
        # Inspect or adapt the parent's response here.
        return result
    parent["routes"][0]["handle"] = handle
    return parent

plugin = {"api_version": 1, "extends": "audio", "extend": extend}
```

The parent and extension must be in the same directory and use the same language.
Dependencies resolve regardless of filename order. Parent functions remain callable
inside the leaf's runtime. The returned manifest supplies combined tools, routes
and codecs. Only leaves register their composed manifests; removing an extension
restores its parent. Chains allow up to 16 scripts. Missing parents, cycles, invalid
manifests and conflicting registrations fail startup. To compose two extensions on
one route, make one extend the other. Restart after changing scripts.

The [QOA example](../plugins/qoa/README.md) wraps audio rather than duplicating its
validation and FFmpeg logic. It adds `qoaconv` and composes decode, transformation
and encode stages. No audio/QOA policy lives in the host.

## Execution and dependencies

Python plugins run in persistent subprocesses, retaining globals between requests.
Callbacks are serialized per plugin; file streaming and processing happen outside
the worker. Initialization and each callback have a one-second deadline, including
callback queue wait. Script sources and generated text bodies are limited to 1 MiB;
protocol messages are limited to 4 MiB. Large responses should use `file`.
Ordinary callback exceptions produce a logged 500 and leave the worker available.
Cancellation during execution, a timeout, worker exit or broken protocol terminates
that worker; subsequent callbacks fail until AssetGoblin restarts. No automatic
reload resets state or changes plugin logic during serving. Shutdown reaps workers.

Workers use Python's `-I` isolated mode. Python environment variables, user site
packages, the current directory and the plugin directory are excluded from import
lookup. Standard-library modules and packages installed in the selected interpreter
remain available. Install dependencies before deployment, preferably in a virtual
environment; no automatic `pip install` runs at startup. Use `extends` for plugin
composition or install helper modules as packages. `print()` is redirected away
from the protocol; worker diagnostics are discarded, while returned errors are
logged by Go without exposing details to HTTP clients.

Install only trusted scripts. Python has filesystem, network and process access
with the server user's permissions. Isolated mode is import isolation, not an OS
sandbox. Callback deadlines do not impose a hard memory limit or contain child
processes; run transformations through the host's declared-tool capability.

## Optional runtimes

`plugin.Backend` is the Go extension interface for optional interpreters. Register a
backend once during application initialization, before loading plugins. Registration
rejects duplicate extensions and the reserved `.py` extension. Python takes priority
when a registered backend has a plugin with the same name; a Python error fails
startup without falling back. Inheritance stays within a runtime.

Backends return manifests made of string-keyed maps, `[]any` arrays, `float64`
numbers, strings, booleans, nil and `plugin.Callback` functions. They return ancestor
filenames, a revision incorporating parent contents, and a non-nil closer. Failed
loads must clean up their own resources. Backends must respect the supplied context
and leave the discovery catalog unchanged. The host serializes callbacks and closing,
validates manifests and responses, and retains ownership of HTTP/file serving,
declared-tool execution, caching and request limits.

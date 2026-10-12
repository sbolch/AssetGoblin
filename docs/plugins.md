---
layout: default
title: Plugin API
nav_order: 6
---

# Plugin API

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/plugins.png">
    <img src="../assets/mascot/plugins.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

## Runtime plugins

AssetGoblin loads `assetgoblin-*.lua` files beside its executable when `-serve`
starts. Deploy plugins independently by copying their files there and restarting
AssetGoblin. The main executable needs no rebuild or external Lua installation. Plugins work on Linux, macOS, and Windows.

Go's native shared-library plugins do not support Windows and require matching
host/plugin toolchains and dependencies. AssetGoblin therefore uses an embedded
[Lua 5.1 interpreter](https://github.com/yuin/gopher-lua) for portable extensions.

See the [example catalog](../plugins/README.md) for available plugins and
copy commands. With `go run`, always specify `-plugins-dir` because its executable
lives in a temporary directory. Discovery is nonrecursive: place selected scripts
in one flat directory.

On Windows, copy the Lua files beside `assetgoblin.exe` and run
`.\assetgoblin.exe -serve`. `-plugins-dir` can override discovery for any build.
Discovery is sorted by filename. Unrelated files are ignored; an empty plugin
directory is valid. Malformed scripts, unsupported API versions, invalid routes and
route conflicts fail startup with a clear error. Plugins load only during server
startup, so CLI commands such as `-version` do not execute them.

## Generic external-tool capability

A trusted plugin may declare tool names in its manifest and return a `process`
response for an input file. The host knows only the executable name, an argument
array, the desired output extension, and input/output placeholders. It does not
interpret query parameters or construct tool-specific arguments.

```lua
return {
    api_version = 1,
    tools = { "my-converter" }, -- Your own executable installed on PATH.
    routes = {
        {
            path = "/custom/",
            directory = "assets/custom",
            handle = function(request)
                -- All custom operation selection and argument construction goes here.
                return {
                    file = request.path,
                    process = {
                        tool = "my-converter",
                        extension = "txt",
                        args = { "--uppercase", "--input", "{input}", "--output", "{output}" },
                    },
                }
            end,
        },
    },
}
```

Tool names must be declared by the plugin, be simple names (no paths), and resolve
on PATH. Arguments are passed directly to the executable, with no implicit shell.
An argument exactly equal to `{input}` or `{output}` is replaced with a staged
input file or temporary output path. Both placeholders are required. Extensions
are 1..16 ASCII alphanumeric characters; argument arrays allow up to 256 strings
and 16 KiB in total. The tool must write its result to the provided output path;
stdout/stderr are discarded rather than buffered.

The generic host limits processing to one worker, a 10-second deadline including
worker wait, regular inputs up to 128 MiB, and outputs smaller than 32 MiB. It
monitors output growth while the tool runs and rejects empty/oversized results.
Only successful complete outputs are atomically published. Input/output temporary
files are removed on completion or failure. Root-scoped input files are staged on
disk so tools can seek without accumulating whole files in Go or Lua memory.

Results are cached in `<OS user cache>/assetgoblin/plugins`, keyed by script
contents, source path/size/modification time, tool, arguments and output extension.
Changing script logic invalidates its cache automatically. Cache hits skip tool
execution; cached output retains HEAD, Range and conditional request support.
Operators should periodically clear old cache files and clear cache after changing
external tool binaries; there is no automatic disk quota.

Install only trusted scripts and tools. Declaring a tool permits that plugin to
run it with script-defined arguments. The subprocess has the server user's OS
permissions; generic process limits and root-scoped staging do not constitute an
OS sandbox. Plugin authors are responsible for validating HTTP-derived arguments
and enforcing any tool-specific protocol, format, filter or resource policies.

## Extending core image formats

A plugin can register `image_codecs` without adding a separate route. Each key
is a lowercase alphanumeric extension (1..16 characters); the value contains
`mime_type`, `decode(request)` and `encode(request)` functions. Each callback
receives `{path = source_identity}` and returns a generic process plan, using
only tools declared in its manifest. Decode must produce `extension = "png"`;
encode must produce the registered extension. The host supplies staged files
through `{input}` and `{output}`, not Lua filesystem access.

The core decodes plugin formats to PNG, applies its existing presets, resize and
query transforms, and encodes the result when a plugin output format is requested.
Registered formats are automatically available to image source discovery and
output validation; no configuration format-list edits are necessary. Native
formats retain their configured order when several sources share a basename.
Plugin revisions participate in image caching.

The [QOI example](../plugins/qoi/README.md) adds QOI input/output through
FFmpeg. `/img/320/photo.qoi` can resize a native source; `/img/320/photo.png`
can decode a QOI-only source. The core has no QOI-specific implementation.

## Extending another plugin

An extension names its parent by filename without `assetgoblin-` and `.lua`:

```lua
return {
    api_version = 1,
    extends = "audio",
    extend = function(parent)
        local original = parent.routes[1].handle
        parent.routes[1].handle = function(request)
            -- Inspect or adapt the request, call original, and adapt its response.
            return original(request)
        end
        return parent
    end,
}
```

The parent and extension must be in the same discovery directory. Dependencies
resolve regardless of filename order. Parent functions remain callable in the
extension's interpreter; unrelated plugins have separate states. The returned
manifest supplies the combined tools, routes and image codecs. Only leaf plugins
register their composed manifests, so inherited routes do not register twice.
Extensions may chain (up to 16 scripts); missing parents, cycles and conflicting
routes/codecs fail startup. To compose two extensions on one route, make one
extend the other. Restart after changing either file; cache revisions include
all ancestor script contents. Removing an extension restores its parent.

The [QOA example](../plugins/qoa/README.md) wraps the audio plugin rather
than duplicating its trim/volume/validation logic. It adds `qoaconv` to the declared
tools and composes decoding, the parent transformation, and encoding using
`processes = { plan1, plan2, ... }`. A response may specify one `process` or
1..8 `processes`, never both, together with `file`. Each stage receives the previous
stage's output as `{input}`. The whole pipeline shares the 10-second deadline,
including worker waits; each stage has the same size limits and cached outputs.
No audio or QOA policy is implemented by the host.

## Writing a plugin

Each file returns a table with `api_version = 1` and nonempty `routes` or
`image_codecs`, or declares `extends` with an `extend(parent)` function.
Route prefixes must be unique, start/end with `/`, and cannot replace `/`.
A file-serving plugin can simply declare a directory:

```lua
return {
    api_version = 1,
    routes = {
        { path = "/a/", directory = "assets/audio" },
    },
}
```

For `file` responses without `process` or `processes`, Go streams files directly, with HEAD,
byte ranges and conditional requests.
Directory listings, traversal and symlinks escaping the media root are rejected.
File paths in `directory` are resolved against the startup working directory.
Signing and rate limiting wrap every plugin route.

For custom logic, provide `handle(request)`. The request has `method`, `path`
(relative to the route prefix), `full_path`, `query` (the first value for each
parameter), and `headers` (lowercase keys). A callback returns either a text
response or a file path beneath its declared directory:

```lua
return {
    api_version = 1,
    routes = {
        {
            path = "/greet/",
            handle = function(request)
                return {
                    status = 200,
                    headers = { ["Content-Type"] = "text/plain; charset=utf-8" },
                    body = "Hello, " .. (request.query.name or "visitor"),
                }
            end,
        },
        {
            path = "/downloads/",
            directory = "assets/downloads",
            handle = function(request)
                return { file = request.path }
            end,
        },
    },
}
```

Response `status` defaults to 200; `body` and `headers` are optional. A `file`
response must have status 200 and no body; Go handles range/conditional statuses.
Other status codes must be integers from 200 to 599. Header names/values are
validated; framing headers such as `Content-Length` are managed by Go. GET and
HEAD are supported; other methods return 405. HEAD runs the callback but suppresses
its response body.

Scripts retain their own Lua state across requests. Lua callbacks are serialized
per plugin; file streaming runs concurrently outside Lua. Initialization and each
callback have a one-second deadline, including waiting for the interpreter, and
request cancellation stops Lua execution. Script errors produce a logged 500
without exposing internal details to clients. Scripts and generated text bodies
are limited to 1 MiB; large files should use `file` responses.

Lua provides base, string, table, and math libraries. Filesystem/process libraries,
package loading, debug access and file-loading functions are unavailable. Install
only trusted scripts: execution deadlines do not impose a hard Lua memory limit.

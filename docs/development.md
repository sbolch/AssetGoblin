---
title: Development
weight: 8
url: /docs/development.html
---

# Development

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/development.png">
    <img src="../assets/mascot/development.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

The HTTP server uses [Fuego](https://go-fuego.dev), with existing `net/http`
handlers to preserve streaming, range requests, conditional requests, and HEAD support.
Asset routes accept GET and HEAD; other methods return 405. OpenAPI generation and
Swagger routes are disabled for this file-serving application. Server timeouts are
15 seconds for reads/writes, 5 seconds for headers, and 60 seconds for idle connections.

```sh
go build ./...
go test ./... -race
go vet ./...
go run . -serve
```

## Behavioral tests

Behavioral checks are defined in `features/**/*.feature`, including configuration,
image options, resources, middleware, CLI/release handling and plugin execution.
Gocuke runners in each Go package bind these scenarios to Go step implementations.
Scenario outlines keep test inputs and expected outcomes in the feature files.

```sh
# Run all feature suites through Go's test runner.
go test ./... -run 'Features$' -v

# Run the complete suite with the race detector.
go test ./... -race
```

Add or change the appropriate feature scenario when behavior changes. Go test
functions serve as suite runners or subprocess fixtures; setup, actions and
assertions belong in step implementations. Conversion scenarios skip only when
their required tools are unavailable. CLI and release tests use a fake release
transport and never contact GitHub or install updates.

## Plugin integration checks

Run `go test ./... -race` when changing discovery, inheritance, callbacks, codecs
or external-tool processing. Core process tests use an arbitrary text tool.
Example loading is always tested; conversion tests require FFmpeg and ImageMagick,
and QOA additionally needs `qoaconv` on PATH. Linux CI builds a pinned QOA tool.
See [QOA setup](../plugins/qoa/README.md) for building it locally.

## Documentation site

The README, reference pages and example READMEs are the documentation sources.
Hugo renders them using the repository’s small documentation layout, with sidebar
navigation, syntax highlighting and automatic light/dark
colors. Markdown links keep working on GitHub; a Hugo render hook changes their
`.md` destinations to published `.html` pages. The existing page URLs and mascot
images are preserved.
Edit a topic in its reference page and link to it from examples instead of copying
shared API, deployment or execution-limit details.

In the repository’s **Settings → Pages**, select **GitHub Actions** as the publishing
source. The documentation workflow builds pull requests for validation and deploys
pushes to `main` or a manual run on `main`. It publishes only documentation and
site assets; application source and Lua scripts are excluded. The workflow uses
GitHub’s Pages actions; see [GitHub’s setup guide](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).

To preview locally, install [Hugo](https://gohugo.io/installation/) 0.167.0
(the version pinned in the workflow), then run:

```sh
hugo server
```

Open `http://localhost:1313/`. To reproduce the deployment build:

```sh
hugo --destination _site --minify --panicOnWarning
```

`hugo.toml` mounts the existing README, reference pages and plugin READMEs without
copying them into a separate content folder. Only Markdown files are mounted from
plugin folders; Lua files and application sources are never published. Page front
matter defines titles, weights and stable URLs. `layouts/` contains the templates
and link hook; `assets/docs.css` provides styling.
Generated `_site` and Hugo build state are ignored. No Ruby, Bundler,
Node.js or external theme installation is required.

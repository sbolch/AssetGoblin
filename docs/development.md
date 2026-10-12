---
layout: default
title: Development
nav_order: 8
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
Jekyll renders them with [Just the Docs](https://just-the-docs.com), providing
search, sidebar navigation, heading links and syntax highlighting. Relative Markdown
links work in GitHub and are rewritten to site URLs by `jekyll-relative-links`.
Edit a topic in its reference page and link to it from examples instead of copying
shared API, deployment or execution-limit details.

In the repository’s **Settings → Pages**, select **GitHub Actions** as the publishing
source. The documentation workflow builds pull requests for validation and deploys
pushes to `main` or a manual run on `main`. It publishes only documentation and
site assets; application source and Lua scripts are excluded. The workflow uses
GitHub’s Pages actions; see [GitHub’s setup guide](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).

To preview the same site locally with Ruby and Bundler installed:

```sh
bundle install
bundle exec jekyll serve --baseurl /AssetGoblin
```

Open `http://localhost:4000/AssetGoblin/`. Generated `_site`, Bundler state and
Jekyll caches are ignored. The theme is pinned in `_config.yml`; update it deliberately.

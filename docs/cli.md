---
title: Command-line usage
weight: 5
url: /docs/cli.html
---

# Command-line usage

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/cli.png">
    <img src="../assets/mascot/cli.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

## -help | -h

List flags

## -serve | -S

Run the server. A startup table shows the listen address, asset directories,
plugin routes and plugin image codecs before the listener starts.

## -clear-gob

Delete the cached gob config file from the OS default cache directory.

## -config

Print the effective config as a table, including the used config file location,
whether it was loaded from gob cache, and the gob cache file path.

## -update

Update the installed executable to the latest compatible stable release using
[go-selfupdate](https://github.com/creativeprojects/go-selfupdate). Drafts and
prereleases are ignored, and released builds never downgrade. Development builds
can update too. The updater selects the platform archive, verifies its SHA-256
against the release's `checksums.txt`, and replaces the running executable's
resolved installation path. Failed downloads or checksum validation leave the
installed executable intact. The install directory must be writable. Bundled
support files and separately installed plugins are not updated by this command.

A progress bar shows downloaded bytes and
percentage when the server supplies the archive size; otherwise it shows the byte
count. Terminal output refreshes in place, while redirected output prints only
the initial and final download lines. Installation success is reported after
checksum verification and executable replacement. Download failures use red styling;
release notes appear under a heading and wrap to the terminal width (up to 80
columns). Release notes retain their original Markdown text.

## -version | -v

Print version info

## -plugins-dir

Override the Python plugin discovery directory. See [plugin deployment](plugins.md#runtime-plugins).

## Terminal formatting

The logo, help and configuration tables, version details, success messages and update
notices use terminal-aware layout and color. Colors adapt to the output terminal,
are removed for redirected output, and respect `NO_COLOR`. Update notices appear
only after a successful release check finds a compatible newer stable release (or a stable release for a development build).

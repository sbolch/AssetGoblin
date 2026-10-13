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

Update to the latest version. A progress bar shows downloaded bytes and
percentage when the server supplies the archive size; otherwise it shows the byte
count. Terminal output refreshes in place, while redirected output prints only
the initial and final download lines. Checksum verification, extraction and
installation are reported as separate stages. Download failures use red styling;
release notes appear under a heading and wrap to the terminal width (up to 80
columns). Release notes retain their original Markdown text.

## -version | -v

Print version info

## -plugins-dir

Override the Lua plugin discovery directory. See [plugin deployment](plugins.md#runtime-plugins).

## Terminal formatting

The logo, help and configuration tables, version details, success messages and update
notices use terminal-aware layout and color. Colors adapt to the output terminal,
are removed for redirected output, and respect `NO_COLOR`. Update notices appear
only after a successful release check reports a different version.

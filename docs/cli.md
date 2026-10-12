---
layout: default
title: Command-line usage
nav_order: 5
---

# Command-line usage

<p>
  <picture>
    <source media="(prefers-reduced-motion: reduce)" srcset="../assets/mascot/cli.png">
    <img src="../assets/mascot/cli.gif" width="96" height="104" alt="" loading="lazy" decoding="async">
  </picture>
</p>

## -help

List flags

## -serve

Run the server

## -clear-gob

Delete the cached gob config file from the OS default cache directory.

## -config

Print the effective config as a table, including the used config file location, whether it was loaded from gob cache, and the gob cache file path.

## -update

Update to latest version

## -version | -v

Print version info

## -plugins-dir

Override the Lua plugin discovery directory. See [plugin deployment](plugins.md#runtime-plugins).

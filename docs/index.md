---
slug: /
sidebar_position: 1
---

# Introduction

A single bash script for managing [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) services on the host. Wraps `podman exec`, `podman quadlet`, and `systemctl`; unrecognized commands pass straight through to `podman`.

Pairs naturally with [foxws/laravel-podman](https://github.com/foxws/laravel-podman), whose Artisan `podman:*` commands render Quadlet files, but `lpod` doesn't require it — point it at any Quadlet-managed service.

## Requirements

- Linux with systemd (macOS, Windows, and WSL are not supported)
- Podman with Quadlet support

Continue to [Installation](./installation.md) to get started.

---
title: Introduction
metadata:
  role: Containers
  eyebrow: "Bash · Podman Quadlet · systemd"
  desc: "A single bash script for managing Podman Quadlet services on the host."
  runtime: "Podman (Quadlet)"
  licence: MIT
---

# Introduction

`lpod` is a single bash script for managing [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) services on your machine. It wraps three tools — `podman exec`, `podman quadlet`, and `systemctl` — behind one simple command. Anything it doesn't recognize is passed straight through to `podman`.

It works well with [foxws/laravel-podman](https://github.com/foxws/laravel-podman), a Laravel package whose Artisan `podman:*` commands generate Quadlet files. But you don't need that package to use `lpod` — it works with any Quadlet-managed service.

## Requirements

- Linux with systemd (macOS, Windows, and WSL are not supported)
- Podman with Quadlet support

Ready to get started? Continue to [Installation](./installation.md).

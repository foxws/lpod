---
section: Getting Started
order: 1
---

# Installation

Download the script and make it executable:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
```

That's it — `lpod` is a single file with no other dependencies. It doesn't need PHP or Composer.

## Pinning a version

The command above always installs the latest release. To pin a specific version instead, download it directly:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/download/v0.1.0/lpod
chmod +x ~/.local/bin/lpod
```

## Optional: lpod-setup

`lpod setup` needs a second binary, `lpod-setup`. It renders presets on a host that has Podman but no PHP — see [foxws/laravel-podman](https://github.com/foxws/laravel-podman). Download it the same way:

```sh
curl -fsSL -o ~/.local/bin/lpod-setup https://github.com/foxws/lpod/releases/latest/download/lpod-setup
chmod +x ~/.local/bin/lpod-setup
```

## Installing from source

You can also clone the repository and copy or symlink `lpod` onto your `PATH`. This gives you the unreleased `main` branch, so `lpod --version` reports `dev` instead of a tagged version number. Either way, it's a single dependency-free script — no PHP or Composer required.

## Optional: shell alias

This alias finds `lpod` relative to your current directory, so the same alias works in every project:

```sh
# Bash/Zsh, in ~/.bashrc or ~/.zshrc
alias lpod='[ -f lpod ] && bash lpod || bash "$(git rev-parse --show-toplevel)/lpod"'
```

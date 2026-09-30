---
section: Getting Started
order: 1
---

# Installation

Run the installer:

```sh
curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
```

It installs `lpod` and `lpod-setup` to `~/.local/bin` (`/usr/local/bin` as root) and writes the systemd templates for the [on-demand idle check](commands.md#on-demand-idle-check). It also offers to enable linger, so your services start at boot without logging in. Run it again to upgrade.

`lpod` is a single bash script with no other dependencies. It doesn't need PHP or Composer.

| Variable | Default | Description |
| --- | --- | --- |
| `LPOD_VERSION` | `latest` | The release to install, e.g. `v2.2.0` |
| `LPOD_INSTALL_DIR` | `~/.local/bin`, or `/usr/local/bin` as root | Where to install the scripts |

## Installing by hand

Download the script and make it executable:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
lpod idle setup
```

## Pinning a version

The installer takes `LPOD_VERSION`. By hand, download the release directly:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/download/v0.1.0/lpod
chmod +x ~/.local/bin/lpod
```

## Optional: lpod-setup

`lpod setup` needs a second binary, `lpod-setup`. It renders presets on a host that has Podman but no PHP — see [foxws/laravel-podman](https://github.com/foxws/laravel-podman). The installer includes it. By hand, download it the same way:

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

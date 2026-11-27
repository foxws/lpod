---
section: Getting Started
order: 1
---

# Installation

Run the installer:

```sh
curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
```

It installs `lpod` to `~/.local/bin` (`/usr/local/bin` as root). It also offers to enable linger, so your services start at boot without logging in.

The installer:
- **Checks the downloads.** Each one is verified against the release's `SHA256SUMS` before anything is replaced. A failed download or checksum, `SHA256SUMS` itself included, leaves your current install as it was.
- **Warns about common setup problems.** It warns when `podman quadlet` isn't available (Podman 6.0 or later is needed), and when your user has no `/etc/subuid` or `/etc/subgid` range for rootless Podman.
- **Never runs half a script.** Everything runs from the last line, so a download that's cut off doesn't run part of the installer.

lpod is being ported from bash to Go, one command at a time ([#59](https://github.com/foxws/lpod/issues/59)). On amd64 and arm64, a release from v4.3.0 installs two files next to each other:

- `lpod`, a static Go binary;
- `lpod.sh`, the bash script, which the binary hands every command it doesn't implement yet to.

Elsewhere, for a branch, and for releases before v4.3.0, `lpod` is the bash script on its own. Either way it doesn't need PHP or Composer, and its `--json` commands, which tools such as Pods run, still need [jq](https://jqlang.org).

| Variable | Default | Description |
| --- | --- | --- |
| `LPOD_VERSION` | `latest` | The release to install, e.g. `v2.2.0` |
| `LPOD_BRANCH` | | A branch to install instead of a release, e.g. `main` |
| `LPOD_INSTALL_DIR` | `~/.local/bin`, or `/usr/local/bin` as root | Where to install `lpod` (and `lpod.sh`) |

## Upgrading

```sh
lpod self-update            # the latest release
lpod self-update v2.2.0     # or a specific one
lpod self-update main       # or a branch, to try what isn't released yet
```

`self-update` runs the installer again for the `lpod` you're running, so it upgrades in place. A branch isn't checked against checksums, and `lpod --version` then says `dev-BRANCH`. Each user has their own `lpod` in `~/.local/bin`, so run it as the user whose services it manages; Pods does that for you with **Update lpod** on a host's settings page. Running the `curl` command again does the same.

## Uninstalling

```sh
curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash -s -- --uninstall
```

This removes `lpod` and `lpod.sh`, and what older releases left: the `lpod-setup` of releases before v4, and the `lpod-idle@*.timer` idle checks of releases before v3 (disabled first). Your services, secrets and volumes stay as they are. Remove those first with `lpod uninstall APPLICATION` if you want them gone too.

## Installing by hand

Download the bash script and make it executable. It works on its own, without the Go binary:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
```

## Pinning a version

The installer takes `LPOD_VERSION`. By hand, download the release directly:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/download/v0.1.0/lpod
chmod +x ~/.local/bin/lpod
```

## Installing from source

You can also clone the repository and copy or symlink `bin/lpod`, the bash script, onto your `PATH`. This gives you the unreleased `main` branch, so `lpod --version` reports `dev` instead of a tagged version number.

To try the Go binary from a checkout, build it and point it at the script with `LPOD_SCRIPT`:

```sh
go build -o ~/.local/bin/lpod ./cmd/lpod
LPOD_SCRIPT="$PWD/bin/lpod" lpod --version
```

## Optional: shell alias

This alias finds `lpod` relative to your current directory, so the same alias works in every project:

```sh
# Bash/Zsh, in ~/.bashrc or ~/.zshrc
alias lpod='[ -f lpod ] && bash lpod || bash "$(git rev-parse --show-toplevel)/lpod"'
```

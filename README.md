# lpod

A single bash script for managing [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) services on the host. Wraps `podman exec`, `podman quadlet`, and `systemctl`; unrecognized commands pass straight through to `podman`.

Pairs naturally with [foxws/laravel-podman](https://github.com/foxws/laravel-podman), whose Artisan `podman:*` commands render Quadlet files, but `lpod` doesn't require it — point it at any Quadlet-managed service.

## Requirements

- Linux with systemd (macOS, Windows, and WSL are not supported)
- Podman with Quadlet support

## Install

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
```

Dependency-free single script — no PHP or Composer required. See [Installation](docs/installation.md) for pinning a release, the optional `lpod-setup` binary, and a shell alias.

## Usage

```sh
lpod SERVICE COMMAND [options] [arguments]
```

```sh
lpod app up            # start the service
lpod app shell         # shell into the container
lpod app artisan migrate
```

`SERVICE` is a Quadlet service name (your app, or a sibling like `pgsql`). See [Usage](docs/usage.md) for details.

## Docs

Full documentation: **[foxws.github.io/lpod](https://foxws.github.io/lpod/)** (or browse [`docs/`](docs))

- [Installation](docs/installation.md)
- [Usage](docs/usage.md)
- [Commands reference](docs/commands.md)
- [Configuration](docs/configuration.md)

## Links

- [foxws/laravel-podman](https://github.com/foxws/laravel-podman) — the Laravel package `lpod` pairs with
- [Podman Quadlet reference](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

# lpod

A single bash script for managing [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) services on the host. Wraps `podman exec`, `podman quadlet`, and `systemctl`; unrecognized commands pass straight through to `podman`.

Pairs naturally with [Pods](https://github.com/foxws/pods), a web UI that manages a host through lpod's `--json` commands, but `lpod` doesn't require it — point it at any Quadlet-managed service.

## Requirements

- Linux with systemd (macOS, Windows, and WSL are not supported)
- Podman with Quadlet support
- For the `--json` commands that tools such as Pods run: `jq`, and `git` for `quadlet fetch`, `catalog`, `tags` and `head`

## Install

```sh
curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
```

Installs `lpod` to `~/.local/bin`, verified against the release's checksums. Upgrade with `lpod self-update`. No PHP or Composer required. lpod is being ported from bash to Go ([#59](https://github.com/foxws/lpod/issues/59)): on amd64 and arm64 it's a Go binary with the bash script next to it as `lpod.sh`. See [Installation](docs/installation.md) for pinning a release, installing by hand, and a shell alias.

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

Full documentation: browse [`docs/`](docs)

- [Installation](docs/installation.md)
- [Usage](docs/usage.md)
- [Commands reference](docs/commands.md)
- [Configuration](docs/configuration.md)

## Testing

```bash
tests/run.sh
```

Runs the `--json` commands against stand-ins for `podman`, `systemctl`, `journalctl` and `loginctl` (`tests/stubs`), which answer from the host in `tests/host`. It checks the output, the keys of the contract's fixtures (`tests/fixtures`) and how often lpod asks the host. It needs bash and jq.

The Go code has its own tests, and the same `tests/run.sh` runs through the Go binary with `LPOD_UNDER_TEST`:

```bash
go test ./...
go build -o /tmp/lpod/lpod ./cmd/lpod && cp bin/lpod /tmp/lpod/lpod.sh
LPOD_UNDER_TEST=/tmp/lpod/lpod tests/run.sh
```

The [devcontainer](.devcontainer) has Go, shellcheck, jq and gh.

## Links

- [foxws/pods](https://github.com/foxws/pods) — the web UI `lpod` pairs with
- [Podman Quadlet reference](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

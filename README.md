# lpod

A single bash script for managing [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) services on the host. Wraps `podman exec`, `podman quadlet`, and `systemctl`; unrecognized commands pass straight through to `podman`.

Pairs naturally with [foxws/laravel-podman](https://github.com/foxws/laravel-podman), whose Artisan `podman:*` commands render Quadlet files, but `lpod` doesn't require it — point it at any Quadlet-managed service.

See the [full documentation](https://foxws.github.io/lpod/) (or browse [`docs/`](docs) directly): [Installation](docs/installation.md), [Usage](docs/usage.md), [Commands](docs/commands.md), [Configuration](docs/configuration.md).

## Requirements

- Linux with systemd (macOS, Windows, and WSL are not supported)
- Podman with Quadlet support

## Installation

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/latest/download/lpod
chmod +x ~/.local/bin/lpod
```

Pin a specific release instead of `latest` for reproducibility:

```sh
curl -fsSL -o ~/.local/bin/lpod https://github.com/foxws/lpod/releases/download/v0.1.0/lpod
chmod +x ~/.local/bin/lpod
```

Or clone the repo and copy/symlink `lpod` onto your `PATH` yourself — note this gets the unreleased `main` branch, whose `lpod --version` reports `dev` rather than a tagged version. It's a single dependency-free script — no PHP or Composer required.

### Optional: shell alias

Resolves `lpod` relative to your current directory, so one alias works across every project:

```sh
# Bash/Zsh, in ~/.bashrc or ~/.zshrc
alias lpod='[ -f lpod ] && bash lpod || bash "$(git rev-parse --show-toplevel)/lpod"'
```

## Usage

```sh
lpod SERVICE COMMAND [options] [arguments]
```

`SERVICE` is the name of a Quadlet service (your app, or a sibling service like `pgsql`). Quadlet management commands (`setup`, `install`, `remove`, `uninstall`, `list`, `print`, `reload`) skip `SERVICE` — they manage Quadlets themselves rather than talking to a running service.

Quadlet installs the actual Podman container for `SERVICE.container` under the name `systemd-SERVICE` (e.g. `systemd-my-app`), to avoid clashing with unmanaged containers. `lpod` accounts for this automatically for `exec`-based commands (`shell`, `run`, `artisan`, etc.) — you always refer to the service by its plain `SERVICE` name, but expect to see the `systemd-` prefix if you inspect containers directly with `podman ps`.

`lpod --version` (or `-v`/`version`) prints the installed version.

## Commands reference

### Lifecycle

| Command           | Description              |
| ------------------ | ------------------------- |
| `lpod app up`      | Start the service         |
| `lpod app down`    | Stop the service           |
| `lpod app restart` | Restart the service        |
| `lpod app status`  | Show the service's status  |
| `lpod app secrets` | Prompt for and set the service's Quadlet secrets |

### Artisan, PHP & Composer

| Command                     | Description                          |
| ----------------------------- | ------------------------------------- |
| `lpod app artisan ...`      | Run an Artisan command (`art`/`a`)   |
| `lpod app php ...`           | Run PHP                               |
| `lpod app composer ...`      | Run Composer                          |
| `lpod app debug ARTISAN...` | Run an Artisan command with Xdebug enabled |
| `lpod app tinker`             | Start a Tinker session                |

### Node, npm, pnpm, Yarn & Bun

| Command            | Description |
| -------------------- | ----------- |
| `lpod app node ...` | Run Node    |
| `lpod app npm ...`  | Run npm     |
| `lpod app npx ...`  | Run npx     |
| `lpod app pnpm ...` | Run pnpm    |
| `lpod app pnpx ...` | Run pnpx    |
| `lpod app yarn ...` | Run Yarn    |
| `lpod app bun ...`  | Run Bun     |
| `lpod app bunx ...` | Run bunx    |

### Testing

| Command                | Description                             |
| ------------------------ | ----------------------------------------- |
| `lpod app test`        | `php artisan test`                      |
| `lpod app phpunit ...` | Run PHPUnit                             |
| `lpod app pest ...`    | Run Pest                                |
| `lpod app pint ...`    | Run Pint                                |
| `lpod app dusk`        | Run Dusk tests (requires laravel/dusk)  |
| `lpod app dusk:fails`  | Re-run previously failed Dusk tests     |

### Container CLI & binaries

| Command              | Description                                  |
| ---------------------- | ----------------------------------------------- |
| `lpod app shell`     | Shell into the container (alias `bash`)      |
| `lpod app root-shell`| Root shell into the container (alias `root-bash`) |
| `lpod app bin TOOL`  | Run `./vendor/bin/TOOL`                      |
| `lpod app run CMD`   | Run an arbitrary command in the container     |

### Other

| Command                            | Description                                              |
| ------------------------------------- | ----------------------------------------------------------- |
| `lpod app open`                    | Open the app URL in the browser                          |
| `lpod app artisan podman:publish`  | Publish the Podman container runtime files                |
| `lpod proxy export-cert [PATH]`    | Export the proxy's local CA certificate (default `~/proxy.crt`) |

### Quadlet management

| Command                                | Description                                        |
| ----------------------------------------- | ----------------------------------------------------- |
| `lpod setup ...`                       | Render presets without PHP on the host (needs the separate `lpod-setup` script from [foxws/laravel-podman](https://github.com/foxws/laravel-podman)) |
| `lpod install PRESET/SERVICE.quadlets` | Install a rendered Quadlet                          |
| `lpod remove NAME`                     | Remove an installed Quadlet                          |
| `lpod uninstall APPLICATION`           | Remove an application and all of its Quadlets       |
| `lpod list`                            | List installed Quadlets                              |
| `lpod print NAME`                      | Print the generated systemd unit                     |
| `lpod reload`                          | Reload the systemd manager configuration (`daemon-reload`) |

All Quadlet management commands except `reload` accept any extra flags `podman quadlet` itself takes (`--replace`, `--application`, `--force`, `--ignore`, etc). `secrets` (see [Lifecycle](#lifecycle) above) also forwards its extra flags, but to `podman secret create`.

`lpod app secrets` is built into `lpod` — no separate script needed. It reads `Secret=` directives off an installed unit (`podman quadlet print SERVICE.container`) and prompts for each: `type=env` asks for a masked value, `type=mount` (the default) asks for a file path (`.env` by default) and stores its contents. A secret reused under multiple names is only prompted once. For `type=env` secrets, leaving the value blank (just pressing Enter) skips that secret and keeps its current value untouched — handy for updating a single secret in a quadlet with `lpod app secrets --replace` without having to re-enter every other one.

> **Warning:** `remove`/`uninstall` delete the Podman volumes owned by the services they remove, with no undo.

## Configuration

| Variable            | Default   | Description                                              |
| ---------------------- | ----------- | ------------------------------------------------------------ |
| `LPOD_PODMAN_BINARY` | `podman`  | Podman binary to use — must be a single executable path   |
| `LPOD_PUBLISH_PATH`  | `podman`  | Where rendered `.quadlets` files are looked up for `install` |
| `APP_PORT`           | `80`      | Used by `lpod SERVICE open`                                |
| `APP_USER`           | `$(id -u)` | User `exec`-based commands run as in the container. Defaults to the invoking shell's own UID; set to an empty string to fall back to the image's own default user instead. `root-shell`/`root-bash` always use `root` regardless of this setting |

`lpod` also sources `.env`/`.env.$APP_ENV` from the current directory, and forwards known AI coding agent env vars (Claude Code, Cursor, Copilot, Codex, Gemini CLI, and others) into `exec`-based commands automatically.

## Links

- [foxws/laravel-podman](https://github.com/foxws/laravel-podman) — the Laravel package `lpod` pairs with
- [Podman Quadlet reference](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

---
section: Usage
order: 2
---

# Commands reference

This page lists every command `lpod` provides, grouped by what they do.

## Lifecycle

| Command             | Description                                       |
| -------------------- | -------------------------------------------------- |
| `lpod app up`      | Start the service                                 |
| `lpod app down`    | Stop the service                                  |
| `lpod app restart` | Restart the service                               |
| `lpod app status`  | Show the service's status                         |
| `lpod app secrets` | Prompt for and set the service's Quadlet secrets  |

## Artisan, PHP & Composer

| Command                     | Description                                 |
| ----------------------------- | --------------------------------------------- |
| `lpod app artisan ...`      | Run an Artisan command (`art`/`a`)          |
| `lpod app php ...`           | Run PHP                                      |
| `lpod app composer ...`      | Run Composer                                 |
| `lpod app debug ARTISAN...` | Run an Artisan command with Xdebug enabled  |
| `lpod app tinker`             | Start a Tinker session                       |

## Node, npm, pnpm, Yarn & Bun

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

## Testing

| Command                | Description                            |
| ------------------------ | ----------------------------------------- |
| `lpod app test`        | `php artisan test`                     |
| `lpod app phpunit ...` | Run PHPUnit                            |
| `lpod app pest ...`    | Run Pest                               |
| `lpod app pint ...`    | Run Pint                               |
| `lpod app dusk`        | Run Dusk tests (requires `laravel/dusk`) |
| `lpod app dusk:fails`  | Re-run previously failed Dusk tests    |

## Container CLI & binaries

| Command                | Description                                       |
| ------------------------- | ---------------------------------------------------- |
| `lpod app shell`      | Shell into the container (alias `bash`)             |
| `lpod app root-shell` | Root shell into the container (alias `root-bash`)   |
| `lpod app bin TOOL`   | Run `./vendor/bin/TOOL`                             |
| `lpod app run CMD`    | Run an arbitrary command in the container           |

## Other

| Command                            | Description                                                      |
| ------------------------------------- | -------------------------------------------------------------------- |
| `lpod app open`                    | Open the app URL in the browser                                  |
| `lpod app artisan podman:publish`  | Publish the Podman container runtime files                       |
| `lpod proxy export-cert [PATH]`    | Export the proxy's local CA certificate (default `~/proxy.crt`)  |

## Quadlet management

| Command                                | Description                                                  |
| ----------------------------------------- | ---------------------------------------------------------------- |
| `lpod setup ...`                       | Render presets without PHP on the host (needs `lpod-setup`, shipped alongside `lpod`) |
| `lpod install PRESET/SERVICE.quadlets` | Install a rendered Quadlet                                   |
| `lpod remove NAME`                     | Remove an installed Quadlet                                  |
| `lpod uninstall APPLICATION`           | Remove an application and all of its Quadlets                |
| `lpod list`                            | List installed Quadlets                                      |
| `lpod print NAME`                      | Print the generated systemd unit                              |
| `lpod reload`                          | Reload the systemd manager configuration (`daemon-reload`)    |

Every Quadlet management command except `reload` accepts the same extra flags as `podman quadlet` itself — things like `--replace`, `--application`, `--force`, or `--ignore`. The `secrets` command (see [Lifecycle](#lifecycle) above) also forwards extra flags, but to `podman secret create` instead.

`lpod app secrets` is built into `lpod` — you don't need a separate script. It reads the `Secret=` lines from an installed unit (`podman quadlet print SERVICE.container`) and asks you for each one:

| Secret type | What it asks for |
| --- | --- |
| `env` | A masked value, entered directly |
| `mount` (default) | A file path (defaults to `.env`); `lpod` stores that file's contents |

If the same secret name is used more than once, `lpod` only asks for it once.

For an `env` secret, leaving the value blank (just pressing Enter) skips it and keeps its current value untouched. This makes it easy to update a single secret with `lpod app secrets --replace` without having to re-enter every other one.

> **Warning:** `remove` and `uninstall` delete the Podman volumes owned by the services they remove. This cannot be undone.

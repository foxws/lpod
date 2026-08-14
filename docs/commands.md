---
sidebar_position: 4
---

# Commands reference

## Lifecycle

| Command           | Description              |
| ------------------ | ------------------------- |
| `lpod app up`      | Start the service         |
| `lpod app down`    | Stop the service           |
| `lpod app restart` | Restart the service        |
| `lpod app status`  | Show the service's status  |
| `lpod app secrets` | Prompt for and set the service's Quadlet secrets |

## Artisan, PHP & Composer

| Command                     | Description                          |
| ----------------------------- | ------------------------------------- |
| `lpod app artisan ...`      | Run an Artisan command (`art`/`a`)   |
| `lpod app php ...`           | Run PHP                               |
| `lpod app composer ...`      | Run Composer                          |
| `lpod app debug ARTISAN...` | Run an Artisan command with Xdebug enabled |
| `lpod app tinker`             | Start a Tinker session                |

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

| Command                | Description                             |
| ------------------------ | ----------------------------------------- |
| `lpod app test`        | `php artisan test`                      |
| `lpod app phpunit ...` | Run PHPUnit                             |
| `lpod app pest ...`    | Run Pest                                |
| `lpod app pint ...`    | Run Pint                                |
| `lpod app dusk`        | Run Dusk tests (requires laravel/dusk)  |
| `lpod app dusk:fails`  | Re-run previously failed Dusk tests     |

## Container CLI & binaries

| Command              | Description                                  |
| ---------------------- | ----------------------------------------------- |
| `lpod app shell`     | Shell into the container (alias `bash`)      |
| `lpod app root-shell`| Root shell into the container (alias `root-bash`) |
| `lpod app bin TOOL`  | Run `./vendor/bin/TOOL`                      |
| `lpod app run CMD`   | Run an arbitrary command in the container     |

## Other

| Command                            | Description                                              |
| ------------------------------------- | ----------------------------------------------------------- |
| `lpod app open`                    | Open the app URL in the browser                          |
| `lpod app artisan podman:publish`  | Publish the Podman container runtime files                |
| `lpod proxy export-cert [PATH]`    | Export the proxy's local CA certificate (default `~/proxy.crt`) |

## Quadlet management

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

:::warning
`remove`/`uninstall` delete the Podman volumes owned by the services they remove, with no undo.
:::

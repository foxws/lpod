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
| `lpod app xdebug on [MODE]` | Turn Xdebug on for web requests and workers (default mode `debug`) |
| `lpod app xdebug off`        | Turn Xdebug off again                        |
| `lpod app xdebug status`     | Show whether Xdebug is on                    |
| `lpod app tinker`             | Start a Tinker session                       |

### Xdebug

`lpod app debug` runs one Artisan command with `XDEBUG_MODE=debug` (or your `XDEBUG_MODE`) and `XDEBUG_TRIGGER=1`.

For web requests, `lpod app xdebug on` writes a Quadlet drop-in, `~/.config/containers/systemd/app.container.d/lpod-xdebug.conf` (`/etc/containers/systemd/` as root), that sets `XDEBUG_MODE`. It then restarts the service if it's running. Pass a mode such as `debug,profile` to use another one. `lpod app xdebug off` removes the drop-in and restarts the service again.

With `xdebug.start_with_request=trigger`, a session only starts for requests that carry `XDEBUG_TRIGGER` or `XDEBUG_SESSION`, e.g. from a browser extension. The image needs the Xdebug extension; `lpod` warns when it's missing.

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
| `lpod install PRESET/UNIT.socket`      | Install and enable a rendered systemd socket or timer        |
| `lpod remove NAME`                     | Remove an installed Quadlet, socket or timer                 |
| `lpod uninstall APPLICATION`           | Remove an application and all of its Quadlets                |
| `lpod list`                            | List installed Quadlets                                      |
| `lpod print NAME`                      | Print the generated systemd unit                              |
| `lpod reload`                          | Reload the systemd manager configuration (`daemon-reload`)    |

Rendered `.socket` and `.timer` units, such as those for [on-demand services](https://foxws.nl/laravel-podman/ondemand), aren't Quadlets. `lpod install` copies them to `~/.config/systemd/user/` (or `/etc/systemd/system/` as root), along with the `.service` of the same name if it was rendered, and enables them. Units a socket or timer lists in `Wants=` that were rendered next to it, such as an on-demand socket's idle timer, are installed and started with it. For an instance of a template, such as `laravel-idle@my-app.timer`, that's the template (`laravel-idle@.timer` and `laravel-idle@.service`) and the instance's drop-in folder (`laravel-idle@my-app.service.d`). Pass `--replace` to overwrite installed ones; templates are shared by every app on the host, so they're always replaced. `lpod remove NAME.socket` disables and deletes them again, along with those wanted units. Of a template instance it only removes the drop-in folder, and leaves the template for the other apps.

Every Quadlet management command except `reload` accepts the same extra flags as `podman quadlet` itself — things like `--replace`, `--application`, `--force`, or `--ignore`. The `secrets` command (see [Lifecycle](#lifecycle) above) also forwards extra flags, but to `podman secret create` instead.

`lpod app secrets` is built into `lpod` — you don't need a separate script. It reads the `Secret=` lines from an installed unit (`podman quadlet print SERVICE.container`) and asks you for each one:

| Secret type | What it asks for |
| --- | --- |
| `env` | A masked value, entered directly |
| `mount` (default) | A file path (defaults to `.env`); `lpod` stores that file's contents |

If the same secret name is used more than once, `lpod` only asks for it once.

For an `env` secret, leaving the value blank (just pressing Enter) skips it and keeps its current value untouched. This makes it easy to update a single secret with `lpod app secrets --replace` without having to re-enter every other one.

## On-demand idle check

| Command                         | Description                                                            |
| -------------------------------- | ----------------------------------------------------------------------- |
| `lpod idle APPLICATION`         | Run the application's idle check once, and show what it did             |
| `lpod idle disable APPLICATION` | Turn off the `lpod-idle@APPLICATION.timer` of lpod releases before v3   |

An [on-demand](https://foxws.nl/laravel-podman/ondemand) app stops when it's idle, but its queue workers and scheduler timer keep running, and they keep the database and cache awake. The idle check stops them once the app has no work left. It's part of [laravel-podman](https://foxws.nl/laravel-podman/ondemand#the-idle-check) 6.1 and later: the `ondemand` preset renders the shared `laravel-idle@.timer` and `laravel-idle@.service` templates and the app's `laravel-idle@APPLICATION.service.d` drop-in next to the socket (`APPLICATION-idle.timer` and `APPLICATION-idle.service` from 6.1 to 6.3), and the socket starts the timer. `lpod install ondemand/APPLICATION-ondemand.socket` installs them with it, so there's nothing to enable. The workers it checks are set with `PODMAN_ONDEMAND_IDLE_WORKERS`.

See how a run went with `journalctl --user -u laravel-idle@APPLICATION` (`APPLICATION-idle` before laravel-podman 6.4).

### Moving from lpod v2

lpod v2 ran its own copy of the check from `lpod-idle@APPLICATION.timer`. That timer keeps working after the upgrade, since it runs `lpod idle APPLICATION`, but it only duplicates the socket's check. Once laravel-podman is on 6.1, reinstall the socket and turn the old timer off:

```bash
lpod install ondemand/my-app-ondemand.socket --replace
lpod idle disable my-app
```

Workers you listed in `LPOD_IDLE_WORKERS` go in `PODMAN_ONDEMAND_IDLE_WORKERS` now. `lpod idle enable` only prints how to install the socket.

## Troubleshooting

| Command       | Description                                                          |
| -------------- | ---------------------------------------------------------------------- |
| `lpod doctor [--json]` | Check the host for what the services need, and how to fix what's missing |

`lpod doctor` checks:

- that Podman runs and has the `podman quadlet` command (Podman 5.6 or newer);
- that the systemd manager is reachable;
- for rootless services: linger, your entries in `/etc/subuid` and `/etc/subgid`, and whether containers may publish ports 80 and 443 for the proxy (`net.ipv4.ip_unprivileged_port_start`);
- that no app still uses the [idle check](#moving-from-lpod-v2) of lpod releases before v3;
- that the host trusts the proxy's local certificate, when the proxy runs;
- that the host in `APP_URL`, from the `.env` in the current directory, resolves;
- that no services have failed.

It changes nothing. It prints how to fix each problem, and the commands that need root are for you to run. It exits with an error when it finds something that stops the services from running, and succeeds on warnings.

`lpod doctor --json` prints the same checks as JSON, for installers and other tools to read. Each check has a stable `id`, a `status` (`ok`, `warn` or `fail`), a `message` and a `fix`, which is empty for passing checks. The exit code is the same:

```json
{
  "lpod": "2.3.0",
  "failures": 0,
  "warnings": 1,
  "checks": [
    { "id": "podman", "status": "ok", "message": "Podman 5.6.1 with Quadlet", "fix": "" },
    { "id": "linger", "status": "warn", "message": "Linger is off, so services stop when you log out and don't start at boot", "fix": "loginctl enable-linger me" }
  ]
}
```

The ids are `podman`, `systemd`, `linger`, `subids`, `ports`, `idle-timers`, `proxy-certificate`, `app-url` and `failed-services`. Checks that don't apply, such as the rootless ones when running as root, are left out.

## Updating

| Command                      | Description                                                         |
| ----------------------------- | -------------------------------------------------------------------- |
| `lpod self-update [VERSION]` | Upgrade `lpod` and `lpod-setup` in place, to the latest or the given release |
| `lpod --version`             | Show the installed version                                          |

> **Warning:** `remove` and `uninstall` delete the Podman volumes owned by the services they remove. This cannot be undone.

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

Rendered `.socket` and `.timer` units, such as those for [on-demand services](https://foxws.nl/laravel-podman/ondemand), aren't Quadlets. `lpod install` copies them to `~/.config/systemd/user/` (or `/etc/systemd/system/` as root), along with the `.service` of the same name if it was rendered, and enables them. Pass `--replace` to overwrite installed ones. `lpod remove NAME.socket` disables and deletes them again.

Every Quadlet management command except `reload` accepts the same extra flags as `podman quadlet` itself — things like `--replace`, `--application`, `--force`, or `--ignore`. The `secrets` command (see [Lifecycle](#lifecycle) above) also forwards extra flags, but to `podman secret create` instead.

`lpod app secrets` is built into `lpod` — you don't need a separate script. It reads the `Secret=` lines from an installed unit (`podman quadlet print SERVICE.container`) and asks you for each one:

| Secret type | What it asks for |
| --- | --- |
| `env` | A masked value, entered directly |
| `mount` (default) | A file path (defaults to `.env`); `lpod` stores that file's contents |

If the same secret name is used more than once, `lpod` only asks for it once.

For an `env` secret, leaving the value blank (just pressing Enter) skips it and keeps its current value untouched. This makes it easy to update a single secret with `lpod app secrets --replace` without having to re-enter every other one.

## On-demand idle check

| Command                         | Description                                                                 |
| -------------------------------- | ---------------------------------------------------------------------------- |
| `lpod idle enable APPLICATION`  | Run the idle check for the application every minute (`lpod-idle@APPLICATION.timer`) |
| `lpod idle disable APPLICATION` | Stop running the idle check for the application                              |
| `lpod idle APPLICATION`         | Run the idle check once                                                      |
| `lpod idle setup`               | Write the `lpod-idle@.timer`/`lpod-idle@.service` templates. The installer runs it |

An [on-demand](https://foxws.nl/laravel-podman/ondemand) app stops when it's idle, but its queue workers and scheduler timer keep running, and they keep the database and cache awake. The idle check stops them once the app has no work left. While the app is asleep, it:

1. skips the run while the app, or a scheduled `schedule:run`, is running, and does nothing unless `APPLICATION-ondemand.socket` is active;
2. runs `php artisan podman:idle` (from `foxws/laravel-podman`) in each running worker: `APPLICATION-queue`, `APPLICATION-horizon`, and any in `LPOD_IDLE_WORKERS`. If one reports work in progress, it keeps everything running;
3. checks again that no request woke the app, then stops the workers and `APPLICATION-schedule.timer`.

The next request starts them again through the app's `Wants=` line. See how a run went with `journalctl --user -u lpod-idle@APPLICATION`.

To check and stop another worker, such as `my-app-imports`, add a drop-in with `systemctl --user edit lpod-idle@my-app.service`:

```ini
[Service]
Environment=LPOD_IDLE_WORKERS=imports
```

The templates run `lpod` by its full path. After moving `lpod`, run `lpod idle setup` again.

## Troubleshooting

| Command       | Description                                                          |
| -------------- | ---------------------------------------------------------------------- |
| `lpod doctor` | Check the host for what the services need, and how to fix what's missing |

`lpod doctor` checks:

- that Podman runs and has the `podman quadlet` command (Podman 5.6 or newer);
- that the systemd manager is reachable;
- for rootless services: linger, your entries in `/etc/subuid` and `/etc/subgid`, and whether containers may publish ports 80 and 443 for the proxy (`net.ipv4.ip_unprivileged_port_start`);
- that the [idle check](#on-demand-idle-check)'s templates are installed;
- that the host trusts the proxy's local certificate, when the proxy runs;
- that the host in `APP_URL`, from the `.env` in the current directory, resolves;
- that no services have failed.

It changes nothing. It prints how to fix each problem, and the commands that need root are for you to run. It exits with an error when it finds something that stops the services from running, and succeeds on warnings.

## Updating

| Command                      | Description                                                         |
| ----------------------------- | -------------------------------------------------------------------- |
| `lpod self-update [VERSION]` | Upgrade `lpod` and `lpod-setup` in place, to the latest or the given release |
| `lpod --version`             | Show the installed version                                          |

> **Warning:** `remove` and `uninstall` delete the Podman volumes owned by the services they remove. This cannot be undone.

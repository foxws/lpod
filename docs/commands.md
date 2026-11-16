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

## JSON contract

Tools such as Pods manage a host by running these commands over SSH. With `--json`, each one prints a single JSON object with a `contract` key, the version of the shapes below. A tool refuses a contract it doesn't know, and a breaking change to any of them bumps it. They need [jq](https://jqlang.org).

| Command | Prints |
| --- | --- |
| `lpod version --json` | `contract`, `lpod`, `podman`, `systemd`, `scope` (`rootless` or `rootful`), `user` and `hostname` |
| `lpod apps [--json]` | `applications`: each installed application with its main unit, `kind`, `status`, `result`, `since`, `image`, `ports`, `memory`, `auto_update`, its `io.foxws.pods.*` labels, and the last `error` of a failed one |
| `lpod host [--json]` | `cpu` (with the `load` averages), `memory` (with swap), `disk` (of Podman's graph root), `containers`, `auto_update` (the next `podman-auto-update.timer` run), `system` (`os`, `kernel`, `booted_at`, `storage_driver`, `network_backend`, `quadlet_dir` and `linger`, which is `null` as root) and the number of `active` and `failed` Quadlet `units` |
| `lpod secrets [--json]` | `secrets` with the applications whose Quadlets use them (`used_by`), and the `missing` ones that Quadlets use but don't exist |
| `lpod secret create NAME --stdin [--driver file\|pass\|shell] [--replace] [--json]` | `{"ok": true}`. The value comes from stdin only, never the command line |
| `lpod APP status --json` | `application`: what `apps` prints, plus its `path`, `volumes`, `secrets` and containers (`items`) with their CPU and memory use |
| `lpod APP files --json` | `files`: the application's Quadlets and their drop-ins, with their contents |
| `lpod APP logs --json [--lines N] [--since TIME]` | `entries`: the journal of the application's units, as `time`, `unit`, `priority` and `message` |
| `lpod APP up\|down\|restart --json [--no-block]` | `{"ok": true}`. `up` and `restart` act on the main unit, `down` stops every unit of the application. With `--no-block`, it returns without waiting for systemd |
| `lpod APP down --force --json` | `{"ok": true}`. Stops every unit of the application right away: it queues the stop (so `Restart=` doesn't start them again), kills their processes with SIGKILL, waits up to 10 seconds for them to go down, then clears their failed state and start limit with `reset-failed`. For an application stuck starting, in a restart loop or slow to shut down |
| `lpod APP redeploy --json [--no-block]` | `{"ok": true}`. Pulls every image the application's containers use, except ones built on the host (`localhost/...`), then restarts its main unit. With `--no-block`, a transient unit, `lpod-redeploy-APP`, does the work and logs to the journal |
| `lpod APP secret add NAME --type env\|mount --target TARGET --json` | `{"ok": true}`. Writes the drop-in `lpod-secret-NAME.conf` for the application's main container, next to its Quadlet |
| `lpod secret rm NAME [--json]` | `{"ok": true}` |
| `lpod APP remove --json [--volumes] [--images] [--secrets]` | `{"ok": true}`. Stops the application and removes its Quadlets. With the flags, also its volumes, and the images and secrets no other application uses |
| `lpod containers --json` | `containers`: every container, managed by Quadlet or not, with its `state`, `health`, `exit_code`, `application`, `quadlet`, `unit`, `pod`, `ports`, CPU and memory use and `started_at` |
| `lpod container NAME inspect --json` | `container`: what `containers` prints, plus `created_at`, `memory_limit`, `restart_policy`, `restarts`, `user`, `auto_update`, `digest`, `health_check`, `mounts`, `networks`, `secrets` (where each is exposed, never its value) and `environment` (the Quadlet's `Environment=` lines) |
| `lpod container NAME logs --json [--lines N]` | `entries`, like `APP logs`. Containers outside Quadlet are read from `podman logs` |
| `lpod container NAME stats --json` | `stats`: `cpu` (%), `memory` and `memory_limit`, `net_in`, `net_out`, `block_in` and `block_out` (bytes since the container started), `pids` and `sampled_at`. A tool works out rates from two samples |
| `lpod containers prune --json` | `removed` and `reclaimed` (`null`): removes every stopped container. Quadlet recreates its containers when their units start |
| `lpod pods --json` | `pods`: every pod with its `status` (`running`, `degraded`, `exited`, …), the `application`, `quadlet` and `unit` of its `.pod` or `.kube` Quadlet (or of its containers' Quadlets), its number of `containers` and how many are `running` (without the infra container), the `ports` its infra container publishes, its `networks` and `created_at` |
| `lpod container NAME start\|stop\|restart --json` | `{"ok": true}`. Goes through the container's systemd unit when it has one, without waiting for it |
| `lpod images --json` | `images`: one per image, with `repository`, `tag`, `size`, `created_at`, the applications that use it (`used_by`), its number of `containers`, `dangling`, `auto_update`, and `update` when the last `auto-update --dry-run` found one |
| `lpod image pull REF --json`, `lpod image rm ID --json` | `{"ok": true}`. `rm` only takes an image ID |
| `lpod images prune --json` | `removed` (image IDs) and `reclaimed` (bytes): removes every image no container uses |
| `lpod auto-update --dry-run --json` | `{"ok": true}`. Asks the registries which images have an update, for `images` |
| `lpod volumes --json` | `volumes`: named, NFS and anonymous volumes and bind mounts, with their `quadlet`, `driver`, `mountpoint`, `type`, `containers` and `applications`. `size` is `null` for now |
| `lpod volume rm NAME --json`, `lpod volumes prune --json` | `{"ok": true}`, or `removed` and `reclaimed` (`null`) for `prune` |
| `lpod networks --json` | `networks` with their `quadlet`, `driver`, `subnet`, `gateway`, `dns`, `internal`, `ipv6`, number of `containers`, `applications` and whether they're the `default` |
| `lpod network rm NAME --json`, `lpod networks prune --json` | `{"ok": true}`, or `removed` and `reclaimed` (`null`) for `prune` |
| `lpod journal --json [--lines N] [--since TIME\|boot] [--priority N]` | `entries`, like `APP logs`, for every unit the installed Quadlets generate and `podman-auto-update.service` |
| `lpod events --json [--since DURATION] [--limit N]` | `events`, newest first: containers that started, stopped, died (with `exit_code`), restarted or were removed, and images that were pulled or removed, with the `application` of each container. `--since` takes `30m`, `24h` and the like (default `24h`), `--limit` up to 999 (default 50) |
| `lpod quadlet check --stdin --json` | `valid`, `errors` and the `units` the files generate, from the Quadlet generator's `-dryrun`, next to the installed Quadlets |
| `lpod quadlet install APP --stdin --json [--replace] [--start]` | `{"ok": true}`. Installs the files as the application `APP` with `podman quadlet install --application`. `--start` starts its units without waiting |
| `lpod quadlet fetch URL --ref REF --json [--path DIR]` | `commit`, `manifest` and `files` (name and content) of a directory of a Git repository, without installing them. `manifest` is the text of the repository's root `pods.yml` (up to 64 KiB, symlinks ignored), or `null`. Takes `https://`, `ssh://` and `git@` URLs |
| `lpod settings --json` | `settings`: the `auto_update` timer (`enabled`, its `OnCalendar` `schedule`, whether a failed update rolls back, `next_run` and `last_run`), `linger` (`null` as root) and `quadlet_dir` |
| `lpod settings set --json [--auto-update on\|off] [--schedule ONCALENDAR] [--rollback on\|off] [--linger on\|off]` | `{"ok": true}`. Changes only the settings given. The schedule and rollback are drop-ins (`lpod-schedule.conf`, `lpod-rollback.conf`) for `podman-auto-update.timer` and `.service`. `--linger` runs `loginctl enable-linger` or `disable-linger`, and does nothing as root |

An application is what `podman quadlet install --application` (or `lpod install`) grouped the Quadlets under. `APP` is its name, or the name of one of its Quadlets without the extension. Its main unit is its pod, or else its kube, or else the container with the shortest name. A Quadlet installed without an application is an application of its own.

On failure, the commands print the reason on stderr and exit with an error. Volume sizes are `null` for now, because working them out takes too long for a page load.

`quadlet check` and `quadlet install` read `{"files": [{"name": ..., "content": ...}]}` on stdin. File names are plain Quadlet file names, so nothing is written outside the Quadlet directory. The commands for containers, images, volumes, networks, the journal, events, pods, Quadlet files and settings only answer with `--json`, so the same words stay free as service names.

The `tests/fixtures/contract-1` folder holds an example payload for each command.

## Troubleshooting

| Command       | Description                                                          |
| -------------- | ---------------------------------------------------------------------- |
| `lpod doctor [--json]` | Check the host for what the services need, and how to fix what's missing |

`lpod doctor` checks:

- that Podman 6.0 or newer runs, with the `podman quadlet` command;
- that jq is installed, for the [JSON commands](#json-contract);
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
  "contract": 1,
  "lpod": "4.0.0",
  "failures": 0,
  "warnings": 1,
  "checks": [
    { "id": "podman", "status": "ok", "message": "Podman 6.1.3 with Quadlet", "fix": "" },
    { "id": "linger", "status": "warn", "message": "Linger is off, so services stop when you log out and don't start at boot", "fix": "loginctl enable-linger me" }
  ]
}
```

The ids are `podman`, `jq`, `systemd`, `linger`, `subids`, `ports`, `idle-timers`, `proxy-certificate`, `app-url` and `failed-services`. Checks that don't apply, such as the rootless ones when running as root, are left out.

## Updating

| Command                      | Description                                                         |
| ----------------------------- | -------------------------------------------------------------------- |
| `lpod self-update [VERSION]` | Upgrade `lpod` and `lpod-setup` in place, to the latest or the given release |
| `lpod --version`             | Show the installed version                                          |

> **Warning:** `remove` and `uninstall` delete the Podman volumes owned by the services they remove. This cannot be undone.

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
| `lpod install PATH`                    | Install Quadlet files (`podman quadlet install`)             |
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

## Proxy

The Pods proxy is an application of its own, `pods-proxy`, installed with `lpod quadlet install pods-proxy --stdin --json` from Pods' `containers/proxy` and removed with `lpod pods-proxy remove --json`. `lpod proxy settings set` picks how it gets certificates, its ports and its image:

| `mode` | Global options | Routes are written as |
| --- | --- | --- |
| `letsencrypt` | `email`, when given | a site per hostname, with a certificate from Let's Encrypt |
| `local` | `local_certs` | a site per hostname, with a certificate from Caddy's local CA (see `proxy certificate`) |
| `wildcard` | `email`, `acme_dns PROVIDER {env.TOKEN}` | handlers inside one `*.DOMAIN` site, with one certificate through a DNS challenge, which `abort`s unknown names |
| `http` | `auto_https off` | a site per hostname over `http://`, for another proxy in front that handles HTTPS |

```bash
echo '{"mode": "wildcard", "email": "you@example.com", "domain": "home.example.com",
    "dns_provider": "cloudflare", "dns_token": "...", "resolvers": "1.1.1.1",
    "http_port": 80, "https_port": 443}' | lpod proxy settings set --stdin --json
```

- It writes `~/.config/pods-proxy/Caddyfile`, with the settings on its `# lpod-proxy:` line. Saving again replaces it, so add your own sites in `sites/*.Caddyfile`, which every mode imports. A Caddyfile lpod didn't write is kept next to it as `Caddyfile.TIMESTAMP.backup` first.
- The Caddyfile's admin API listens on `unix//config/admin.sock` instead of port 2019, and lpod reloads Caddy through it.
- `pods-proxy.container.d/lpod-proxy.conf` publishes `http_port` (default 80) and `https_port` (default 443, also over UDP for HTTP/3; not in `http` mode), and sets the `image`. A drop-in that only publishes ports, such as `install.sh`'s `install.conf` or a `ports.conf`, is removed; one that also sets other things is refused, so ports aren't published twice. Rootless Podman can't publish ports below `net.ipv4.ip_unprivileged_port_start`, which `proxy settings` reports.
- The `wildcard` mode needs Caddy with the DNS provider's plugin, so its image is `ghcr.io/foxws/pods-proxy:main` unless `image` says otherwise; the other modes keep the Quadlet's `caddy:2`. `cloudflare` is the only `dns_provider` so far.
- `dns_token` is kept as the Podman secret `pods-proxy-dns-token` and passed as `CLOUDFLARE_API_TOKEN`. Leave it out to keep the stored one. A drop-in of your own that already passes `CLOUDFLARE_API_TOKEN` keeps doing so, and a new token is then refused.
- Every application's routes are written again for the new layout, then systemd is reloaded and the proxy restarted.

## Routes

The Pods proxy (`pods-proxy`, a Caddy Quadlet; see Pods' README) sends hostnames to the containers of the applications on the host. lpod keeps each application's routes and writes what the proxy needs. How it writes them depends on the proxy's Caddyfile (`~/.config/pods-proxy/Caddyfile`):

| Layout | When | Routes file |
| --- | --- | --- |
| `sites` | The default | `sites/APP.Caddyfile`: a site per route, `{$PROXY_SCHEME:https}://HOST`, each getting its own certificate |
| `handlers` | The Caddyfile has `import sites/*.caddy` inside a wildcard site (`*.home.example.com { … }`), with one certificate through a DNS challenge | `sites/APP.caddy`: a matcher and a `handle` per route, inside that site |

```caddy
# sites/stry.caddy in the handlers layout
@stry host stry.home.example.com stry-ws.home.example.com
handle @stry {
	reverse_proxy host.containers.internal:8000 {
		stream_close_delay 5m
	}
}
```

The routes file's `# lpod-routes:` line holds the routes as given. `pods-proxy.container.d/lpod-routes-APP.conf` has a `Network=` for each `.network` Quadlet of the application, so the proxy reaches its containers by name.

```bash
echo '{"routes": [
    {"role": "app", "host": "stry.home.arpa", "upstream": "systemd-stry:8000"},
    {"role": "ws", "host": "stry-ws.home.arpa", "upstream": "systemd-stry-reverb:6001", "via": "main"},
    {"role": "mail", "host": "stry-mail.home.arpa", "upstream": "systemd-stry-mailpit:8025"}
]}' | lpod stry routes set --stdin --json
```

- A `role` names the route (`app`, `ws`), a `host` is a hostname, and an `upstream` is a container and port. Hosts and roles are unique, and hosts are kept in lowercase. In the `handlers` layout, a host must be one label under the wildcard domain (`stry-ws.home.example.com`, not `ws.stry.home.example.com`), since that's all the certificate covers.
- The first route is the main one unless another says `"main": true`. While the application is on demand, its main route goes to its socket (`host.containers.internal:PORT`), so a request wakes it. `ondemand enable` and `disable` rewrite it. The socket must listen on an address the proxy container reaches, so not `--listen 127.0.0.1`.
- A route with `"via": "main"` goes where the main route goes, and shares its site or matcher, for applications that route their own hostnames (such as stry, whose own Caddy forwards to Reverb and RustFS). Such routes wake a sleeping application too. The others go to their containers, and get a 502 while the application sleeps.
- Every `reverse_proxy` keeps WebSocket connections open for 5 minutes after a reload (`stream_close_delay 5m`), since lpod reloads Caddy whenever routes change.
- When the networks the proxy joins change, lpod reloads systemd and restarts the proxy. Otherwise it reloads Caddy (`caddy reload` in `systemd-pods-proxy`, through the admin socket when the Caddyfile says `admin unix//…`), which keeps open connections up. A proxy that isn't running picks the routes up when it starts.
- `PROXY_SCHEME` is `https` unless the proxy's environment says otherwise, such as `http` behind another proxy that handles HTTPS. The `handlers` layout uses the wildcard site's scheme.

## Timers

`lpod APP timers set` starts containers of an application on a schedule, such as a job that fetches new data every hour. The application names the container and the schedule; lpod writes a systemd timer for it, `APP-NAME.timer` (so `systemctl --user list-timers` shows where it came from), which starts the container's service.

```bash
echo '{"timers": [
    {"name": "retrieve", "container": "spotweb-retrieve", "on_calendar": "hourly",
     "persistent": true, "randomized_delay_sec": "5min", "timeout_start_sec": "infinity"}
]}' | lpod spotweb timers set --stdin --json
```

| Key | Directive | Description |
| --- | --- | --- |
| `name` | | Names the timer: `APP-NAME.timer`. Lowercase letters, digits, `-` and `_`. `idle` is taken by [on-demand](#on-demand) |
| `container` | `Unit=` | A container Quadlet of the application, without `.container`. Each container has at most one timer |
| `on_calendar` | `OnCalendar=` | A calendar event (`hourly`, `*-*-* 02:00`, see systemd.time(7)), or a list of them |
| `persistent` | `Persistent=` | Run once at boot when the host was off at a due time |
| `randomized_delay_sec` | `RandomizedDelaySec=` | Wait up to this long after the due time, so hosts don't all start at once |
| `timeout_start_sec` | `TimeoutStartSec=` | How long the container may take, in a `lpod-timer.conf` drop-in for it, such as `infinity` for a oneshot job that can take hours |

The container should run once and exit: `Type=oneshot` in its `[Service]`, no `[Install]` section (so it doesn't also start at boot), and `HealthCmd=none`. It starts the services it `Requires=`, so a timer also wakes an application that's [on demand](#on-demand) and asleep, which sleeps again after its idle timeout.

Setting the timers replaces the ones the application had, and an empty list removes them. The settings are kept on each timer's `# lpod-timer:` line, so don't edit it by hand: run `set` again instead. `lpod APP timers run NAME` starts the container now, without waiting for it, and `lpod APP remove` removes the timers with the application.

## On-demand

`lpod APP ondemand enable` lets a systemd socket start an application on its first request, and lets the application and the services it needs sleep when nobody uses it. The application only names its units; lpod writes the rest:

| File | Role |
| --- | --- |
| `APP-ondemand.socket` | Listens on `--port` (on the `--listen` address only, when given), and starts the proxy on the first connection |
| `APP-ondemand.service` | `systemd-socket-proxyd`: starts the app, forwards to `127.0.0.1` on `--port` + 10000, and exits after `--idle-timeout` without connections |
| `lpod-ondemand.conf` for the main unit | Publishes the app's `--app-port` only on `127.0.0.1` on `--port` + 10000, stops it from starting at boot, and lets it stop once the proxy is gone (`StopWhenUnneeded=yes`) |
| `APP-idle.service`, `APP-idle.timer` | With `--worker`: every minute, once the app is asleep, the scheduler isn't running and the `--idle-command` succeeds, stops the workers, so the services they need can sleep too |
| `SCHEDULE.timer`, `lpod-ondemand.conf` for the scheduler | With `--schedule`: turns the scheduler container into a oneshot that only runs at the `--schedule-at` minutes |

Starting the main unit on its own doesn't work once it has `StopWhenUnneeded=yes`: nothing needs it, so systemd stops it again. `lpod APP up` therefore starts `APP-ondemand.service` instead, the same way a request does, and the app sleeps again after `--idle-timeout` without connections. `restart` and `redeploy` do the same while the app sleeps, and restart it (and the proxy with it) while it's awake. `lpod APP down` stops it, and the next request wakes it.

```bash
lpod stry ondemand enable --json \
    --port 8000 --app-port 8000 \
    --worker stry-horizon \
    --idle-container stry-horizon --idle-user docker \
    --idle-command "php -d variables_order=EGPCS /app/artisan app:idle" \
    --schedule stry-schedule --schedule-at "*:00" --schedule-at 02:00 \
    --schedule-env "APP_COMMAND=php -d variables_order=EGPCS /app/artisan schedule:run"
```

| Option | Description |
| --- | --- |
| `--port PORT` | The port the socket listens on, on every address unless `--listen` says otherwise (up to 55535). A proxy sends the app's hostname here. The app gets this port + 10000 on `127.0.0.1`, where the proxy forwards to |
| `--app-port PORT` | The port the app listens on inside its container |
| `--listen ADDRESS` | The IP address the socket listens on, such as `127.0.0.1` to keep the app on this machine only, like `cockpit.socket` |
| `--idle-timeout SPAN` | How long the proxy waits without connections before the app stops (default `10min`) |
| `--worker NAME` | A container Quadlet of the app that keeps running after the app stops, such as a queue worker. Repeat it for more. The idle check stops them |
| `--idle-container NAME`, `--idle-command CMD` | The command the idle check runs in that container with `podman exec`. It succeeds when there's no work left. The command is split on whitespace, without quotes |
| `--idle-user USER` | The user `podman exec` runs the idle command as |
| `--schedule NAME` | The scheduler's container Quadlet |
| `--schedule-at TIME` | A minute the scheduler runs in, as `*:MM` (every hour) or `HH:MM` (every day), in UTC. Repeat it for more. Each one also gets a wake-up 30 seconds before it, which starts the services the scheduler needs; the scheduler then waits for the due minute |
| `--schedule-env KEY=VALUE` | An `Environment=` line for the scheduler, such as the command that runs the due tasks once. Repeat it for more. Values can't contain `$` |

The app's container (or pod) is its main unit, as in the [JSON contract](#json-contract). The scheduler also gets the main unit's `Requires=`, with `After=`, since it now runs while the app sleeps. Enabling stops every unit of the application, so the app starts on the next request with the new port. Enabling again replaces what the last enable wrote. `lpod APP ondemand disable` removes it all and starts the app again, which then also starts at boot.

The idle check needs the app's Quadlets to be installed as an application (`podman quadlet install --application`), so its units are known by name. The settings are kept on the `# lpod-ondemand:` line of the socket, so don't edit it by hand: run `enable` again instead.

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
| `lpod APP files remove NAME... --json` | `removed`: the names it removed. Each `NAME` is a file as `APP files` names it: a Quadlet, whose unit stops (`podman quadlet rm --force`), or a drop-in next to one (`NAME.container.d/FILE.conf`). Every name must be a file of the application, or nothing is removed. lpod's own drop-ins (`lpod-*.conf`) are refused. systemd is reloaded once afterwards |
| `lpod APP logs --json [--lines N] [--since TIME]` | `entries`: the journal of the application's units, as `time`, `unit`, `priority` and `message` |
| `lpod APP up\|down\|restart --json [--no-block]` | `{"ok": true}`. `up` and `restart` act on the main unit, `down` stops every unit of the application. An on-demand application that sleeps is woken instead (see [On-demand](#on-demand)). With `--no-block`, it returns without waiting for systemd |
| `lpod APP down --force --json` | `{"ok": true}`. Stops every unit of the application right away: it queues the stop (so `Restart=` doesn't start them again), kills their processes with SIGKILL, waits up to 10 seconds for them to go down, then clears their failed state and start limit with `reset-failed`. For an application stuck starting, in a restart loop or slow to shut down |
| `lpod APP redeploy --json [--no-block]` | `{"ok": true}`. Pulls every image the application's containers use, except ones built on the host (`localhost/...`), then restarts its main unit, or wakes it when it's on demand and asleep. With `--no-block`, a transient unit, `lpod-redeploy-APP`, does the work and logs to the journal |
| `lpod APP secret add NAME --type env\|mount --target TARGET --json` | `{"ok": true}`. Writes the drop-in `lpod-secret-NAME.conf` for the application's main container, next to its Quadlet |
| `lpod APP ondemand enable --json --port PORT --app-port PORT [...]` | `{"ok": true}`. Lets a socket start the application on its first request, and its units sleep when idle. See [On-demand](#on-demand) for the options |
| `lpod APP routes --json` | `routes`: the application's routes (see [Routes](#routes)), each with its `role`, `host`, `upstream`, whether it's the `main` one, and its `target`: the upstream, or the on-demand socket for the main route of an on-demand application |
| `lpod APP routes set --stdin --json` | `{"ok": true}`. Replaces the application's routes with `{"routes": [{"role", "host", "upstream", "main", "via"}]}` on stdin, and applies them. An empty list removes them |
| `lpod APP routes remove --json` | `{"ok": true}`. Removes the application's routes |
| `lpod APP timers --json` | `timers`: the application's timers (see [Timers](#timers)), each with its `name`, `unit` (`APP-NAME.timer`), `container`, `service`, the settings it was set with (`on_calendar` as a list, `persistent`, `randomized_delay_sec`, `timeout_start_sec`, `null` when not given), whether it's `enabled`, whether its container is `running`, its `next_run` and `last_run`, and the `last_result` of the container's service (`success`, `exit-code`, …, or `null` before the first run and while it runs) |
| `lpod APP timers set --stdin --json` | `{"ok": true}`. Replaces the application's timers with `{"timers": [{"name", "container", "on_calendar", "persistent", "randomized_delay_sec", "timeout_start_sec"}]}` on stdin, and enables them. An empty list removes them |
| `lpod APP timers remove --json` | `{"ok": true}`. Removes the application's timers |
| `lpod APP timers run NAME --json` | `{"ok": true}`. Starts the container of the timer `NAME` now, without waiting for it |
| `lpod proxy routes --json` | `proxy` (`installed`, `running`, the `layout` of its Caddyfile, `sites` or `handlers`, and the wildcard `domain` in the `handlers` layout) and `routes`: every application's routes, each with its `application` |
| `lpod proxy settings --json` | `proxy` (`installed`, `running`), `settings` (see [Proxy](#proxy)): whether lpod wrote the Caddyfile (`managed`; if not, the settings are read from it as far as they can be), `mode`, `email`, `domain`, `dns_provider`, `resolvers`, `image` (`null` for the mode's own), `http_port` and `https_port` (as published, or `null`), and whether a `dns_token` is there; `host` (`scope`, and `unprivileged_port_start` when rootless) |
| `lpod proxy settings set --stdin --json` | `{"ok": true, "backup": PATH}`, where `backup` is the hand-written Caddyfile it replaced, or `null`. Takes the settings on stdin |
| `lpod proxy certificate --json` | `certificate`: the PEM of the proxy's local CA root (or `null` until Caddy has made one, or while the proxy isn't running), with its SHA-256 `fingerprint` and when it `expires` (`null` without openssl on the host) |
| `lpod APP ondemand disable --json` | `{"ok": true}`. Removes what `enable` wrote, and starts the application again |
| `lpod APP ondemand status --json` | `ondemand`: whether it's `enabled`, whether the socket is `listening`, whether the app is `awake` (its main unit runs), and the settings it was enabled with: `port`, `local_port`, `app_port`, `listen` (`null` for every address), `idle_timeout`, `workers`, `idle_check` (`container`, `user` and `command`, or `null`) and `schedule` (`container`, `at` and `environment`, or `null`). They're `null` (`workers` empty) when it's off |
| `lpod secret rm NAME [--json]` | `{"ok": true}` |
| `lpod APP remove --json [--volumes] [--images] [--secrets]` | `{"ok": true}`. Stops the application and removes its Quadlets. With the flags, also its volumes, and the images and secrets no other application uses |
| `lpod containers --json` | `containers`: every container, managed by Quadlet or not, with its `state`, `health`, `exit_code`, `application`, `quadlet`, `unit`, `pod`, `ports`, CPU and memory use and `started_at` |
| `lpod container NAME inspect --json` | `container`: what `containers` prints, plus `created_at`, `memory_limit`, `restart_policy`, `restarts`, `user`, `auto_update`, `digest`, `health_check`, `mounts`, `networks`, `secrets` (where each is exposed, never its value) and `environment` (the Quadlet's `Environment=` lines) |
| `lpod container NAME logs --json [--lines N]` | `entries`, like `APP logs`. Containers outside Quadlet are read from `podman logs` |
| `lpod container NAME stats --json` | `stats`: `cpu` (%), `memory` and `memory_limit`, `net_in`, `net_out`, `block_in` and `block_out` (bytes since the container started), `pids` and `sampled_at`. A tool works out rates from two samples |
| `lpod container NAME exec [--stdin] [--user USER] [--workdir DIR] --json -- COMMAND [ARG...]` | `exit_code`, `stdout` and `stderr` (the last 64 KiB of each) of the command, run with `podman exec` in the running container. lpod succeeds when the command ran, whatever its exit code. With `--stdin`, the command reads lpod's input, so a tool can pass a password without it showing in the arguments; without it, the command reads nothing |
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
| `lpod quadlet fetch URL --ref REF --json [--path DIR]` | `commit`, `manifest`, `manifest_path` and `files` (name and content) of a directory of a Git repository, without installing them. `manifest` is the text of the nearest `pods.yml` (up to 64 KiB, symlinks ignored): the one in `DIR`, or else in a directory above it up to the root, or `null`. `manifest_path` is the directory it was found in (`""` for the root, `null` without one), which its sources' paths are relative to. Takes `https://`, `ssh://` and `git@` URLs |
| `lpod quadlet catalog URL --ref REF --json` | `commit` and `applications`: the `path` and `manifest` text of every directory at the root of a Git repository with a `pods.yml` (up to 200, 64 KiB each, symlinks ignored), so a tool such as Pods can offer each as an application |
| `lpod quadlet tags URL --json` | `tags`: the `name` and `commit` (short) of every tag of a Git repository, newest version first (up to 100), read with `git ls-remote` without cloning. An annotated tag gives the commit it points at. Tags `quadlet fetch --ref` would refuse are left out |
| `lpod quadlet head URL --ref BRANCH --json` | `branch` and its newest `commit` (in full), read with `git ls-remote` without cloning, or a `null` commit when the repository has no such branch, such as when `--ref` names a tag |
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

The ids are `podman`, `jq`, `systemd`, `linger`, `subids`, `ports`, `proxy-certificate`, `app-url` and `failed-services`. Checks that don't apply, such as the rootless ones when running as root, are left out.

## Updating

| Command                      | Description                                                         |
| ----------------------------- | -------------------------------------------------------------------- |
| `lpod self-update [VERSION\|BRANCH] [--json]` | Upgrade `lpod` in place: to the latest release, a given one (`v4.0.0`), or a branch (`main`, unverified, reported as `dev-main`). It fails when there's no such release or branch. With `--json`, the installer's output goes to stderr and the new `lpod` prints its `version --json` |
| `lpod --version`             | Show the installed version                                          |

> **Warning:** `remove` and `uninstall` delete the Podman volumes owned by the services they remove. This cannot be undone.

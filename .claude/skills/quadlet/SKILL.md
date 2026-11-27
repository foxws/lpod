---
name: quadlet
description: Reference for how Podman Quadlet units (.container, .pod, .volume, .network, .image, .build, .kube), their drop-ins and the systemd [Unit]/[Service]/[Install] settings around them behave. Use when changing lpod code that reads, writes, installs or checks Quadlets and drop-ins (quadlet install/check/fetch, files, on-demand, timers, routes, the proxy, xdebug), its test host, or explaining why a unit does not generate, start, stop or restart as expected.
---

# Podman Quadlet and systemd

lpod's conventions in AGENTS.md come first. This skill covers what Podman and systemd actually do, so lpod reads and writes Quadlets the way Quadlet itself does. It comes from [foxws/quadlets](https://github.com/foxws/quadlets) (`.claude/skills/quadlet`), whose catalog applications lpod installs, so keep the two in step.

## Where to look things up

Check facts in the docs rather than relying on memory. Keys and defaults change between releases, and lpod needs Podman 6+.

1. Man pages on the host, which match the installed version: `man podman-container.unit`, `podman-pod.unit`, `podman-volume.unit`, `podman-network.unit`, `podman-image.unit`, `podman-build.unit`, `podman-kube.unit`, `podman-systemd.unit`, `podman-quadlet-install`, `podman-auto-update`, `systemd.service`, `systemd.unit`, `systemd.exec`, `systemd.kill`. When Podman isn't installed where you work, ask the user to run them on a host.
2. The Context7 MCP: `/websites/podman_io` for Podman docs and `/systemd/systemd` (v261) for systemd.
3. The web: <https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html>, the per-type `podman-<type>.unit.5.html` pages, and <https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html>.

## How Quadlet works

- Quadlet is a systemd generator. At `daemon-reload` it turns each Quadlet file into a transient `.service`. Never edit the generated unit, and never `systemctl enable` one.
- Only the type section (`[Container]`, `[Pod]`, `[Volume]`, `[Network]`, `[Image]`, `[Build]`, `[Kube]`) and `[Quadlet]` are Quadlet's. `[Unit]`, `[Service]` and `[Install]` pass through to systemd.
- Generated names:

  | File | Service | Podman object (default name) |
  |---|---|---|
  | `x.container` | `x.service` | container `systemd-x` (`ContainerName=`) |
  | `x.pod` | `x-pod.service` | pod `systemd-x` (`PodName=`) |
  | `x.network` | `x-network.service` | network `systemd-x` (`NetworkName=`) |
  | `x.volume` | `x-volume.service` | volume `systemd-x` (`VolumeName=`) |
  | `x.image` / `x.build` | `x-image.service` / `x-build.service` | |
  | `x.kube` | `x.service` | |

- References by file name add the dependency automatically, and the referenced file must exist: `Network=x.network`, `Volume=x.volume:/path`, `Pod=x.pod`, `Image=x.image` or `x.build`, `Network=y.container`. A bare name such as `Volume=x-data:/data` is a plain named volume that Podman creates on first use, with no unit and no dependency.
- `[Unit]` keys (`Requires=`, `After=`, `Wants=`, `BindsTo=`, …) take generated service names, such as `After=immich-database.service`. Requirement and ordering are separate: `Requires=` without `After=` starts both units in parallel.
- `[Install]` is applied at generation. Only `WantedBy=`, `RequiredBy=`, `UpheldBy=` and `Alias=` are supported. Rootless units use `WantedBy=default.target`, because `multi-user.target` doesn't exist in the user manager.
- Quadlet adds a network-online dependency (`podman-user-wait-network-online.service` for rootless units). `[Quadlet] DefaultDependencies=false` turns it off.
- `.container` units get `Type=notify`. Rootless search paths are `~/.config/containers/systemd/`, `$XDG_RUNTIME_DIR/containers/systemd/` and `/etc/containers/systemd/users/[$UID]/`, searched recursively. Quadlet needs cgroup v2.
- Drop-ins: `x.container.d/*.conf` merge in alphabetical order, for every type (`x.volume.d/`, `x.pod.d/`, …). The type-level `container.d/` and dash prefixes (`foo-.container.d/`) also apply, and a file with the same name lower down overrides one higher up. After `podman quadlet install --application=<app>`, they go in `~/.config/containers/systemd/<app>/<unit>.d/`. Checked with the generator on Podman 6.2:
  - A single-value key (`Device=`, `Image=`) is replaced.
  - A repeatable Quadlet key (`Environment=`, `Volume=`, `PublishPort=`) is appended to. An empty assignment (`PublishPort=`) clears it first.
  - `[Unit]`/`[Service]` keys pass through as extra lines, so systemd's rules apply: the last value wins for a single-value key such as `TimeoutStopSec=`, and an empty `ExecStartPre=` resets the list.
- lpod's own drop-ins are `lpod-*.conf` (`lpod-ondemand.conf`, `lpod-timer.conf`, `lpod-proxy.conf`, `lpod-xdebug.conf`), and only lpod's commands change them. Pods adds the owner's drop-ins as `NAME.local.conf` through `APP files write`. Because drop-ins merge in alphabetical order, `lpod-ondemand.conf` sorts before most local names, so a local `PublishPort=` adds to the on-demand port instead of replacing it.
- Templates: `x@.container` becomes `x@.service`. Enabling one through `[Install]` needs `DefaultInstance=`.

## Installing

`podman quadlet install [--application=<name>] [--replace] [--reload-systemd=false] <file|dir> [extra files]`

- A directory install needs `--application`. All its files, including non-Quadlet ones like `.env`, become one application, and removing any of its Quadlets removes the whole application.
- It reloads systemd by default.
- A `.quadlets` file can hold several units separated by `---`, each with a `# FileName=<name>` comment.
- Other subcommands: `podman quadlet list`, `print`, `rm` and `exists`.

## Lifecycle: stop, restart, health

- **Stop:** `StopTimeout=` (Podman, default 10s) must be lower than `[Service] TimeoutStopSec=` (systemd). Otherwise systemd kills `podman` while it's still stopping or removing the container. foxws/quadlets uses 25/30 or 55/60.
- **Restart:** `Restart=on-failure` restarts on a non-clean exit code, signal, timeout, watchdog or OOM. `Restart=always` also restarts on a clean exit. A manual `systemctl stop` never triggers a restart. Restarts count towards `StartLimitIntervalSec=`/`StartLimitBurst=` in `[Unit]`.
- **Clean exit:** exit 0, plus SIGHUP, SIGINT, SIGTERM and SIGPIPE for non-oneshot units, plus anything in `SuccessExitStatus=`. Podman reports a container stopped by SIGTERM as exit code 143, which is not a signal, so `SuccessExitStatus=143` is needed. Multiple `SuccessExitStatus=` lines accumulate and an empty one resets the list, so set it once.
- **Health:** `Notify=healthy` holds the unit in `activating` until the first successful health check, so dependents start only once it's healthy. It needs a `HealthCmd=`. `HealthStartupCmd=` defaults to `HealthCmd=`, and `HealthStartupInterval=` defaults to `0s`, which means no timer, so always set it. `HealthOnFailure=` defaults to `none`, so an unhealthy container just stays up. Under systemd, prefer `kill` or `stop`, which lets `Restart=` take over.
- **Start timeout:** `TimeoutStartSec=` must cover the image pull plus time to healthy. foxws/quadlets uses 300.
- **Reload:** `ReloadCmd=` or `ReloadSignal=` adds an `ExecReload=` (pick one).

## Auto-update

- `AutoUpdate=registry` needs a fully qualified image (`docker.io/library/…`, `ghcr.io/…`). `AutoUpdate=local` restarts the unit when the local image changes.
- `podman-auto-update.timer` runs daily. `podman auto-update --dry-run` shows pending updates. `--rollback` (on by default) goes back to the previous image if the restart fails, which works best when the unit reports readiness (`Notify=`).
- In a pod, updating one container can restart the pod's other containers.

## Containers, pods and storage

- `UserNS=`, `UIDMap=` and `GIDMap=` are ignored with `Pod=`, so set them on the `.pod` instead. Containers in a pod share the pod's network, so they reach each other on `127.0.0.1`, and only the `.pod` gets `PublishPort=` and `Network=`. On that network, other containers resolve the pod's name (`PodName=`), not the names of the containers in it.
- `GroupAdd=keep-groups` (crun only) keeps the host user's groups, for example for `/dev/dri` access. It can't be combined with other groups.
- Volume options: `Z` gives a private SELinux relabel and `z` a shared one. Both write the label to the host files. Use `Z` on a directory one container owns and `z` on one several containers share, and never on `%h` itself or a system directory. `U` recursively chowns to the container user. A bind through a `.volume` with `Type=none`, `Options=bind` and `Device=%h/<dir>` needs the host directory to exist.
- SELinux (enforcing on Fedora hosts) denies a container reading a home directory it hasn't relabeled, such as one labeled `cache_home_t` or `user_home_t`. `Z` or `z` on the `Volume=` line works through a bind `.volume` and labels the directory `container_file_t`, which the user can still edit. The first start relabels the whole tree, which takes a while on a large library.
- `Secret=name,type=env,target=VAR` sets an env var, while `type=mount` (the default) creates `/run/secrets/name` with mode 0444. The secret must exist before the unit starts.
- `.network`: changing a network's settings has no effect until it's removed by hand (`podman network rm`) and the service restarted. `NetworkDeleteOnStop=` defaults to `false`. `Internal=true` cuts off outside access.
- Prefer real keys over `PodmanArgs=`/`GlobalArgs=`: Quadlet validates keys, but passes args through unchecked.
- In `ExecStartPre=`/`ExecStart=`, systemd expands specifiers and variables first, so a literal `%` (as in `printf "%%s\\n"`) is written `%%` and a literal `$` (a shell variable such as `$$1`) is written `$$`.
- Comments go on their own line, `#` or `;`. A `#` after a value becomes part of the value.

## Useful systemd specifiers

`%h` home, `%N` unit name without suffix, `%n` full unit name, `%i` instance, `%t` runtime dir (`$XDG_RUNTIME_DIR`), `%S` state dir, `%E` config dir, `%U` UID.

## Validate and debug

```bash
# Generate without installing. QUADLET_UNIT_DIRS must be absolute: a relative
# path parses nothing and still exits 0.
QUADLET_UNIT_DIRS="$PWD/<dir>" /usr/libexec/podman/quadlet -dryrun -user
# Same, via the systemd generator path
QUADLET_UNIT_DIRS="$PWD/<dir>" /usr/lib/systemd/system-generators/podman-system-generator --user --dryrun
# After installing: errors plus systemd's own checks
systemd-analyze --user --generators=true verify <name>.service
systemctl --user status <name>.service
journalctl --user -u <name>.service
podman healthcheck run <container>
```

A unit missing after `daemon-reload` almost always means the generator rejected it. Run the dry run for that directory and read the error.

## Review checklist

- The file's extension matches its section. A `.container` has exactly one of `Image=` or `Rootfs=`, and the image is fully qualified.
- Every `x.volume`, `x.network`, `x.pod` or `x.image` reference exists in the same directory.
- `[Unit]` dependencies use generated service names, and every `Requires=` has a matching `After=`.
- `StopTimeout=` is below `TimeoutStopSec=`.
- `Notify=healthy` comes with `HealthCmd=` and `HealthStartupInterval=`. The probe binary exists in the image.
- `SuccessExitStatus=` is set once.
- Ports bind to `127.0.0.1` unless exposure is intended and noted in `pods.yml`.
- No secret values in tracked files, and every `Secret=` is in `pods.yml`.
- Every host bind has `Z` (one container) or `z` (shared), never on `%h` itself. `U` only where the container user isn't yours.
- No `UserNS=` on a container that has `Pod=`.
- The dry run for the directory succeeds.

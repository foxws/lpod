---
sidebar_position: 3
---

# Usage

```sh
lpod SERVICE COMMAND [options] [arguments]
```

`SERVICE` is the name of a Quadlet service (your app, or a sibling service like `pgsql`). Quadlet management commands (`setup`, `install`, `remove`, `uninstall`, `list`, `print`, `reload`) skip `SERVICE` — they manage Quadlets themselves rather than talking to a running service.

Quadlet installs the actual Podman container for `SERVICE.container` under the name `systemd-SERVICE` (e.g. `systemd-my-app`), to avoid clashing with unmanaged containers. `lpod` accounts for this automatically for `exec`-based commands (`shell`, `run`, `artisan`, etc.) — you always refer to the service by its plain `SERVICE` name, but expect to see the `systemd-` prefix if you inspect containers directly with `podman ps`.

`lpod --version` (or `-v`/`version`) prints the installed version.

Continue to [Commands reference](./commands.md) for the full list of commands.

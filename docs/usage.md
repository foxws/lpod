---
section: Usage
order: 1
---

# Usage

Every command follows the same shape:

```sh
lpod SERVICE COMMAND [options] [arguments]
```

`SERVICE` is the name of a Quadlet service — your app, or a sibling service like `pgsql`.

A few commands manage Quadlets themselves instead of talking to a running service, so they skip the `SERVICE` name: `setup`, `install`, `remove`, `uninstall`, `list`, `print`, and `reload`.

## Container naming

Quadlet names the actual container `systemd-SERVICE` (for example, `systemd-my-app`) rather than just `SERVICE`, so it doesn't clash with containers you're not managing through Quadlet. `lpod` handles this naming for you automatically in commands that run inside the container (`shell`, `run`, `artisan`, and similar) — you always refer to the service by its plain `SERVICE` name. You'll only see the `systemd-` prefix if you inspect containers directly with `podman ps`.

## Checking the version

Run `lpod --version` (or `-v` / `version`) to print the installed version.

Continue to [Commands reference](./commands.md) for the full list of commands.

---
sidebar_position: 5
---

# Configuration

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

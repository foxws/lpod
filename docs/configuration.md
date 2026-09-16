---
section: Reference
order: 1
---

# Configuration

`lpod` reads its settings from environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `LPOD_PODMAN_BINARY` | `podman` | The Podman binary to use. Must be a single executable path. |
| `LPOD_PUBLISH_PATH` | `podman` | Where `lpod` looks for rendered `.quadlets` files when you run `install`. |
| `APP_PORT` | `80` | Used by `lpod SERVICE open`. |
| `APP_USER` | `$(id -u)` | The user that `exec`-based commands run as inside the container. Defaults to your shell's own user ID. Set it to an empty string to use the image's default user instead. `root-shell` and `root-bash` always run as `root`, regardless of this setting. |

`lpod` also loads `.env` and `.env.$APP_ENV` from the current directory. It automatically forwards known AI coding agent environment variables (from Claude Code, Cursor, Copilot, Codex, Gemini CLI, and others) into `exec`-based commands.

## Links

- [foxws/laravel-podman](https://github.com/foxws/laravel-podman) — the Laravel package `lpod` pairs with
- [Podman Quadlet reference](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)

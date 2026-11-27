# lpod

lpod runs Podman Quadlet applications on a host. [Pods](https://github.com/foxws/pods) runs its `--json` commands over SSH, and people use the rest directly. It's being rewritten from a bash script (`bin/lpod`) in Go, one command at a time, as planned in [#59](https://github.com/foxws/lpod/issues/59).

## Layout

- `bin/lpod`: the bash script, which still implements every command. The release workflow replaces `LPOD_VERSION`, and `LPOD_CONTRACT` is the version of the JSON contract.
- `cmd/lpod` and `internal/`: the Go rewrite. The binary hands every command it doesn't implement yet to the bash script, installed next to it as `lpod.sh` (`internal/handoff`).
- `.devcontainer/`: a development container with Go, shellcheck, jq and gh.
- `install.sh`: the installer, which `lpod self-update` also runs. It checks every download against the release's `SHA256SUMS`.
- `docs/commands.md`: the reference for every command and its JSON output.
- `tests/`:
  - `run.sh` is the test runner;
  - `stubs/` has stand-ins for `podman`, `systemctl`, `journalctl`, `git` and the other commands lpod runs;
  - `host/` is the host they answer from;
  - `fixtures/contract-1/` holds the contract's sample payloads.
- `.github/workflows/`: `tests.yml` runs shellcheck and the tests, and `release.yml` runs on `v*` tags.

## The JSON contract

- Every `--json` command prints one JSON object with `contract`. A command that only changes something prints `{"contract": 1, "ok": true}` (`json_ok`). Errors go to stderr with a non-zero exit code, and Pods shows that text.
- Pods parses these payloads, and its tests keep a copy of `tests/fixtures/contract-1` in `tests/Fixtures/lpod/contract-1`. Keep both copies the same.
- Adding a key doesn't change the contract. Renaming, removing or changing the type of one is a breaking change, which bumps `LPOD_CONTRACT`. Pods refuses a contract it doesn't know.
- Pods and lpod change together, and Pods is young, so breaking changes are fine when they make the contract better. Make them on purpose: bump `LPOD_CONTRACT`, add the new fixtures, and change Pods with them.
- A new command or key also updates its fixture, `docs/commands.md`, and `tests/run.sh`.
- Pods pins the lpod release it needs (`PODS_LPOD_VERSION`, and `LPOD_REF` in its `install.sh`), and treats a command an older lpod lacks as unsupported. So Pods only uses a new command after a release.

## Bash conventions

Follow the code around your change:

- `function name {`, upper-case variable names, `local` inside functions, and a blank line before `return` or `exit` in a block.
- A comment above every function and every block that isn't obvious, written as a sentence that ends in `...`.
- Run Podman as `"${LPOD_PODMAN_BINARY:-podman}"` and systemd as `"${SYSTEMCTL_CMD[@]}"`. That's how the user and system scopes, and the test stand-ins, work.
- `--json` commands call `require_jq` first and build JSON with jq (`--arg`, `--argjson`, `--slurpfile`), never by joining strings.
- The shared jq definitions are in `LPOD_JQ` (times, `systemctl show` properties, Quadlet directives, ports) and `LPOD_SOURCES_JQ` (`source($name)`). Start a program with `"${LPOD_JQ}"'…'` instead of defining them again.
- A listing that reads the host in several places calls `load_host_state` first, so `quadlet list` and `ps` run once. A command that changes the host doesn't, so it never acts on what it read before the change.
- `podman stats` is slow, so only call `host_containers_json --stats` where CPU or memory is shown.
- shellcheck must pass. Disable a check (`# shellcheck disable=SC…`) only where the warning is wrong, on that line.

## Go

lpod in Go is a rewrite from the ground up, not a translation of the bash script, following the plan in [#59](https://github.com/foxws/lpod/issues/59): rootless first, one command at a time, straight to `main`.

- **Rewriting a command:** design it on its own merits, and fix what's wrong with the bash version (see "Found in phase 0" in #59) instead of carrying it over. The Go binary handles the command, and only then is the bash version removed.
- **`tests/run.sh` is the reference:** it runs through the binary, and the Go version passes it. Where Go deliberately behaves differently, change the case (and the fixture, for a contract change) in the same PR, and say so under "Behavior changes, for the release notes".
- **Dependencies:** prefer well-maintained modules over lpod's own logic, such as go-systemd and Podman's own packages. Build without cgo (`CGO_ENABLED=0`), so the binary stays static.
- **Code copied from another project,** such as lerd, keeps that project's license notice.
- **Checks:** `gofmt`, `go vet`, golangci-lint (`.golangci.yml`), `govulncheck` and `go test -race ./...` all run in CI, next to shellcheck and `tests/run.sh`.
- **Version:** releases set it with `-ldflags "-X main.version=…"`, and the bash script keeps its own `LPOD_VERSION`.
- **Trying a build from a checkout:** set `LPOD_SCRIPT` to `bin/lpod`, so the binary hands commands to that script instead of an `lpod.sh` next to it.

## Security

Pods passes values from users, Git repositories and Quadlet files to lpod. Treat every argument, file name and Quadlet value as untrusted:

- Check names with `valid_name` and the other `valid_*` helpers before using them, and put `--` before arguments that could start with `-`.
- Never follow a symlink out of a fetched repository, and never source a file. `.env` is read as text (`load_env_file`), and only for known keys.
- Secret values and file contents come from stdin (`--stdin`), never from arguments, so they don't show up in the process list.
- Only install releases that match their `SHA256SUMS`.

## Areas that need care

- **On-demand** (`APP ondemand enable`): it writes a `.socket`, a `systemd-socket-proxyd` service, an idle-check timer and a scheduler timer. The shell inside those units is escaped for systemd (`$$`, `%%`). Test a change by reading the generated units, as well as the JSON.
- **Proxy and routes:** the Pods proxy is Caddy. lpod keeps its settings and routes as JSON on `# lpod-proxy:` and `# lpod-routes:` lines. It reloads Caddy with `caddy reload` inside the proxy container (`proxy_reload`). That goes through the admin unix socket that lpod's Caddyfile sets (`/config/admin.sock`), while a hand-written Caddyfile may still use Caddy's default port 2019.
- **lpod's own drop-ins** are named `lpod-*.conf` (`lpod-ondemand.conf`, `lpod-timer.conf`, `lpod-proxy.conf`, `lpod-xdebug.conf`). Only lpod's commands change them, and `APP files write` refuses those names.
- For what Podman and systemd actually do with Quadlets and drop-ins, use the `quadlet` skill.

## Testing

- `tests/run.sh` runs the `--json` commands against the stand-ins, which answer from a copy of `tests/host` and log every call. It checks:
  - the output, with jq filters (`check`);
  - that payloads keep every key of their fixture (`keeps_contract`);
  - how often lpod asks the host (`calls`).
- Add a case for every change in behaviour. A stand-in fails on a call it doesn't know ("unexpected call"), so extend it and `tests/host` when a command needs more.
- The stand-ins aren't Podman or systemd. When a change hasn't run on a real host, rootless and rootful, say so in the PR.
- Run `shellcheck bin/lpod install.sh tests/run.sh tests/stubs/*` and `tests/run.sh` before you push. For Go changes, also run `go test ./...`, and `tests/run.sh` through a build of the binary (`LPOD_UNDER_TEST`, see the README).

## Commits, pull requests and releases

- Branch names use lowercase letters and hyphens.
- Commits follow Conventional Commits (`fix:`, `feat:`, `perf:`, `refactor:`, `test:`, `docs:`, `chore:`), with a lowercase summary in the present tense and a body of short `-` bullets.
- A PR description says what changes. When there are behaviour changes, it lists them under "Behavior changes, for the release notes". It ends with "Checked": what ran, and what didn't. PRs are squash-merged.
- **Release:**
  - Tag `vX.Y.Z` on `main`. `release.yml` puts the version in `bin/lpod` and builds `lpod-linux-amd64` and `lpod-linux-arm64`. It attaches those, `lpod` (the bash script), `install.sh` and `SHA256SUMS`, with build provenance, and generates the notes.
  - Above the generated notes, add a short header: who should update, and what changes for them ("Before you update").
  - Then bump `PODS_LPOD_VERSION` and `LPOD_REF` in Pods.

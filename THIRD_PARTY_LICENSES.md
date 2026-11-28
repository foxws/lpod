# Third-party licenses

lpod is MIT licensed (see [LICENSE](LICENSE)). This file lists the code in this repository that comes from other projects, with the license it's under.

## Copied code

Code copied or adapted from another project, such as [lerd](https://github.com/lerd-env/lerd), keeps that project's copyright and license notice in a comment at the top of the file. Each file is listed here, with the project, the commit it came from and its license.

| File | From | Commit | License |
| --- | --- | --- | --- |
| [internal/quadlet/quadlet.go](internal/quadlet/quadlet.go) (`read`) | [Podman](https://github.com/podman-container-tools/podman), `loadUnitDropins` in `cmd/quadlet/main.go` | `3471162efb28760529d852706c493c29245def6d` (v6.2.1) | Apache-2.0 |

## Go modules

The release binaries include the Go modules in [go.mod](go.mod). Each module's license is in its own source, which `go mod download` fetches. A module added to `go.mod` is listed here with its license.

| Module | License |
| --- | --- |
| [github.com/spf13/cobra](https://github.com/spf13/cobra) | Apache-2.0 |
| [github.com/spf13/pflag](https://github.com/spf13/pflag) | BSD-3-Clause |
| [go.podman.io/podman/v6](https://github.com/podman-container-tools/podman), only `pkg/systemd/parser` and `pkg/systemd/quadlet` | Apache-2.0 |
| [go.podman.io/storage](https://github.com/podman-container-tools/container-libs), only `pkg/regexp` | Apache-2.0 |
| [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap) | Apache-2.0, only in Windows builds, so not in lpod's |

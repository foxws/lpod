# Third-party licenses

lpod is MIT licensed (see [LICENSE](LICENSE)). This file lists the code in this repository that comes from other projects, with the license it's under.

## Copied code

Code copied or adapted from another project, such as [lerd](https://github.com/lerd-env/lerd), keeps that project's copyright and license notice in a comment at the top of the file. Each file is listed here, with the project, the commit it came from and its license.

| File | From | Commit | License |
| --- | --- | --- | --- |

None yet.

## Go modules

The release binaries include the Go modules in [go.mod](go.mod). Each module's license is in its own source, which `go mod download` fetches. A module added to `go.mod` is listed here with its license.

| Module | License |
| --- | --- |
| [github.com/spf13/cobra](https://github.com/spf13/cobra) | Apache-2.0 |
| [github.com/spf13/pflag](https://github.com/spf13/pflag) | BSD-3-Clause |
| [go.podman.io/podman/v6](https://github.com/containers/podman), only `pkg/systemd/parser` | Apache-2.0 |
| [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap) | Apache-2.0, only in Windows builds, so not in lpod's |

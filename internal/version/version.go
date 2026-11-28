// Package version implements "lpod --version" and "lpod version --json".
package version

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/contract"
)

// Host is what Collect reads from the host. Tests replace its functions
// with fakes.
type Host struct {
	// Run runs a command and returns its stdout.
	Run func(ctx context.Context, name string, args ...string) (string, error)
	// Getenv reads an environment variable, such as LPOD_PODMAN_BINARY.
	Getenv func(string) string
	// Geteuid tells root, which runs rootful Quadlets, from other users.
	Geteuid func() int
	// Username and Hostname name the user and the host.
	Username func() (string, error)
	Hostname func() (string, error)
}

// System returns the Host of the machine lpod runs on.
func System() Host {
	return Host{
		Run:     run,
		Getenv:  os.Getenv,
		Geteuid: os.Geteuid,
		Username: func() (string, error) {
			current, err := user.Current()
			if err != nil {
				return "", err
			}

			return current.Username, nil
		},
		Hostname: os.Hostname,
	}
}

// timeout is how long Collect waits for podman and systemctl, so a hanging
// Podman doesn't hang the command Pods runs over SSH.
const timeout = 10 * time.Second

// Text is what "lpod --version" prints, such as "lpod 4.3.0".
func Text(lpod string) string {
	return "lpod " + lpod
}

// Collect reads the versions for "lpod version --json". A version that can't
// be read is "", as in the bash script, so the command still answers on a
// host without Podman or systemd.
func Collect(ctx context.Context, lpod string, host Host) contract.Version {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	podman := host.Getenv("LPOD_PODMAN_BINARY")
	if podman == "" {
		podman = "podman"
	}

	payload := contract.Version{
		Contract: contract.Current,
		Lpod:     strings.TrimPrefix(lpod, "v"),
		Scope:    "rootless",
	}

	if out, err := host.Run(ctx, podman, "version", "--format", "{{.Client.Version}}"); err == nil {
		payload.Podman = strings.TrimSpace(out)
	}

	if out, err := host.Run(ctx, "systemctl", "--version"); err == nil {
		payload.Systemd = systemdVersion(out)
	}

	if host.Geteuid() == 0 {
		payload.Scope = "rootful"
	}

	if name, err := host.Username(); err == nil {
		payload.User = name
	}

	if name, err := host.Hostname(); err == nil {
		payload.Hostname = name
	}

	return payload
}

// systemdVersion returns the version from the first line of
// "systemctl --version", such as 258 from "systemd 258 (258.1-1.fc45)".
func systemdVersion(out string) string {
	first, _, _ := strings.Cut(out, "\n")
	fields := strings.Fields(first)

	if len(fields) < 2 {
		return ""
	}

	return fields[1]
}

// run runs name with args, and returns its stdout.
func run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()

	return string(out), err
}

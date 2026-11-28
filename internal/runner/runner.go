// Package runner runs the commands lpod asks the host with, such as podman
// and systemctl. Code takes a Func, so tests give it a fake instead.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Func runs name with args and returns its stdout. An error carries what the
// command printed on stderr.
type Func func(ctx context.Context, name string, args ...string) ([]byte, error)

// Exec runs the command on the host. Its stdin is empty, so a command never
// waits for input that Pods, over SSH, can't give it.
func Exec(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return out, failure(name, args, stderr.String(), err)
	}

	return out, nil
}

// failure describes a command that failed, with the last line it printed on
// stderr, which usually says why.
func failure(name string, args []string, stderr string, err error) error {
	command := name
	if len(args) > 0 {
		command += " " + args[0]
	}

	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if last := lines[len(lines)-1]; last != "" {
		return fmt.Errorf("%s: %s", command, last)
	}

	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("%s exited with %d", command, exit.ExitCode())
	}

	return fmt.Errorf("%s: %w", command, err)
}

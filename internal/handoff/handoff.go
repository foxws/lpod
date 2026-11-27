// Package handoff runs the commands lpod doesn't implement in Go yet with the
// bash script, lpod.sh, which is installed next to the lpod binary.
package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ScriptName is the name of the bash script next to the lpod binary.
const ScriptName = "lpod.sh"

// ScriptEnv names another bash script to hand commands to, such as
// bin/lpod in a checkout of the repository, for tests and development.
const ScriptEnv = "LPOD_SCRIPT"

// ScriptPath returns the bash script to hand a command to: the one ScriptEnv
// names, or else lpod.sh in the directory of executable, after following
// symlinks, so a symlink to lpod elsewhere on the PATH still finds it.
func ScriptPath(executable string, getenv func(string) string) (string, error) {
	if script := getenv(ScriptEnv); script != "" {
		return script, nil
	}

	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("can't resolve %s: %w", executable, err)
	}

	return filepath.Join(filepath.Dir(resolved), ScriptName), nil
}

// Exec replaces this process with the bash script, which gets args and the
// same environment, stdin, stdout and stderr, so it can stream output or read
// a secret from stdin as before. It only returns when that fails.
func Exec(args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("can't find the lpod binary: %w", err)
	}

	script, err := ScriptPath(executable, os.Getenv)
	if err != nil {
		return err
	}

	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("can't find %s, which runs the commands lpod doesn't implement in Go yet. Install lpod again: https://github.com/foxws/lpod#install", script)
	}

	// The script's "#!/usr/bin/env bash" line picks the interpreter, and the
	// script sees its own path as $0, as when it's run directly.
	if err := syscall.Exec(script, append([]string{script}, args...), os.Environ()); err != nil {
		return fmt.Errorf("can't run %s: %w", script, err)
	}

	return nil
}

// Command lpod runs Podman Quadlet applications on a host.
//
// lpod is being ported from bash to Go one command at a time
// (https://github.com/foxws/lpod/issues/59). Until a command is ported, this
// binary hands it to the bash script, lpod.sh, installed next to it.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/handoff"
	lpodversion "github.com/foxws/lpod/internal/version"
)

// version is set when a release is built, with
// -ldflags "-X main.version=4.3.0".
var version = "dev"

func main() {
	if code, ported := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); ported {
		os.Exit(code)
	}

	err := handoff.Exec(os.Args[1:])

	fmt.Fprintf(os.Stderr, "lpod %s: %v\n", version, err)
	os.Exit(1)
}

// run runs the command in args when it's ported to Go, and returns its exit
// code. ported is false for a command the bash script still implements.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) (code int, ported bool) {
	switch {
	case isVersionJSON(args):
		payload := lpodversion.Collect(ctx, version, lpodversion.System())

		return report(contract.Write(stdout, payload), stderr), true

	case isVersionText(args):
		_, err := fmt.Fprintln(stdout, lpodversion.Text(version))

		return report(err, stderr), true
	}

	return 0, false
}

// isVersionText is "lpod --version", "lpod -v", or "lpod version" without
// --json right after it.
func isVersionText(args []string) bool {
	if len(args) == 0 {
		return false
	}

	return args[0] == "--version" || args[0] == "-v" || (args[0] == "version" && !isVersionJSON(args))
}

// isVersionJSON is "lpod version --json". Like the bash script, it ignores
// what comes after --json.
func isVersionJSON(args []string) bool {
	return len(args) >= 2 && args[0] == "version" && args[1] == "--json"
}

// report prints err to stderr, and returns the exit code for it. A failure
// to print the error itself has nowhere else to go.
func report(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}

	_, _ = fmt.Fprintln(stderr, err)

	return 1
}

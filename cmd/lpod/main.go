// Command lpod runs Podman Quadlet applications on a host. The commands it
// implements in Go are cobra commands, and it hands every other command to
// the bash script, lpod.sh (see internal/handoff).
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/foxws/lpod/internal/handoff"
)

// version is set by the release build, with -ldflags "-X main.version=…".
var version = "dev"

func main() {
	if code, ported := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); ported {
		os.Exit(code)
	}

	err := handoff.Exec(os.Args[1:])
	fmt.Fprintf(os.Stderr, "lpod %s: %v\n", version, err)
	os.Exit(1)
}

// run runs args when lpod implements them in Go, and says whether it did.
// It returns the exit code, which is 1 when the command failed.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) (code int, ported bool) {
	root := newRoot(stdout)

	if !isPorted(root, args) {
		return 0, false
	}

	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		// Nothing can be done when stderr itself fails...
		_, _ = fmt.Fprintln(stderr, err)

		return 1, true
	}

	return 0, true
}

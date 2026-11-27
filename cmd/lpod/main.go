// Command lpod runs Podman Quadlet applications on a host.
//
// lpod is being ported from bash to Go one command at a time
// (https://github.com/foxws/lpod/issues/59). Until a command is ported, this
// binary hands it to the bash script, lpod.sh, installed next to it.
package main

import (
	"fmt"
	"os"

	"github.com/foxws/lpod/internal/handoff"
)

// version is set when a release is built, with
// -ldflags "-X main.version=4.3.0".
var version = "dev"

func main() {
	err := handoff.Exec(os.Args[1:])

	fmt.Fprintf(os.Stderr, "lpod %s: %v\n", version, err)
	os.Exit(1)
}

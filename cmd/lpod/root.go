package main

import (
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/runner"
	"github.com/foxws/lpod/internal/systemd"
)

// newRoot returns the "lpod" command with the commands lpod implements in
// Go under it. A cobra command is much like an Artisan command class in
// Laravel: its Use is the signature, and RunE is the handle method.
func newRoot(stdout io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:     "lpod",
		Short:   "Run Podman Quadlet applications",
		Version: version,
		// lpod prints its own errors, once, without the usage...
		SilenceErrors: true,
		SilenceUsage:  true,
		// "lpod help" and "lpod completion" are still the bash script's...
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	root.SetOut(stdout)
	root.SetVersionTemplate("lpod {{.Version}}\n")
	root.AddCommand(newVersion(), newHost())

	return root
}

// isPorted says whether lpod implements args in Go: "--version", or one of
// root's commands. Anything else, such as "lpod stry status", goes to the
// bash script.
func isPorted(root *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}

	if slices.Contains([]string{"--version", "-v"}, args[0]) {
		return true
	}

	command, _, err := root.Find(args)

	return err == nil && command != root
}

// rootful says whether lpod runs as root, which manages the system's
// Quadlets instead of the user's.
func rootful() bool {
	return os.Geteuid() == 0
}

// clients returns the Podman and systemd clients for the host lpod runs on.
func clients() (podman.Client, systemd.Client) {
	return podman.New(os.Getenv, runner.Exec), systemd.New(rootful(), runner.Exec)
}

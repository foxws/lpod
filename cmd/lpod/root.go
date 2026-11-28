package main

import (
	"io"
	"os"
	"slices"
	"strings"

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
	root.AddCommand(newVersion(), newHost(), newAutoUpdate(), newEvents(), newJournal(), newSettings(), newApps(), newApp())
	root.AddCommand(newListings()...)

	return root
}

// jsonOnly marks a command that only answers with --json, such as
// "lpod images --json". Without --json, the same word is free as the name of
// an application or service, as in "lpod images status".
const jsonOnly = "lpod/json-only"

// isPorted says whether lpod implements args in Go: "--version", or one of
// root's commands. Anything else, such as "lpod demo status", goes to the
// bash script.
func isPorted(root *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}

	if slices.Contains([]string{"--version", "-v"}, args[0]) {
		return true
	}

	command, rest, err := root.Find(args)
	if err != nil || command == root {
		return false
	}

	if command.Annotations[jsonOnly] == "" {
		return true
	}

	// The flags are parsed to find what's left: a word after a --json
	// listing is an application named like it, as in "lpod images status
	// --json", or one of its subcommands that's still the bash script's,
	// such as "lpod images prune --json". A flag lpod doesn't know is an
	// error for lpod in Go to print...
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		return false
	}

	if err := command.ParseFlags(rest); err != nil {
		return slices.Contains(rest, "--json")
	}

	asJSON, _ := command.Flags().GetBool("json")

	return asJSON && len(command.Flags().Args()) == 0
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

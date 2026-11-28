package main

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/autoupdate"
	"github.com/foxws/lpod/internal/contract"
)

// newAutoUpdate returns "lpod auto-update --dry-run --json".
func newAutoUpdate() *cobra.Command {
	var dryRun bool

	command := &cobra.Command{
		Use:         "auto-update --dry-run --json",
		Short:       "Ask the registries which containers have a newer image, for lpod images",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{jsonOnly: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// The updates themselves are for podman-auto-update.timer, or for
			// "lpod APP redeploy"...
			if !dryRun {
				return errors.New("Usage: lpod auto-update --dry-run --json") //nolint:staticcheck // printed as it is, like the usage lines of the bash script
			}

			podmanClient, _ := clients()

			if err := autoupdate.DryRun(cmd.Context(), podmanClient, autoupdate.CacheFile(os.Getenv)); err != nil {
				return err
			}

			return contract.Write(cmd.OutOrStdout(), contract.Done())
		},
	}

	command.Flags().BoolVar(&dryRun, "dry-run", false, "only ask, without updating")
	command.Flags().Bool("json", false, "print the outcome as JSON")

	return command
}

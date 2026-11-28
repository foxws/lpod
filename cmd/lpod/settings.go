package main

import (
	"os/user"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/settings"
)

// newSettings returns "lpod settings --json". "lpod settings set --json"
// is still the bash script's.
func newSettings() *cobra.Command {
	command := &cobra.Command{
		Use:         "settings --json",
		Short:       "Show when auto-update runs, whether it rolls back, lingering and the Quadlet directory",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{jsonOnly: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, systemdClient := clients()
			machine := settings.System(systemdClient, rootful(), func() (string, error) {
				current, err := user.Current()
				if err != nil {
					return "", err
				}

				return current.Username, nil
			})

			return contract.Write(cmd.OutOrStdout(), settings.Show(cmd.Context(), machine))
		},
	}

	command.Flags().Bool("json", false, "print them as JSON, which is the only way it prints them")

	return command
}

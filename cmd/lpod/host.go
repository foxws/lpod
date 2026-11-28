package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/host"
)

// newHost returns "lpod host [--json]".
func newHost() *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "host",
		Short: "Show CPU, memory and disk use, the containers, and when auto-update runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			podmanClient, systemdClient := clients()
			payload := host.Collect(cmd.Context(), host.System(podmanClient, systemdClient, rootful()))

			if asJSON {
				return contract.Write(cmd.OutOrStdout(), payload)
			}

			_, err := fmt.Fprint(cmd.OutOrStdout(), host.Text(payload))

			return err
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print it as JSON")

	return command
}

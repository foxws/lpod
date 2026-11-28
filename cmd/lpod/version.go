package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/contract"
	lpodversion "github.com/foxws/lpod/internal/version"
)

// newVersion returns "lpod version [--json]".
func newVersion() *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "version",
		Short: "Print the versions of lpod, Podman and systemd",
		// The bash script took any arguments after "version"...
		Args:               cobra.ArbitraryArgs,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if asJSON {
				return contract.Write(cmd.OutOrStdout(), lpodversion.Collect(cmd.Context(), version, lpodversion.System()))
			}

			_, err := fmt.Fprintln(cmd.OutOrStdout(), lpodversion.Text(version))

			return err
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print the versions of lpod, Podman and systemd, and the host, as JSON")

	return command
}

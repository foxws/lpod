package main

import (
	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/events"
	"github.com/foxws/lpod/internal/journal"
	"github.com/foxws/lpod/internal/runner"
)

// newEvents returns "lpod events --json [--since DURATION] [--limit N]".
func newEvents() *cobra.Command {
	var options events.Options

	command := &cobra.Command{
		Use:         "events --json",
		Short:       "List what happened to the containers and images lately",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{jsonOnly: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			podmanClient, _ := clients()

			payload, err := events.List(cmd.Context(), podmanClient, options)
			if err != nil {
				return err
			}

			return contract.Write(cmd.OutOrStdout(), payload)
		},
	}

	command.Flags().Bool("json", false, "print them as JSON, which is the only way it prints them")
	command.Flags().StringVar(&options.Since, "since", "24h", "how far back, such as 30m or 24h")
	command.Flags().IntVar(&options.Limit, "limit", 50, "how many, the newest, up to 999")

	return command
}

// newJournal returns "lpod journal --json [--lines N] [--since TIME|boot]
// [--priority N]".
func newJournal() *cobra.Command {
	var options journal.Options

	command := &cobra.Command{
		Use:         "journal --json",
		Short:       "Show the journal of the Quadlets' units and of auto-update",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{jsonOnly: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			podmanClient, _ := clients()
			reader := journal.Reader{User: !rootful(), Run: runner.Exec}

			payload, err := journal.List(cmd.Context(), podmanClient, reader, options)
			if err != nil {
				return err
			}

			return contract.Write(cmd.OutOrStdout(), payload)
		},
	}

	command.Flags().Bool("json", false, "print it as JSON, which is the only way it prints it")
	command.Flags().IntVar(&options.Lines, "lines", 500, "how many entries, the last ones, up to 99999")
	command.Flags().StringVar(&options.Since, "since", "", "from a time journalctl reads, such as 2026-11-18 or -1h, or boot")
	command.Flags().IntVar(&options.Priority, "priority", -1, "only entries of this priority and up, from 0 (emergency) to 7 (debug)")

	return command
}

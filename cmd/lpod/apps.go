package main

import (
	"fmt"
	"slices"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/applications"
	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/journal"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/runner"
)

// appCommand is the hidden command the application commands run as, such
// as "lpod demo status --json" as "lpod _app status demo --json". No
// application is named "_app", since a name starts with a letter or digit.
const appCommand = "_app"

// appActions are the actions of "lpod APP ACTION --json" lpod answers in
// Go. The others are still the bash script's.
var appActions = []string{"status"}

// listingNames are the listings an application may be named like, as in
// "lpod pods status --json" for Pods' own application, as the bash script
// has them.
var listingNames = []string{"containers", "images", "volumes", "networks", "journal", "events", "settings", "pods"}

// asAppCommand returns args as the hidden application command when they're
// "APP ACTION ... --json" with an action lpod answers in Go, and APP isn't
// one of root's commands, or is a listing.
func asAppCommand(root *cobra.Command, args []string) ([]string, bool) {
	if len(args) < 2 || !slices.Contains(appActions, args[1]) || !slices.Contains(args, "--json") {
		return args, false
	}

	if command, _, err := root.Find(args[:1]); err == nil && command != root && !slices.Contains(listingNames, args[0]) {
		return args, false
	}

	return append([]string{appCommand, args[1], args[0]}, args[2:]...), true
}

// sources are the clients the application commands read the host with.
func sources() applications.Sources {
	podmanClient, systemdClient := clients()

	return applications.Sources{
		Podman:  podmanClient,
		Systemd: systemdClient,
		Journal: journal.Reader{User: !rootful(), Run: runner.Exec},
		Dirs:    quadlet.SearchDirs(rootful()),
	}
}

// newApps returns "lpod apps [--json]".
func newApps() *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "apps",
		Short: "List the installed applications, with their state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			apps, err := applications.List(cmd.Context(), sources())
			if err != nil {
				return err
			}

			if asJSON {
				payload := contract.Apps{Contract: contract.Current, Applications: []contract.Application{}}
				for _, app := range apps {
					payload.Applications = append(payload.Applications, app.Application)
				}

				return contract.Write(cmd.OutOrStdout(), payload)
			}

			table := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(table, "APPLICATION\tSTATUS\tUNIT\tIMAGE")

			for _, app := range apps {
				image := ""
				if app.Image != nil {
					image = *app.Image
				}

				_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", app.Name, app.Status, app.Unit, image)
			}

			return table.Flush()
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print them as JSON")

	return command
}

// newApp returns the hidden "lpod _app ACTION APP", which runs
// "lpod APP ACTION".
func newApp() *cobra.Command {
	app := &cobra.Command{Use: appCommand + " ACTION APP", Hidden: true}

	status := &cobra.Command{
		Use:   "status APP --json",
		Short: "Show an application, with its volumes, secrets and containers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			found, err := applications.Find(cmd.Context(), sources(), args[0])
			if err != nil {
				return err
			}

			return contract.Write(cmd.OutOrStdout(), contract.AppStatus{Contract: contract.Current, Application: found.ApplicationDetail})
		},
	}

	status.Flags().Bool("json", false, "print it as JSON, which is the only way it prints it")
	app.AddCommand(status)

	return app
}

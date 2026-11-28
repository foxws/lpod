package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/foxws/lpod/internal/autoupdate"
	"github.com/foxws/lpod/internal/containers"
	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/images"
	"github.com/foxws/lpod/internal/networks"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/pods"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/volumes"
)

// newListing returns a "lpod NAME --json" command that prints what list
// reads from Podman. Like the bash script, it only answers with --json.
func newListing(name, short string, list func(context.Context, podman.Client) (any, error)) *cobra.Command {
	command := &cobra.Command{
		Use:         name + " --json",
		Short:       short,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{jsonOnly: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			podmanClient, _ := clients()

			payload, err := list(cmd.Context(), podmanClient)
			if err != nil {
				return err
			}

			return contract.Write(cmd.OutOrStdout(), payload)
		},
	}

	command.Flags().Bool("json", false, "print them as JSON, which is the only way it prints them")

	return command
}

// newListings returns the listings lpod answers in Go.
func newListings() []*cobra.Command {
	return []*cobra.Command{
		newListing("images", "List the images, what uses them, and their updates", func(ctx context.Context, client podman.Client) (any, error) {
			return images.List(ctx, client, autoupdate.CacheFile(os.Getenv))
		}),
		newListing("containers", "List the containers, with their applications and use", func(ctx context.Context, client podman.Client) (any, error) {
			return containers.List(ctx, client)
		}),
		newListing("pods", "List the pods, with their applications and containers", func(ctx context.Context, client podman.Client) (any, error) {
			return pods.List(ctx, client)
		}),
		newListing("volumes", "List the volumes, and the folders containers mount", func(ctx context.Context, client podman.Client) (any, error) {
			return volumes.List(ctx, client, quadlet.Dir(rootful(), os.Getenv))
		}),
		newListing("networks", "List the networks, with what uses them", func(ctx context.Context, client podman.Client) (any, error) {
			return networks.List(ctx, client, quadlet.Dir(rootful(), os.Getenv))
		}),
	}
}

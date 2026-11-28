// Package images implements "lpod images --json": the images on the host,
// the applications that use them, and the updates the last
// "lpod auto-update --dry-run" found.
package images

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"time"

	"github.com/foxws/lpod/internal/autoupdate"
	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
)

// timeout is how long List waits for Podman, so a hanging Podman doesn't
// hang the command Pods runs over SSH.
const timeout = 30 * time.Second

// Host is what List reads: the images, the containers and the Quadlets,
// each from one podman command, and the updates the last dry run kept.
type Host struct {
	Images     []podman.Image
	Containers []podman.Container
	Quadlets   []podman.Quadlet
	Pending    []autoupdate.Pending
}

// List reads the host with client, and the updates kept in cacheFile, and
// lists the images.
func List(ctx context.Context, client podman.Client, cacheFile string) (contract.Images, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var host Host
	var err error

	if host.Images, err = client.Images(ctx); err != nil {
		return contract.Images{}, err
	}

	if host.Containers, err = client.Containers(ctx); err != nil {
		return contract.Images{}, err
	}

	if host.Quadlets, err = client.Quadlets(ctx); err != nil {
		return contract.Images{}, err
	}

	if host.Pending, err = autoupdate.Load(cacheFile); err != nil {
		return contract.Images{}, err
	}

	return contract.Images{Contract: contract.Current, Images: Build(host)}, nil
}

// Build lists the images of host with what uses them. It only reads what
// it's given, so tests call it without a host. Read-only images, of an
// additional image store, are left out, since lpod can't remove them.
func Build(host Host) []contract.Image {
	applications := map[string]string{}
	for _, quadlet := range host.Quadlets {
		applications[quadlet.UnitName] = quadlet.Application()
	}

	listed := []contract.Image{}

	for _, image := range host.Images {
		if image.ReadOnly {
			continue
		}

		listed = append(listed, describe(image, host, applications))
	}

	// Named images first, by repository and tag, then the dangling ones.
	// cmp.Or picks the first comparison that isn't a tie, like orderBy()
	// calls chained in Eloquent...
	slices.SortStableFunc(listed, func(a, b contract.Image) int {
		return cmp.Or(
			compareBool(a.Dangling, b.Dangling),
			cmp.Compare(value(a.Repository), value(b.Repository)),
			cmp.Compare(value(a.Tag), value(b.Tag)),
		)
	})

	return listed
}

// describe is image as the contract lists it. applications has the
// application of each Quadlet's unit.
func describe(image podman.Image, host Host, applications map[string]string) contract.Image {
	described := contract.Image{
		ID:       shortID(image.ID),
		Size:     image.Size,
		UsedBy:   []string{},
		Dangling: len(image.Names) == 0,
	}

	if image.Created > 0 {
		described.CreatedAt = contract.Time{Time: time.Unix(image.Created, 0)}
	}

	if !described.Dangling {
		repository, tag := splitReference(image.Names[0])
		described.Repository, described.Tag = &repository, &tag

		for _, pending := range host.Pending {
			if pending.ImageID == image.ID {
				described.Update = &tag
			}
		}
	}

	for _, container := range host.Containers {
		if container.ImageID == "" || container.ImageID != image.ID {
			continue
		}

		described.Containers++

		if application := applications[container.Unit()]; container.Unit() != "" && application != "" && !slices.Contains(described.UsedBy, application) {
			described.UsedBy = append(described.UsedBy, application)
		}

		if policy := container.Labels["io.containers.autoupdate"]; policy != "" && described.AutoUpdate == nil {
			described.AutoUpdate = &policy
		}
	}

	slices.Sort(described.UsedBy)

	return described
}

// tagPattern finds the tag at the end of a reference, such as "9" in
// "docker.io/library/valkey:9". A ":" before the last "/" is a registry's
// port, as in "localhost:5000/app".
var tagPattern = regexp.MustCompile(`:([^:/]+)$`)

// splitReference splits a reference into its repository and tag. Without a
// tag, it's "latest", as Podman reads it.
func splitReference(reference string) (repository, tag string) {
	match := tagPattern.FindStringSubmatchIndex(reference)
	if match == nil {
		return reference, "latest"
	}

	return reference[:match[0]], reference[match[2]:match[3]]
}

// shortID is the first 12 characters of an ID, as Podman shows it.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}

	return id
}

// compareBool orders false before true.
func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}

	return -1
}

// value is the string s points to, or "".
func value(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// Package containers implements "lpod containers --json": every container
// on the host, managed by Quadlet or not, with its application, ports and
// use of the CPU and memory.
package containers

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
)

// timeout is how long List waits for Podman, so a hanging Podman doesn't
// hang the command Pods runs over SSH.
const timeout = 30 * time.Second

// Host is what Build reads.
type Host struct {
	Containers []podman.Container
	Quadlets   []podman.Quadlet
	// Stats are nil when they weren't read, and then CPU and memory are
	// null.
	Stats []podman.Stat
}

// List reads the containers, the Quadlets and the containers' stats with a
// podman command each, and lists the containers.
func List(ctx context.Context, client podman.Client) (contract.Containers, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var host Host
	var err error

	if host.Containers, err = client.Containers(ctx); err != nil {
		return contract.Containers{}, err
	}

	if host.Quadlets, err = client.Quadlets(ctx); err != nil {
		return contract.Containers{}, err
	}

	// Without the stats, as on a host whose cgroups don't allow them, the
	// list is still worth showing...
	host.Stats, _ = client.Stats(ctx)

	return contract.Containers{Contract: contract.Current, Containers: Build(host)}, nil
}

// Build lists the containers of host, without pods' infra containers, by
// application and then name, with those Quadlet doesn't run last.
func Build(host Host) []contract.Container {
	units := podman.ByUnit(host.Quadlets)
	listed := []contract.Container{}

	for _, container := range host.Containers {
		if !container.IsInfra {
			listed = append(listed, describe(container, units, host.Stats))
		}
	}

	slices.SortStableFunc(listed, func(a, b contract.Container) int {
		return cmp.Or(
			compareNull(a.Application, b.Application),
			cmp.Compare(value(a.Application), value(b.Application)),
			cmp.Compare(a.Name, b.Name),
		)
	})

	return listed
}

// describe is container as the contract lists it.
func describe(container podman.Container, units map[string]podman.Quadlet, stats []podman.Stat) contract.Container {
	described := contract.Container{
		ID:     ShortID(container.ID),
		Name:   first(container.Names),
		Image:  container.Image,
		State:  container.State,
		Health: Health(container.Status),
		Pod:    contract.String(container.PodName),
		Ports:  Ports(container.Ports),
	}

	if container.State == "exited" {
		described.ExitCode = &container.ExitCode
	}

	if quadlet, ok := units[container.Unit()]; ok && container.Unit() != "" {
		described.Application = contract.String(quadlet.Application())
		described.Quadlet = contract.String(quadlet.Name)
		described.Unit = contract.String(quadlet.UnitName)
	}

	if container.State == "running" {
		described.StartedAt = contract.Unix(container.StartedAt)

		for _, stat := range stats {
			if stat.Name == described.Name {
				described.CPU, described.Memory = &stat.CPU, &stat.MemUsage
			}
		}
	}

	return described
}

// healthPattern finds the health in a status such as "Up 2 hours (healthy)".
var healthPattern = regexp.MustCompile(`\((healthy|unhealthy|starting)\)`)

// Health is the health check's state in status, or nil.
func Health(status string) *string {
	match := healthPattern.FindStringSubmatch(status)
	if match == nil {
		return nil
	}

	return &match[1]
}

// Ports writes the published ports as the contract does.
func Ports(ports []podman.Port) []string {
	written := make([]string, 0, len(ports))
	for _, port := range ports {
		written = append(written, port.String())
	}

	return written
}

// ShortID is the first 12 characters of an ID, as Podman shows it.
func ShortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}

	return id
}

// first is the first of names, or "".
func first(names []string) string {
	if len(names) == 0 {
		return ""
	}

	return names[0]
}

// compareNull orders a value before null.
func compareNull(a, b *string) int {
	switch {
	case (a == nil) == (b == nil):
		return 0
	case a == nil:
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

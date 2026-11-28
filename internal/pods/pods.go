// Package pods implements "lpod pods --json": every pod on the host, with
// the application it belongs to, its containers and the ports it publishes.
package pods

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/containers"
	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
)

// timeout is how long List waits for Podman, so a hanging Podman doesn't
// hang the command Pods runs over SSH.
const timeout = 30 * time.Second

// Host is what Build reads.
type Host struct {
	Pods       []podman.Pod
	Containers []podman.Container
	Quadlets   []podman.Quadlet
}

// List reads the pods, the containers and the Quadlets with a podman
// command each, and lists the pods.
func List(ctx context.Context, client podman.Client) (contract.Pods, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var host Host
	var err error

	if host.Pods, err = client.Pods(ctx); err != nil {
		return contract.Pods{}, err
	}

	if host.Containers, err = client.Containers(ctx); err != nil {
		return contract.Pods{}, err
	}

	// A Podman without "podman quadlet" still lists its pods, without
	// their applications...
	host.Quadlets, _ = client.Quadlets(ctx)

	return contract.Pods{Contract: contract.Current, Pods: Build(host)}, nil
}

// Build lists the pods of host by application and then name, with those
// Quadlet doesn't run last.
func Build(host Host) []contract.Pod {
	units := podman.ByUnit(host.Quadlets)
	listed := []contract.Pod{}

	for _, pod := range host.Pods {
		listed = append(listed, describe(pod, host.Containers, units))
	}

	slices.SortStableFunc(listed, func(a, b contract.Pod) int {
		return cmp.Or(
			compareNull(a.Application, b.Application),
			cmp.Compare(value(a.Application), value(b.Application)),
			cmp.Compare(a.Name, b.Name),
		)
	})

	return listed
}

// describe is pod as the contract lists it, with its members among
// containers.
func describe(pod podman.Pod, all []podman.Container, units map[string]podman.Quadlet) contract.Pod {
	described := contract.Pod{
		ID:        containers.ShortID(pod.ID),
		Name:      pod.Name,
		Status:    strings.ToLower(cmp.Or(pod.Status, "unknown")),
		Ports:     []string{},
		Networks:  pod.Networks,
		CreatedAt: contract.ParseTime(pod.Created),
	}

	if described.Networks == nil {
		described.Networks = []string{}
	}

	// The pod's own Quadlet is a .pod, whose unit labels the pod or its
	// infra container...
	own, owned := units[pod.Unit()]
	// ...and the Quadlet of one of its containers says the application,
	// such as the .kube that created them all, or the .container of a .pod
	// that has none...
	var member podman.Quadlet
	var hasMember bool

	for _, container := range all {
		if container.Pod != pod.ID {
			continue
		}

		if container.IsInfra {
			described.Ports = containers.Ports(container.Ports)

			if !owned {
				own, owned = units[container.Unit()]
			}

			continue
		}

		described.Containers++

		if container.State == "running" {
			described.Running++
		}

		if quadlet, ok := units[container.Unit()]; ok && !hasMember {
			member, hasMember = quadlet, true
		}
	}

	// A .kube's pods are its own, as their containers run in its unit...
	if !owned && hasMember && member.Ext() == "kube" {
		own, owned = member, true
	}

	switch {
	case owned:
		described.Application = contract.String(own.Application())
		described.Quadlet = contract.String(own.Name)
		described.Unit = contract.String(own.UnitName)
	case hasMember:
		described.Application = contract.String(member.Application())
	}

	return described
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

// Package networks implements "lpod networks --json": the Podman networks,
// with the containers attached to them and the applications that use them.
package networks

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
)

// timeout is how long List waits for Podman, so a hanging Podman doesn't
// hang the command Pods runs over SSH.
const timeout = 30 * time.Second

// defaultNetwork is Podman's own network.
const defaultNetwork = "podman"

// Host is what Build reads.
type Host struct {
	Networks   []podman.Network
	Units      []quadlet.Unit
	Containers []podman.Container
}

// List reads the networks, the Quadlets and their files, with drop-ins from quadletDirs, and
// the containers, and lists the networks.
func List(ctx context.Context, client podman.Client, quadletDirs []string) (contract.Networks, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var host Host
	var err error

	if host.Networks, err = client.Networks(ctx); err != nil {
		return contract.Networks{}, err
	}

	quadlets, err := client.Quadlets(ctx)
	if err != nil {
		return contract.Networks{}, err
	}

	host.Units = quadlet.Load(quadlets, quadletDirs)

	if host.Containers, err = client.Containers(ctx); err != nil {
		return contract.Networks{}, err
	}

	return contract.Networks{Contract: contract.Current, Networks: Build(host)}, nil
}

// Build lists the networks of host, Podman's own first and then by name.
func Build(host Host) []contract.Network {
	quadlets := map[string]string{}
	byUnit := map[string]podman.Quadlet{}

	// The applications whose Quadlets name a network in Network=, by what
	// they name: a .network, or a network's own name...
	references := map[string][]string{}

	for _, unit := range host.Units {
		byUnit[unit.UnitName] = unit.Quadlet

		switch unit.Ext() {
		case "network":
			quadlets[unit.ResourceName()] = unit.Name
		case "container", "pod":
			for _, network := range unit.All("Network") {
				name, _, _ := strings.Cut(network, ":")
				references[name] = append(references[name], unit.Application())
			}
		}
	}

	listed := []contract.Network{}

	for _, network := range host.Networks {
		described := contract.Network{
			Name:         network.Name,
			Driver:       cmp.Or(network.Driver, "bridge"),
			DNS:          network.DNSEnabled,
			Internal:     network.Internal,
			IPv6:         network.IPv6Enabled,
			Applications: []string{},
			Default:      network.Name == defaultNetwork,
		}

		if len(network.Subnets) > 0 {
			described.Subnet = contract.String(network.Subnets[0].Subnet)
			described.Gateway = contract.String(network.Subnets[0].Gateway)
		}

		applications := slices.Clone(references[network.Name])

		if name, ok := quadlets[network.Name]; ok {
			described.Quadlet = &name
			applications = append(applications, references[name]...)
		}

		for _, container := range host.Containers {
			if !slices.Contains(container.Networks, network.Name) {
				continue
			}

			described.Containers++

			if quadlet, ok := byUnit[container.Unit()]; ok && container.Unit() != "" {
				applications = append(applications, quadlet.Application())
			}
		}

		slices.Sort(applications)
		described.Applications = append(described.Applications, slices.Compact(applications)...)
		listed = append(listed, described)
	}

	slices.SortStableFunc(listed, func(a, b contract.Network) int {
		return cmp.Or(compareBool(b.Default, a.Default), cmp.Compare(a.Name, b.Name))
	})

	return listed
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

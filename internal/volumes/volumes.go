// Package volumes implements "lpod volumes --json": the volumes, and the
// folders of the host that containers mount, with what uses them.
package volumes

import (
	"cmp"
	"context"
	"regexp"
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

// Host is what Build reads.
type Host struct {
	Volumes    []podman.Volume
	Units      []quadlet.Unit
	Containers []podman.Container
	// Inspected has the containers' mounts.
	Inspected []podman.Inspected
}

// List reads the volumes, the Quadlets and their files, with drop-ins from quadletDirs, and
// the containers with their mounts, and lists the volumes.
func List(ctx context.Context, client podman.Client, quadletDirs []string) (contract.Volumes, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var host Host
	var err error

	if host.Volumes, err = client.Volumes(ctx); err != nil {
		return contract.Volumes{}, err
	}

	quadlets, err := client.Quadlets(ctx)
	if err != nil {
		return contract.Volumes{}, err
	}

	host.Units = quadlet.Load(quadlets, quadletDirs)

	if host.Containers, err = client.Containers(ctx); err != nil {
		return contract.Volumes{}, err
	}

	ids := make([]string, 0, len(host.Containers))
	for _, container := range host.Containers {
		ids = append(ids, container.ID)
	}

	if host.Inspected, err = client.Inspect(ctx, ids); err != nil {
		return contract.Volumes{}, err
	}

	return contract.Volumes{Contract: contract.Current, Volumes: Build(host)}, nil
}

// volumeName is a name Volume= takes for a volume, rather than a path or a
// .volume.
var volumeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// Build lists the volumes of host, in Podman's order, and then the folders
// containers mount, by path.
func Build(host Host) []contract.Volume {
	// The Quadlet of each volume: the .volume that creates it, or else the
	// .container that names it in Volume=, which Podman creates then...
	quadlets := map[string]string{}

	type reference struct{ volume, quadlet string }
	var named []reference

	for _, unit := range host.Units {
		switch unit.Ext() {
		case "volume":
			quadlets[unit.ResourceName()] = unit.Name
		case "container":
			for _, volume := range unit.All("Volume") {
				source, _, _ := strings.Cut(volume, ":")

				if volumeName.MatchString(source) && !strings.HasSuffix(source, ".volume") {
					named = append(named, reference{source, unit.Name})
				}
			}
		}
	}

	for _, ref := range named {
		if _, ok := quadlets[ref.volume]; !ok {
			quadlets[ref.volume] = ref.quadlet
		}
	}

	applications := applicationsByName(host.Containers, host.Units)
	listed := []contract.Volume{}

	for _, volume := range host.Volumes {
		users := users(host.Inspected, func(m podman.Mount) bool { return m.Type == "volume" && m.Name == volume.Name })

		described := contract.Volume{
			Name:         volume.Name,
			Driver:       cmp.Or(volume.Driver, "local"),
			Mountpoint:   cmp.Or(volume.Options["device"], volume.Mountpoint),
			Type:         volumeType(volume),
			Containers:   users,
			Applications: applicationsOf(users, applications),
		}

		if name, ok := quadlets[volume.Name]; ok {
			described.Quadlet = &name
		}

		listed = append(listed, described)
	}

	var folders []string
	for _, inspected := range host.Inspected {
		for _, mount := range inspected.Mounts {
			if mount.Type == "bind" && !slices.Contains(folders, mount.Source) {
				folders = append(folders, mount.Source)
			}
		}
	}

	slices.Sort(folders)

	for _, folder := range folders {
		users := users(host.Inspected, func(m podman.Mount) bool { return m.Type == "bind" && m.Source == folder })

		listed = append(listed, contract.Volume{
			Name:         folder,
			Driver:       "local",
			Mountpoint:   folder,
			Type:         "bind",
			Containers:   users,
			Applications: applicationsOf(users, applications),
		})
	}

	return listed
}

// volumeType says what backs volume.
func volumeType(volume podman.Volume) string {
	switch {
	case volume.Anonymous:
		return "anonymous"
	case volume.Options["type"] == "nfs":
		return "nfs"
	case volume.Options["device"] != "":
		return "device"
	}

	return "named"
}

// users are the names of the containers with a mount that matches, sorted.
func users(inspected []podman.Inspected, matches func(podman.Mount) bool) []string {
	names := []string{}

	for _, container := range inspected {
		if slices.ContainsFunc(container.Mounts, matches) && !slices.Contains(names, container.Name) {
			names = append(names, container.Name)
		}
	}

	slices.Sort(names)

	return names
}

// applicationsByName has the application of each container Quadlet runs,
// by the container's name.
func applicationsByName(containers []podman.Container, units []quadlet.Unit) map[string]string {
	quadlets := map[string]podman.Quadlet{}
	for _, unit := range units {
		quadlets[unit.UnitName] = unit.Quadlet
	}

	applications := map[string]string{}

	for _, container := range containers {
		if quadlet, ok := quadlets[container.Unit()]; ok && container.Unit() != "" && len(container.Names) > 0 {
			applications[container.Names[0]] = quadlet.Application()
		}
	}

	return applications
}

// applicationsOf are the applications of the containers names, sorted.
func applicationsOf(names []string, applications map[string]string) []string {
	found := []string{}

	for _, name := range names {
		if application, ok := applications[name]; ok && !slices.Contains(found, application) {
			found = append(found, application)
		}
	}

	slices.Sort(found)

	return found
}

// Package applications implements "lpod apps" and "lpod APP status": the
// installed applications, each from the Quadlets installed together, with
// the state of its main unit and its containers.
package applications

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/containers"
	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/journal"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/systemd"
)

// timeout is how long Read waits for Podman, systemctl and journalctl.
const timeout = 30 * time.Second

// labelPrefix is the prefix of the labels Pods reads from Quadlets.
const labelPrefix = "io.foxws.pods."

// Host is what Build reads.
type Host struct {
	// Units are the Quadlets of the applications to build.
	Units []quadlet.Unit
	// All are every installed Quadlet, so an Image= can name a .build or
	// an .image of another application.
	All []quadlet.Unit
	// States are the systemd properties of the units' services.
	States []systemd.Unit
	// Containers are every container, with their stats.
	Containers []contract.Container
}

// Application is an application with the services of its Quadlets, which
// the commands that act on it use.
type Application struct {
	contract.ApplicationDetail

	Services []string
}

// Sources are the clients Read reads the host with.
type Sources struct {
	Podman  podman.Client
	Systemd systemd.Client
	Journal journal.Reader
	// Dirs are where Quadlet looks for drop-ins (quadlet.SearchDirs).
	Dirs []string
}

// List reads every application, for "lpod apps".
func List(ctx context.Context, sources Sources) ([]Application, error) {
	return read(ctx, sources, func([]quadlet.Unit) []quadlet.Unit { return nil })
}

// Find reads the application name, for "lpod APP status". name is an
// application, or one of its Quadlets without the extension, as the other
// commands of lpod take a service.
func Find(ctx context.Context, sources Sources, name string) (Application, error) {
	var application string

	found, err := read(ctx, sources, func(all []quadlet.Unit) []quadlet.Unit {
		for _, unit := range all {
			if unit.Application() == name || unit.Base() == name {
				application = unit.Application()

				break
			}
		}

		var units []quadlet.Unit
		for _, unit := range all {
			if application != "" && unit.Application() == application {
				units = append(units, unit)
			}
		}

		return units
	})
	if err != nil {
		return Application{}, err
	}

	if application == "" {
		return Application{}, fmt.Errorf("No application or Quadlet named %s is installed.", name) //nolint:staticcheck // printed as it is
	}

	for _, app := range found {
		if app.Name == application {
			return app, nil
		}
	}

	return Application{}, fmt.Errorf("%s has no container, pod or kube Quadlet.", application) //nolint:staticcheck // printed as it is
}

// read reads the host, and builds the applications of the units pick
// returns, or of every unit when it returns nil.
func read(ctx context.Context, sources Sources, pick func([]quadlet.Unit) []quadlet.Unit) ([]Application, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	quadlets, err := sources.Podman.Quadlets(ctx)
	if err != nil {
		return nil, err
	}

	host := Host{All: quadlet.Load(quadlets, sources.Dirs)}

	host.Units = pick(host.All)
	if host.Units == nil {
		host.Units = host.All
	}

	var services []string
	for _, unit := range host.Units {
		if unit.RunsAService() {
			services = append(services, unit.UnitName)
		}
	}

	// Without the states, the applications still show, as "unknown"...
	host.States, _ = sources.Systemd.Show(ctx, services, "ActiveState", "SubState", "Result", "StateChangeTimestamp")

	ps, err := sources.Podman.Containers(ctx)
	if err != nil {
		return nil, err
	}

	// Without the stats, CPU and memory are null...
	stats, _ := sources.Podman.Stats(ctx)
	host.Containers = containers.Build(containers.Host{Containers: ps, Quadlets: quadlets, Stats: stats})
	applications := Build(host)

	// A failed application gets the last error its main unit logged...
	for i, app := range applications {
		if app.Status == "failed" {
			applications[i].Error = contract.Text(sources.Journal.LastError(ctx, app.Unit))
		}
	}

	return applications, nil
}

// Build groups the units of host into applications, by name. An
// application without a .container, .pod or .kube is left out.
func Build(host Host) []Application {
	groups := map[string][]quadlet.Unit{}
	var names []string

	for _, unit := range host.Units {
		name := unit.Application()
		if _, ok := groups[name]; !ok {
			names = append(names, name)
		}

		groups[name] = append(groups[name], unit)
	}

	slices.Sort(names)

	applications := []Application{}

	for _, name := range names {
		if app, ok := build(groups[name], host); ok {
			applications = append(applications, app)
		}
	}

	return applications
}

// build is the application of units.
func build(units []quadlet.Unit, host Host) (Application, bool) {
	containerUnits := ofType(units, "container")
	podUnits := ofType(units, "pod")
	kubeUnits := ofType(units, "kube")

	// The main unit is the .pod, else the .kube, else the .container, with
	// the shortest name, and the primary one the main .container...
	main, ok := shortest(podUnits)
	if !ok {
		main, ok = shortest(kubeUnits)
	}

	if !ok {
		main, ok = shortest(containerUnits)
	}

	if !ok {
		return Application{}, false
	}

	primary, hasPrimary := main, main.Ext() == "container"
	if !hasPrimary {
		primary, hasPrimary = shortest(containerUnits)
	}

	var services []string
	for _, unit := range units {
		services = append(services, unit.UnitName)
	}

	var running []contract.Container
	for _, container := range host.Containers {
		if container.Unit != nil && slices.Contains(services, *container.Unit) {
			running = append(running, container)
		}
	}

	state := stateOf(host.States, main.UnitName)

	app := contract.Application{
		Name:   main.Application(),
		Unit:   main.UnitName,
		Kind:   main.Ext(),
		Status: status(state),
		Since:  contract.Time{Time: systemd.Timestamp(state["StateChangeTimestamp"])},
		Pods:   len(podUnits) + len(kubeUnits),
		Ports:  ports(append(slices.Clone(containerUnits), podUnits...)),
		Labels: main.Labels(labelPrefix),
	}

	if result, ok := state["Result"]; ok {
		app.Result = &result
	}

	if hasPrimary {
		app.Image = image(primary, host.All)
		app.AutoUpdate = contract.String(primary.Last("AutoUpdate"))

		for key, value := range primary.Labels(labelPrefix) {
			if _, set := app.Labels[key]; !set || primary.Name == main.Name {
				app.Labels[key] = value
			}
		}

		for _, container := range running {
			if *container.Unit == primary.UnitName {
				app.Image = contract.String(container.Image)

				break
			}
		}
	}

	if main.Ext() == "kube" {
		app.Containers = len(running)
	} else {
		app.Containers = len(containerUnits)
	}

	for _, container := range running {
		if container.Memory != nil {
			app.Memory = add(app.Memory, *container.Memory)
		}
	}

	return Application{
		ApplicationDetail: contract.ApplicationDetail{
			Application: app,
			Path:        filepath.Dir(main.Path),
			Volumes:     volumes(units),
			Secrets:     secrets(containerUnits),
			Items:       items(containerUnits, host),
		},
		Services: serviceNames(containerUnits, podUnits, kubeUnits),
	}, true
}

// ofType are the units of the type ext.
func ofType(units []quadlet.Unit, ext string) []quadlet.Unit {
	var found []quadlet.Unit

	for _, unit := range units {
		if unit.Ext() == ext {
			found = append(found, unit)
		}
	}

	return found
}

// shortest is the unit with the shortest file name, the first of them on a
// tie.
func shortest(units []quadlet.Unit) (quadlet.Unit, bool) {
	if len(units) == 0 {
		return quadlet.Unit{}, false
	}

	return slices.MinFunc(units, func(a, b quadlet.Unit) int { return cmp.Compare(len(a.Name), len(b.Name)) }), true
}

// stateOf is the systemd properties of the unit id, or none.
func stateOf(states []systemd.Unit, id string) systemd.Unit {
	if unit := systemd.ByID(states, id); unit != nil {
		return unit
	}

	return systemd.Unit{}
}

// status is the application's status from its main unit's ActiveState.
func status(state systemd.Unit) string {
	switch active, ok := state["ActiveState"]; {
	case !ok:
		return "unknown"
	case active == "active":
		return "running"
	default:
		return active
	}
}

// image is what unit's Image= names. A .build is its ImageTag=, and an
// .image its Image=, when all has it.
func image(unit quadlet.Unit, all []quadlet.Unit) *string {
	name := unit.Last("Image")
	if name == "" {
		return nil
	}

	key := map[string]string{".build": "ImageTag", ".image": "Image"}[filepath.Ext(name)]

	if key != "" {
		for _, other := range all {
			if other.Name == name {
				if value := other.Last(key); value != "" {
					return &value
				}
			}
		}
	}

	return &name
}

// portPattern is a PublishPort=, such as "127.0.0.1:8080:80/tcp".
var portPattern = regexp.MustCompile(`^((.+):)?([0-9-]+):([0-9-]+)(/([a-z]+))?$`)

// ports are the PublishPort= of units, as "IP:HOST->CONTAINER/PROTOCOL",
// sorted. One without a host port is left out, since Podman picks it.
func ports(units []quadlet.Unit) []string {
	found := []string{}

	for _, unit := range units {
		for _, value := range unit.All("PublishPort") {
			match := portPattern.FindStringSubmatch(value)
			if match == nil {
				continue
			}

			port := fmt.Sprintf("%s:%s->%s/%s", cmp.Or(match[2], "0.0.0.0"), match[3], match[4], cmp.Or(match[6], "tcp"))
			if !slices.Contains(found, port) {
				found = append(found, port)
			}
		}
	}

	slices.Sort(found)

	return found
}

// volumes are the volumes the .volume Quadlets of units create.
func volumes(units []quadlet.Unit) []contract.ApplicationVolume {
	found := []contract.ApplicationVolume{}

	for _, unit := range ofType(units, "volume") {
		found = append(found, contract.ApplicationVolume{Name: unit.ResourceName()})
	}

	return found
}

// secrets are the Podman secrets the containers use in Secret=, sorted.
func secrets(units []quadlet.Unit) []string {
	found := []string{}

	for _, unit := range units {
		for _, value := range unit.All("Secret") {
			name, _, _ := strings.Cut(value, ",")
			if !slices.Contains(found, name) {
				found = append(found, name)
			}
		}
	}

	slices.Sort(found)

	return found
}

// items are the containers of the .container units: the running one by
// its name, or else what the unit says.
func items(units []quadlet.Unit, host Host) []contract.ApplicationItem {
	found := []contract.ApplicationItem{}

	for _, unit := range units {
		name := cmp.Or(unit.Last("ContainerName"), "systemd-"+unit.Base())
		item := contract.ApplicationItem{Name: name, Image: image(unit, host.All), State: status(stateOf(host.States, unit.UnitName))}

		for _, container := range host.Containers {
			if container.Name == name {
				item.Image = contract.String(container.Image)
				item.State = container.State
				item.CPU, item.Memory = container.CPU, container.Memory

				break
			}
		}

		found = append(found, item)
	}

	return found
}

// serviceNames are the services of the application's containers, pods and
// kubes.
func serviceNames(groups ...[]quadlet.Unit) []string {
	var names []string

	for _, units := range groups {
		for _, unit := range units {
			names = append(names, unit.UnitName)
		}
	}

	return names
}

// add adds bytes to total, which is nil before the first.
func add(total *uint64, bytes uint64) *uint64 {
	sum := bytes
	if total != nil {
		sum += *total
	}

	return &sum
}

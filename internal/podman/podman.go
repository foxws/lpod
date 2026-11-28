// Package podman reads the host through the podman command, the same way a
// person would. Each method runs one podman command and decodes its JSON.
package podman

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/foxws/lpod/internal/runner"
)

// Client runs podman. It's like a small service class in Laravel: the
// command it runs and the Run it runs it with are its dependencies.
type Client struct {
	// Binary is the podman to run, LPOD_PODMAN_BINARY or else "podman".
	Binary string
	Run    runner.Func
}

// New returns a Client for the podman that getenv names in
// LPOD_PODMAN_BINARY, or else the one on the PATH.
func New(getenv func(string) string, run runner.Func) Client {
	binary := getenv("LPOD_PODMAN_BINARY")
	if binary == "" {
		binary = "podman"
	}

	return Client{Binary: binary, Run: run}
}

// Info is the part of "podman info" lpod reads.
type Info struct {
	Host struct {
		NetworkBackend string `json:"networkBackend"`
	} `json:"host"`
	Store struct {
		GraphRoot       string `json:"graphRoot"`
		GraphDriverName string `json:"graphDriverName"`
	} `json:"store"`
}

// Container is the part of a container in "podman ps --all" lpod reads.
type Container struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
	// ImageID is the full ID of the image the container runs, which stays
	// the same when a pull moves the image's name to a newer one.
	ImageID string `json:"ImageID"`
	State   string `json:"State"`
	// Labels has PODMAN_SYSTEMD_UNIT, the unit of a container Quadlet
	// runs, and io.containers.autoupdate, its AutoUpdate= policy.
	Labels map[string]string `json:"Labels"`
	// IsInfra is a pod's infra container, which holds its namespaces.
	IsInfra bool `json:"IsInfra"`
	// Image is the reference the container was created from.
	Image string `json:"Image"`
	// Status is for people, such as "Up 2 hours (healthy)".
	Status   string `json:"Status"`
	ExitCode int    `json:"ExitCode"`
	// StartedAt is in seconds since the epoch, or 0.
	StartedAt int64 `json:"StartedAt"`
	// Pod is the ID of the container's pod, and PodName its name.
	Pod     string `json:"Pod"`
	PodName string `json:"PodName"`
	Ports   []Port `json:"Ports"`
}

// Port is a port a container publishes on the host.
type Port struct {
	HostIP        string `json:"host_ip"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	// Range is the number of ports from HostPort and ContainerPort on, 1
	// for one.
	Range int `json:"range"`
}

// String writes the port as the contract does, such as
// "0.0.0.0:8000->8080/tcp", or "0.0.0.0:8000-8009->8080-8089/tcp" for a
// range.
func (p Port) String() string {
	ip := cmp.Or(p.HostIP, "0.0.0.0")
	host, container := strconv.Itoa(p.HostPort), strconv.Itoa(p.ContainerPort)

	if p.Range > 1 {
		host += "-" + strconv.Itoa(p.HostPort+p.Range-1)
		container += "-" + strconv.Itoa(p.ContainerPort+p.Range-1)
	}

	return net.JoinHostPort(ip, host) + "->" + container + "/" + cmp.Or(p.Protocol, "tcp")
}

// Pod is a pod in "podman pod ps".
type Pod struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
	// Status is such as "Running", "Degraded" or "Exited".
	Status string `json:"Status"`
	// Created is a time such as "2026-11-17T10:00:00.123456789Z".
	Created  string            `json:"Created"`
	Labels   map[string]string `json:"Labels"`
	Networks []string          `json:"Networks"`
}

// Unit is the systemd unit Quadlet runs the pod with, or "".
func (p Pod) Unit() string {
	return p.Labels["PODMAN_SYSTEMD_UNIT"]
}

// Stat is the CPU and memory a running container uses, from "podman stats".
type Stat struct {
	Name string `json:"Name"`
	// CPU is a percentage of one core.
	CPU float64 `json:"CPU"`
	// MemUsage is in bytes.
	MemUsage uint64 `json:"MemUsage"`
}

// Unit is the systemd unit Quadlet runs the container with, or "".
func (c Container) Unit() string {
	return c.Labels["PODMAN_SYSTEMD_UNIT"]
}

// Image is an image in "podman images --all".
type Image struct {
	ID string `json:"Id"`
	// Names are its references, such as "docker.io/library/valkey:9". A
	// dangling image has none.
	Names []string `json:"Names"`
	// Size is in bytes, or nil when Podman didn't say.
	Size *int64 `json:"Size"`
	// Created is in seconds since the epoch.
	Created int64 `json:"Created"`
	// ReadOnly is an image of an additional, read-only image store.
	ReadOnly bool `json:"ReadOnly"`
}

// Update is a container in "podman auto-update --dry-run". Raw keeps every
// key Podman printed, so lpod can store the entry as Podman wrote it.
type Update struct {
	ContainerID string
	// Updated is "pending" when the registry has a newer image.
	Updated string
	Raw     map[string]json.RawMessage
}

// Quadlet is an installed Quadlet in "podman quadlet list".
type Quadlet struct {
	// Name is the file name, such as "demo-app.container".
	Name string `json:"Name"`
	// UnitName is the service Quadlet generates, such as "demo-app.service".
	// ServiceName= sets it, so it comes from the file's author: pass it to
	// systemctl after "--".
	UnitName string `json:"UnitName"`
	Path     string `json:"Path"`
	Status   string `json:"Status"`
	// App is the application "podman quadlet install" installed it with,
	// or "" for a Quadlet installed on its own.
	App string `json:"App"`
}

// Ext is the type of the Quadlet, such as "container" or "volume".
func (q Quadlet) Ext() string {
	return strings.TrimPrefix(path.Ext(q.Name), ".")
}

// Base is the name without its extension, such as "demo-app".
func (q Quadlet) Base() string {
	return strings.TrimSuffix(q.Name, path.Ext(q.Name))
}

// Application is the application the Quadlet belongs to. A Quadlet
// installed on its own is an application of its own.
func (q Quadlet) Application() string {
	if q.App == "" {
		return q.Base()
	}

	return q.App
}

// RunsAService says whether the Quadlet's unit is a long-running service,
// which a .container, .pod or .kube is. The others run once and exit.
func (q Quadlet) RunsAService() bool {
	switch q.Ext() {
	case "container", "pod", "kube":
		return true
	}

	return false
}

// Info runs "podman info".
func (c Client) Info(ctx context.Context) (Info, error) {
	var info Info

	return info, c.json(ctx, &info, "info", "--format", "json")
}

// Containers runs "podman ps --all": every container, running or not.
func (c Client) Containers(ctx context.Context) ([]Container, error) {
	var containers []Container

	return containers, c.json(ctx, &containers, "ps", "--all", "--format", "json")
}

// Pods runs "podman pod ps".
func (c Client) Pods(ctx context.Context) ([]Pod, error) {
	var pods []Pod

	return pods, c.json(ctx, &pods, "pod", "ps", "--format", "json")
}

// Stats runs "podman stats" once, for the running containers. It's slow,
// since it reads each container's cgroup, so only call it where the CPU or
// memory is shown.
func (c Client) Stats(ctx context.Context) ([]Stat, error) {
	out, err := c.Run(ctx, c.Binary, "stats", "--no-stream", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}

	// It prints a JSON object per line...
	var stats []Stat
	decoder := json.NewDecoder(bytes.NewReader(out))

	for decoder.More() {
		var stat Stat
		if err := decoder.Decode(&stat); err != nil {
			return nil, fmt.Errorf("can't read podman stats: %w", err)
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// ByUnit returns quadlets by the unit each one generates.
func ByUnit(quadlets []Quadlet) map[string]Quadlet {
	units := make(map[string]Quadlet, len(quadlets))
	for _, quadlet := range quadlets {
		units[quadlet.UnitName] = quadlet
	}

	return units
}

// Images runs "podman images --all", which includes the intermediate images
// of builds.
func (c Client) Images(ctx context.Context) ([]Image, error) {
	var images []Image

	return images, c.json(ctx, &images, "images", "--all", "--format", "json")
}

// AutoUpdateDryRun runs "podman auto-update --dry-run", which asks the
// registries whether the containers with an AutoUpdate= policy have a newer
// image, without pulling it. Podman prints nothing, or null, when no
// container has a policy.
func (c Client) AutoUpdateDryRun(ctx context.Context) ([]Update, error) {
	out, err := c.Run(ctx, c.Binary, "auto-update", "--dry-run", "--format", "json")
	if err != nil {
		return nil, err
	}

	var raw []map[string]json.RawMessage
	if text := strings.TrimSpace(string(out)); text != "" && text != "null" {
		if err := json.Unmarshal(out, &raw); err != nil {
			return nil, fmt.Errorf("can't read podman auto-update: %w", err)
		}
	}

	updates := make([]Update, 0, len(raw))

	for _, entry := range raw {
		update := Update{Raw: entry}
		_ = json.Unmarshal(entry["ContainerID"], &update.ContainerID)
		_ = json.Unmarshal(entry["Updated"], &update.Updated)
		updates = append(updates, update)
	}

	return updates, nil
}

// Quadlets runs "podman quadlet list": the Quadlets installed for this user,
// or for the system as root.
func (c Client) Quadlets(ctx context.Context) ([]Quadlet, error) {
	var quadlets []Quadlet

	return quadlets, c.json(ctx, &quadlets, "quadlet", "list", "--format", "json")
}

// json runs podman with args and decodes its stdout into v. Podman prints
// "null" for an empty list, which leaves v as it is.
func (c Client) json(ctx context.Context, v any, args ...string) error {
	out, err := c.Run(ctx, c.Binary, args...)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("can't read podman %s: %w", args[0], err)
	}

	return nil
}

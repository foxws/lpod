// Package podman reads the host through the podman command, the same way a
// person would. Each method runs one podman command and decodes its JSON.
package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
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
	// Name is the file name, such as "stry-app.container".
	Name string `json:"Name"`
	// UnitName is the service Quadlet generates, such as "stry-app.service".
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

// Base is the name without its extension, such as "stry-app".
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

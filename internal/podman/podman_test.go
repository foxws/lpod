package podman

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// answer returns a Run that prints out for the podman command want, and
// fails for any other.
func answer(want, out string) func(context.Context, string, ...string) ([]byte, error) {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		if call := strings.Join(append([]string{name}, args...), " "); call != want {
			return nil, errors.New("unexpected call: " + call)
		}

		return []byte(out), nil
	}
}

func TestNewRunsThePodmanTheEnvironmentNames(t *testing.T) {
	getenv := func(name string) string {
		if name == "LPOD_PODMAN_BINARY" {
			return "/opt/podman"
		}

		return ""
	}

	if got := New(getenv, nil).Binary; got != "/opt/podman" {
		t.Errorf("got %s, want /opt/podman", got)
	}

	if got := New(func(string) string { return "" }, nil).Binary; got != "podman" {
		t.Errorf("got %s, want podman", got)
	}
}

func TestInfo(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman info --format json",
		`{"host": {"networkBackend": "netavark"}, "store": {"graphRoot": "/srv/storage", "graphDriverName": "overlay"}}`)}

	info, err := client.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if info.Host.NetworkBackend != "netavark" || info.Store.GraphRoot != "/srv/storage" || info.Store.GraphDriverName != "overlay" {
		t.Errorf("got %+v", info)
	}
}

func TestContainers(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman ps --all --format json",
		`[{"Id": "abc", "Names": ["demo-app"], "State": "running"}]`)}

	containers, err := client.Containers(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(containers) != 1 || containers[0].Names[0] != "demo-app" || containers[0].State != "running" {
		t.Errorf("got %+v", containers)
	}
}

func TestContainersReadsNullAsNone(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman ps --all --format json", "null\n")}

	containers, err := client.Containers(context.Background())
	if err != nil || len(containers) != 0 {
		t.Errorf("got %+v and %v, want none", containers, err)
	}
}

func TestQuadlets(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman quadlet list --format json", `[
		{"Name": "demo-app.container", "UnitName": "demo-app.service", "App": "demo"},
		{"Name": "glance.container", "UnitName": "glance.service", "App": ""},
		{"Name": "demo-data.volume", "UnitName": "demo-data-volume.service", "App": "demo"}
	]`)}

	quadlets, err := client.Quadlets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	type summary struct {
		base, ext, app string
		service        bool
	}

	want := []summary{
		{"demo-app", "container", "demo", true},
		{"glance", "container", "glance", true},
		{"demo-data", "volume", "demo", false},
	}

	for i, q := range quadlets {
		if got := (summary{q.Base(), q.Ext(), q.Application(), q.RunsAService()}); got != want[i] {
			t.Errorf("%s: got %+v, want %+v", q.Name, got, want[i])
		}
	}
}

func TestAFailedCommandIsAnError(t *testing.T) {
	failure := errors.New("podman quadlet: Error: no such command")
	client := Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if _, err := client.Quadlets(context.Background()); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

func TestOutputThatIsntJSONIsAnError(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman info --format json", "Error\n")}

	if _, err := client.Info(context.Background()); err == nil || !strings.Contains(err.Error(), "podman info") {
		t.Errorf("got %v, want an error about podman info", err)
	}
}

func TestContainersReadsTheImageAndLabels(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman ps --all --format json",
		`[{"Id": "abc", "ImageID": "a1a1", "Labels": {"PODMAN_SYSTEMD_UNIT": "demo-app.service"}}, {"Id": "def", "Labels": null, "IsInfra": true}]`)}

	containers, err := client.Containers(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if containers[0].ImageID != "a1a1" || containers[0].Unit() != "demo-app.service" {
		t.Errorf("got %+v", containers[0])
	}

	if containers[1].Unit() != "" || !containers[1].IsInfra {
		t.Errorf("got %+v, want an infra container without a unit", containers[1])
	}
}

func TestImages(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman images --all --format json",
		`[{"Id": "c3c3", "Names": ["ghcr.io/example/demo:4"], "Size": 2000000000, "Created": 1795000000}, {"Id": "a1a1", "Names": null}]`)}

	images, err := client.Images(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 2 || *images[0].Size != 2000000000 || images[0].Created != 1795000000 || images[1].Names != nil || images[1].Size != nil {
		t.Errorf("got %+v", images)
	}
}

func TestAutoUpdateDryRun(t *testing.T) {
	client := Client{Binary: "podman", Run: answer("podman auto-update --dry-run --format json",
		`[{"Unit": "demo-valkey.service", "ContainerID": "2222", "Updated": "pending"}, {"Unit": "demo-app.service", "ContainerID": "1111", "Updated": "false"}]`)}

	updates, err := client.AutoUpdateDryRun(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(updates) != 2 || updates[0].ContainerID != "2222" || updates[0].Updated != "pending" || string(updates[0].Raw["Unit"]) != `"demo-valkey.service"` {
		t.Errorf("got %+v", updates)
	}
}

func TestAutoUpdateDryRunWithoutContainers(t *testing.T) {
	for _, out := range []string{"", "null\n"} {
		client := Client{Binary: "podman", Run: answer("podman auto-update --dry-run --format json", out)}

		updates, err := client.AutoUpdateDryRun(context.Background())
		if err != nil || len(updates) != 0 {
			t.Errorf("%q: got %+v and %v, want none", out, updates, err)
		}
	}
}

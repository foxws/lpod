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
		`[{"Id": "abc", "Names": ["stry-app"], "State": "running"}]`)}

	containers, err := client.Containers(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(containers) != 1 || containers[0].Names[0] != "stry-app" || containers[0].State != "running" {
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
		{"Name": "stry-app.container", "UnitName": "stry-app.service", "App": "stry"},
		{"Name": "glance.container", "UnitName": "glance.service", "App": ""},
		{"Name": "stry-data.volume", "UnitName": "stry-data-volume.service", "App": "stry"}
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
		{"stry-app", "container", "stry", true},
		{"glance", "container", "glance", true},
		{"stry-data", "volume", "stry", false},
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

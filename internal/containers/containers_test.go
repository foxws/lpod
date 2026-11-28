package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

func host() Host {
	return Host{
		Containers: []podman.Container{
			{ID: "3333333333333333", Names: []string{"scratch"}, Image: "docker.io/library/alpine:3", State: "exited", ExitCode: 0, Status: "Exited (0) 1 day ago", PodName: "scratch-pod"},
			{ID: "1111111111111111", Names: []string{"demo-app"}, Image: "ghcr.io/example/demo:4", State: "running", Status: "Up 2 hours (healthy)", StartedAt: 1795000000,
				Labels: map[string]string{"PODMAN_SYSTEMD_UNIT": "demo-app.service"},
				Ports:  []podman.Port{{HostPort: 8000, ContainerPort: 8080, Protocol: "tcp", Range: 1}}},
			{ID: "2222222222222222", Names: []string{"systemd-glance"}, Image: "docker.io/glanceapp/glance:latest", State: "exited", ExitCode: 1, StartedAt: 1795000000,
				Labels: map[string]string{"PODMAN_SYSTEMD_UNIT": "glance.service"}},
			{ID: "5555555555555555", Names: []string{"5555-infra"}, IsInfra: true, State: "running"},
		},
		Quadlets: []podman.Quadlet{
			{Name: "demo-app.container", UnitName: "demo-app.service", App: "demo"},
			{Name: "glance.container", UnitName: "glance.service"},
		},
		Stats: []podman.Stat{{Name: "demo-app", CPU: 2.5, MemUsage: 300000000}, {Name: "systemd-glance", CPU: 9, MemUsage: 9}},
	}
}

func TestBuild(t *testing.T) {
	out := encode(t, Build(host()))

	want := `[` +
		`{"id":"111111111111","name":"demo-app","image":"ghcr.io/example/demo:4","state":"running","health":"healthy","exit_code":null,"application":"demo","quadlet":"demo-app.container","unit":"demo-app.service","pod":null,"ports":["0.0.0.0:8000->8080/tcp"],"cpu":2.5,"memory":300000000,"started_at":"2026-11-18T11:06:40+00:00"},` +
		`{"id":"222222222222","name":"systemd-glance","image":"docker.io/glanceapp/glance:latest","state":"exited","health":null,"exit_code":1,"application":"glance","quadlet":"glance.container","unit":"glance.service","pod":null,"ports":[],"cpu":null,"memory":null,"started_at":null},` +
		`{"id":"333333333333","name":"scratch","image":"docker.io/library/alpine:3","state":"exited","health":null,"exit_code":0,"application":null,"quadlet":null,"unit":null,"pod":"scratch-pod","ports":[],"cpu":null,"memory":null,"started_at":null}` +
		`]`

	if out != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

// encode writes v as JSON on one line, without escaping ">" as
// contract.Write doesn't.
func encode(t *testing.T, v any) string {
	t.Helper()

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		t.Fatal(err)
	}

	return strings.TrimSuffix(out.String(), "\n")
}

func TestBuildWithoutStats(t *testing.T) {
	h := host()
	h.Stats = nil

	for _, container := range Build(h) {
		if container.CPU != nil || container.Memory != nil {
			t.Errorf("%s: got CPU or memory without stats", container.Name)
		}
	}
}

func TestAContainerOfAUnitWithoutAQuadletHasNoApplication(t *testing.T) {
	h := host()
	h.Quadlets = nil

	if got := Build(h)[0]; got.Application != nil || got.Unit != nil {
		t.Errorf("got %+v, want no application", got)
	}
}

func TestHealth(t *testing.T) {
	for status, want := range map[string]string{"Up 2 hours (healthy)": "healthy", "Up 1 second (starting)": "starting", "Up 2 hours (unhealthy)": "unhealthy"} {
		if got := Health(status); got == nil || *got != want {
			t.Errorf("Health(%q) = %v, want %s", status, got, want)
		}
	}

	if Health("Up 2 hours") != nil {
		t.Error("found a health without a check")
	}
}

func TestListShowsContainersWithoutStats(t *testing.T) {
	var calls []string
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))

		if args[0] == "stats" {
			return nil, errors.New("podman stats: Error: cgroups v1 isn't supported")
		}

		return []byte(`[]`), nil
	}}

	payload, err := List(context.Background(), client)
	if err != nil || payload.Contract != 1 || len(calls) != 3 {
		t.Errorf("got %+v and %v after %q", payload, err, calls)
	}
}

func TestListStopsWhenPsFails(t *testing.T) {
	failure := errors.New("podman ps: Error: database is locked")
	client := podman.Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if _, err := List(context.Background(), client); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

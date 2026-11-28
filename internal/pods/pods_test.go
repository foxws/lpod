package pods

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

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

// host has a .pod of demo, a pod of a .kube, and a pod Quadlet doesn't run.
func host() Host {
	unit := func(name string) map[string]string { return map[string]string{"PODMAN_SYSTEMD_UNIT": name} }

	return Host{
		Pods: []podman.Pod{
			{ID: "5555555555555555", Name: "scratch-pod", Status: "Degraded", Created: "2026-11-17T10:00:00.123456789Z", Networks: []string{"podman"}},
			{ID: "aaaaaaaaaaaaaaaa", Name: "demo", Status: "Running", Created: "2026-11-17T11:00:00+01:00", Networks: []string{"demo"}},
			{ID: "kkkkkkkkkkkkkkkk", Name: "paperless", Status: "Running"},
		},
		Containers: []podman.Container{
			{ID: "1", Pod: "5555555555555555", State: "exited"},
			{ID: "2", Pod: "5555555555555555", IsInfra: true, Ports: []podman.Port{{HostIP: "127.0.0.1", HostPort: 9000, ContainerPort: 80, Protocol: "tcp"}}},
			{ID: "3", Pod: "aaaaaaaaaaaaaaaa", IsInfra: true, Labels: unit("demo-pod.service")},
			{ID: "4", Pod: "aaaaaaaaaaaaaaaa", State: "running", Labels: unit("demo-app.service")},
			{ID: "5", Pod: "kkkkkkkkkkkkkkkk", State: "running", Labels: unit("paperless.service")},
			{ID: "6", Pod: "kkkkkkkkkkkkkkkk", State: "exited", Labels: unit("paperless.service")},
			{ID: "7", State: "running"},
		},
		Quadlets: []podman.Quadlet{
			{Name: "demo.pod", UnitName: "demo-pod.service", App: "demo"},
			{Name: "demo-app.container", UnitName: "demo-app.service", App: "demo"},
			{Name: "paperless.kube", UnitName: "paperless.service"},
		},
	}
}

func TestBuild(t *testing.T) {
	want := `[` +
		`{"id":"aaaaaaaaaaaa","name":"demo","status":"running","application":"demo","quadlet":"demo.pod","unit":"demo-pod.service","containers":1,"running":1,"ports":[],"networks":["demo"],"created_at":"2026-11-17T10:00:00+00:00"},` +
		`{"id":"kkkkkkkkkkkk","name":"paperless","status":"running","application":"paperless","quadlet":"paperless.kube","unit":"paperless.service","containers":2,"running":1,"ports":[],"networks":[],"created_at":null},` +
		`{"id":"555555555555","name":"scratch-pod","status":"degraded","application":null,"quadlet":null,"unit":null,"containers":1,"running":0,"ports":["127.0.0.1:9000->80/tcp"],"networks":["podman"],"created_at":"2026-11-17T10:00:00+00:00"}` +
		`]`

	if got := encode(t, Build(host())); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestAPodWithoutItsQuadletHasTheApplicationOfAContainer(t *testing.T) {
	h := host()
	h.Quadlets = h.Quadlets[1:]

	got := Build(h)[0]

	if got.Name != "demo" || got.Application == nil || *got.Application != "demo" || got.Quadlet != nil || got.Unit != nil {
		t.Errorf("got %+v, want demo's application without a Quadlet", got)
	}
}

func TestBuildWithoutPods(t *testing.T) {
	if got := encode(t, Build(Host{})); got != "[]" {
		t.Errorf("got %s, want an empty list", got)
	}
}

func TestListStopsWhenPodmanFails(t *testing.T) {
	failure := errors.New("podman pod: Error: database is locked")
	client := podman.Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if _, err := List(context.Background(), client); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

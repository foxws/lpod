package networks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
)

func host() Host {
	unit := func(name string) map[string]string { return map[string]string{"PODMAN_SYSTEMD_UNIT": name} }
	network := func(name, subnet, gateway string, dns bool) podman.Network {
		n := podman.Network{Name: name, Driver: "bridge", DNSEnabled: dns}
		n.Subnets = append(n.Subnets, struct {
			Subnet  string `json:"subnet"`
			Gateway string `json:"gateway"`
		}{subnet, gateway})

		return n
	}

	return Host{
		Networks: []podman.Network{
			network("demo", "10.89.0.0/24", "10.89.0.1", true),
			network("podman", "10.88.0.0/16", "10.88.0.1", false),
			{Name: "shared", Internal: true},
		},
		Units: []quadlet.Unit{
			quadlet.Parse(podman.Quadlet{Name: "demo.network", UnitName: "demo-network.service", App: "demo"}, "[Network]\nNetworkName=demo\n"),
			quadlet.Parse(podman.Quadlet{Name: "demo-app.container", UnitName: "demo-app.service", App: "demo"}, "[Container]\nNetwork=demo.network\n"),
			quadlet.Parse(podman.Quadlet{Name: "glance.container", UnitName: "glance.service"}, "[Container]\n"),
			quadlet.Parse(podman.Quadlet{Name: "other.pod", UnitName: "other-pod.service", App: "other"}, "[Pod]\nNetwork=shared:ip=10.0.0.5\n"),
		},
		Containers: []podman.Container{
			{Names: []string{"demo-app"}, Networks: []string{"demo"}, Labels: unit("demo-app.service")},
			{Names: []string{"demo-valkey"}, Networks: []string{"demo"}},
			{Names: []string{"systemd-glance"}, Networks: []string{"podman"}, Labels: unit("glance.service")},
			{Names: []string{"scratch"}, Networks: []string{"podman"}},
		},
	}
}

func TestBuild(t *testing.T) {
	out, err := json.Marshal(Build(host()))
	if err != nil {
		t.Fatal(err)
	}

	want := `[` +
		`{"name":"podman","quadlet":null,"driver":"bridge","subnet":"10.88.0.0/16","gateway":"10.88.0.1","dns":false,"internal":false,"ipv6":false,"containers":2,"applications":["glance"],"default":true},` +
		`{"name":"demo","quadlet":"demo.network","driver":"bridge","subnet":"10.89.0.0/24","gateway":"10.89.0.1","dns":true,"internal":false,"ipv6":false,"containers":2,"applications":["demo"],"default":false},` +
		`{"name":"shared","quadlet":null,"driver":"bridge","subnet":null,"gateway":null,"dns":false,"internal":true,"ipv6":false,"containers":0,"applications":["other"],"default":false}` +
		`]`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestBuildWithoutNetworks(t *testing.T) {
	out, _ := json.Marshal(Build(Host{}))
	if string(out) != "[]" {
		t.Errorf("got %s, want an empty list", out)
	}
}

func TestListDoesntInspectContainers(t *testing.T) {
	var calls []string
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))

		return []byte(`[]`), nil
	}}

	if _, err := List(context.Background(), client, t.TempDir()); err != nil || len(calls) != 3 {
		t.Errorf("got %v after %q, want three calls", err, calls)
	}
}

func TestListStopsWhenPodmanFails(t *testing.T) {
	failure := errors.New("podman network: Error: database is locked")
	client := podman.Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if _, err := List(context.Background(), client, t.TempDir()); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

package volumes

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
)

func host() Host {
	unit := func(name string) map[string]string { return map[string]string{"PODMAN_SYSTEMD_UNIT": name} }

	return Host{
		Volumes: []podman.Volume{
			{Name: "demo-data", Driver: "local", Mountpoint: "/volumes/demo-data/_data"},
			{Name: "demo-cache", Mountpoint: "/volumes/demo-cache/_data"},
			{Name: "f00dcafe", Driver: "local", Mountpoint: "/volumes/f00dcafe/_data", Anonymous: true},
			{Name: "media", Driver: "local", Options: map[string]string{"type": "nfs", "device": "nas:/media"}},
			{Name: "disk", Driver: "local", Options: map[string]string{"device": "/dev/sdb1"}},
		},
		Units: []quadlet.Unit{
			quadlet.Parse(podman.Quadlet{Name: "demo-data.volume", UnitName: "demo-data-volume.service", App: "demo"}, "[Volume]\nVolumeName=demo-data\n"),
			quadlet.Parse(podman.Quadlet{Name: "demo-app.container", UnitName: "demo-app.service", App: "demo"},
				"[Container]\nVolume=demo-data.volume:/data\nVolume=demo-cache:/cache\nVolume=/srv/media:/media:ro\n"),
			quadlet.Parse(podman.Quadlet{Name: "demo-valkey.container", UnitName: "demo-valkey.service", App: "demo"}, "[Container]\n"),
		},
		Containers: []podman.Container{
			{ID: "1", Names: []string{"demo-app"}, Labels: unit("demo-app.service")},
			{ID: "2", Names: []string{"demo-valkey"}, Labels: unit("demo-valkey.service")},
			{ID: "3", Names: []string{"scratch"}},
		},
		Inspected: []podman.Inspected{
			{Name: "demo-app", Mounts: []podman.Mount{{Type: "volume", Name: "demo-data"}, {Type: "volume", Name: "demo-cache"}, {Type: "bind", Source: "/srv/media"}}},
			{Name: "demo-valkey", Mounts: []podman.Mount{{Type: "volume", Name: "f00dcafe"}}},
			{Name: "scratch", Mounts: []podman.Mount{{Type: "bind", Source: "/srv/media"}, {Type: "bind", Source: "/etc/hosts"}}},
		},
	}
}

func TestBuild(t *testing.T) {
	out, err := json.Marshal(Build(host()))
	if err != nil {
		t.Fatal(err)
	}

	want := `[` +
		`{"name":"demo-data","quadlet":"demo-data.volume","driver":"local","mountpoint":"/volumes/demo-data/_data","size":null,"type":"named","containers":["demo-app"],"applications":["demo"]},` +
		`{"name":"demo-cache","quadlet":"demo-app.container","driver":"local","mountpoint":"/volumes/demo-cache/_data","size":null,"type":"named","containers":["demo-app"],"applications":["demo"]},` +
		`{"name":"f00dcafe","quadlet":null,"driver":"local","mountpoint":"/volumes/f00dcafe/_data","size":null,"type":"anonymous","containers":["demo-valkey"],"applications":["demo"]},` +
		`{"name":"media","quadlet":null,"driver":"local","mountpoint":"nas:/media","size":null,"type":"nfs","containers":[],"applications":[]},` +
		`{"name":"disk","quadlet":null,"driver":"local","mountpoint":"/dev/sdb1","size":null,"type":"device","containers":[],"applications":[]},` +
		`{"name":"/etc/hosts","quadlet":null,"driver":"local","mountpoint":"/etc/hosts","size":null,"type":"bind","containers":["scratch"],"applications":[]},` +
		`{"name":"/srv/media","quadlet":null,"driver":"local","mountpoint":"/srv/media","size":null,"type":"bind","containers":["demo-app","scratch"],"applications":["demo"]}` +
		`]`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestAVolumeQuadletWinsOverAContainerThatNamesIt(t *testing.T) {
	h := host()
	h.Units[1] = quadlet.Parse(h.Units[1].Quadlet, "[Container]\nVolume=demo-data:/data\n")

	if got := *Build(h)[0].Quadlet; got != "demo-data.volume" {
		t.Errorf("got %s, want the .volume", got)
	}
}

func TestBuildWithoutVolumes(t *testing.T) {
	out, _ := json.Marshal(Build(Host{}))
	if string(out) != "[]" {
		t.Errorf("got %s, want an empty list", out)
	}
}

func TestListInspectsEveryContainerOnce(t *testing.T) {
	var inspects int
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		switch args[0] {
		case "ps":
			return []byte(`[{"Id": "1111"}, {"Id": "2222"}]`), nil
		case "container":
			inspects++

			if len(args) != 5 || args[2] != "--" {
				t.Errorf("ran container %q, want both after --", args[1:])
			}
		}

		return []byte(`[]`), nil
	}}

	if _, err := List(context.Background(), client, t.TempDir()); err != nil || inspects != 1 {
		t.Errorf("got %v after %d inspects, want one", err, inspects)
	}
}

func TestListStopsWhenPodmanFails(t *testing.T) {
	failure := errors.New("podman volume: Error: database is locked")
	client := podman.Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if _, err := List(context.Background(), client, t.TempDir()); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

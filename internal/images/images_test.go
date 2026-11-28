package images

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/autoupdate"
	"github.com/foxws/lpod/internal/podman"
)

func size(bytes int64) *int64 {
	return &bytes
}

// host has stry, whose app runs its own image and valkey, which a
// container outside Quadlet runs too, glance on its own, and an old,
// dangling valkey.
func host() Host {
	return Host{
		Images: []podman.Image{
			{ID: "a1a1a1a1a1a1a1a1", Names: []string{"docker.io/library/valkey:9"}, Size: size(100), Created: 1795000000},
			{ID: "00ff00ff00ff00ff", Size: size(50), Created: 1790000000},
			{ID: "c3c3c3c3c3c3c3c3", Names: []string{"ghcr.io/francoism90/stry:4"}, Size: size(2000)},
			{ID: "d4d4d4d4d4d4d4d4", Names: []string{"localhost:5000/glance"}},
			{ID: "e5e5e5e5e5e5e5e5", Names: []string{"registry.example/shared:1"}, ReadOnly: true},
		},
		Containers: []podman.Container{
			{ID: "1", ImageID: "c3c3c3c3c3c3c3c3", Labels: map[string]string{"PODMAN_SYSTEMD_UNIT": "stry-app.service", "io.containers.autoupdate": "registry"}},
			{ID: "2", ImageID: "a1a1a1a1a1a1a1a1", Labels: map[string]string{"PODMAN_SYSTEMD_UNIT": "stry-valkey.service", "io.containers.autoupdate": "registry"}},
			{ID: "3", ImageID: "d4d4d4d4d4d4d4d4", Labels: map[string]string{"PODMAN_SYSTEMD_UNIT": "glance.service"}},
			{ID: "4", ImageID: "a1a1a1a1a1a1a1a1"},
		},
		Quadlets: []podman.Quadlet{
			{Name: "stry-app.container", UnitName: "stry-app.service", App: "stry"},
			{Name: "stry-valkey.container", UnitName: "stry-valkey.service", App: "stry"},
			{Name: "glance.container", UnitName: "glance.service"},
		},
		Pending: []autoupdate.Pending{{ImageID: "a1a1a1a1a1a1a1a1"}},
	}
}

func TestBuild(t *testing.T) {
	out, err := json.Marshal(Build(host()))
	if err != nil {
		t.Fatal(err)
	}

	want := `[` +
		`{"id":"a1a1a1a1a1a1","repository":"docker.io/library/valkey","tag":"9","size":100,"created_at":"2026-11-18T11:06:40+00:00","used_by":["stry"],"containers":2,"dangling":false,"auto_update":"registry","update":"9"},` +
		`{"id":"c3c3c3c3c3c3","repository":"ghcr.io/francoism90/stry","tag":"4","size":2000,"created_at":null,"used_by":["stry"],"containers":1,"dangling":false,"auto_update":"registry","update":null},` +
		`{"id":"d4d4d4d4d4d4","repository":"localhost:5000/glance","tag":"latest","size":null,"created_at":null,"used_by":["glance"],"containers":1,"dangling":false,"auto_update":null,"update":null},` +
		`{"id":"00ff00ff00ff","repository":null,"tag":null,"size":50,"created_at":"2026-09-21T14:13:20+00:00","used_by":[],"containers":0,"dangling":true,"auto_update":null,"update":null}` +
		`]`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestBuildWithoutImages(t *testing.T) {
	out, err := json.Marshal(Build(Host{}))
	if err != nil || string(out) != "[]" {
		t.Errorf("got %s and %v, want an empty list", out, err)
	}
}

func TestAnUpdateOnlyAppliesToTheImageItWasFoundFor(t *testing.T) {
	h := host()
	// A pull moved valkey:9 to a newer image, and left the old one
	// dangling...
	h.Images[0].ID = "b2b2b2b2b2b2b2b2"

	for _, image := range Build(h) {
		if image.Update != nil {
			t.Errorf("%s: got update %s, want none after the pull", image.ID, *image.Update)
		}
	}
}

func TestSplitReference(t *testing.T) {
	cases := map[string][2]string{
		"docker.io/library/valkey:9": {"docker.io/library/valkey", "9"},
		"localhost:5000/app":         {"localhost:5000/app", "latest"},
		"localhost:5000/app:dev":     {"localhost:5000/app", "dev"},
		"docker.io/library/alpine":   {"docker.io/library/alpine", "latest"},
	}

	for reference, want := range cases {
		if repository, tag := splitReference(reference); repository != want[0] || tag != want[1] {
			t.Errorf("splitReference(%s) = %s, %s, want %s, %s", reference, repository, tag, want[0], want[1])
		}
	}
}

func TestListStopsWhenPodmanFails(t *testing.T) {
	failure := errors.New("podman images: Error: database is locked")
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "images" {
			return nil, failure
		}

		return []byte("[]"), nil
	}}

	if _, err := List(context.Background(), client, filepath.Join(t.TempDir(), "none.json")); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}
}

func TestListReadsEachOnce(t *testing.T) {
	var calls []string
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))

		return []byte("[]"), nil
	}}

	cache := filepath.Join(t.TempDir(), "auto-update.json")
	if err := os.WriteFile(cache, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}

	payload, err := List(context.Background(), client, cache)
	if err != nil {
		t.Fatal(err)
	}

	if payload.Contract != 1 || len(calls) != 3 {
		t.Errorf("got %+v after %q, want contract 1 and three calls", payload, calls)
	}
}

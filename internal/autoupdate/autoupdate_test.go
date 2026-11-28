package autoupdate

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

// fakePodman answers the dry run with dryRun, and "ps" with two containers,
// and records its calls.
func fakePodman(dryRun string, calls *[]string) podman.Client {
	return podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		call := strings.Join(append([]string{name}, args...), " ")
		*calls = append(*calls, call)

		switch call {
		case "podman auto-update --dry-run --format json":
			return []byte(dryRun), nil
		case "podman ps --all --format json":
			return []byte(`[{"Id": "1111", "ImageID": "c3c3"}, {"Id": "2222", "ImageID": "a1a1"}]`), nil
		}

		return nil, errors.New("unexpected call: " + call)
	}}
}

func TestCacheFile(t *testing.T) {
	env := map[string]string{"HOME": "/home/blue"}
	getenv := func(name string) string { return env[name] }

	if got := CacheFile(getenv); got != "/home/blue/.cache/lpod/auto-update.json" {
		t.Errorf("got %s", got)
	}

	env["XDG_CACHE_HOME"] = "/srv/cache"
	if got := CacheFile(getenv); got != "/srv/cache/lpod/auto-update.json" {
		t.Errorf("got %s", got)
	}

	env["XDG_CACHE_HOME"] = "cache"
	if got := CacheFile(getenv); got != "/home/blue/.cache/lpod/auto-update.json" {
		t.Errorf("got %s for a relative XDG_CACHE_HOME", got)
	}
}

func TestDryRunKeepsTheImageOfEachPendingContainer(t *testing.T) {
	var calls []string
	file := filepath.Join(t.TempDir(), "lpod", "auto-update.json")
	client := fakePodman(`[
		{"Unit": "demo-app.service", "ContainerID": "1111", "Updated": "false"},
		{"Unit": "demo-valkey.service", "ContainerID": "2222", "Updated": "pending"},
		{"Unit": "gone.service", "ContainerID": "9999", "Updated": "pending"}
	]`, &calls)

	if err := DryRun(context.Background(), client, file); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	var entries []map[string]any
	if err := json.Unmarshal(content, &entries); err != nil {
		t.Fatal(err)
	}

	if entries[0]["ImageID"] != nil || entries[1]["ImageID"] != "a1a1" || entries[2]["ImageID"] != nil {
		t.Errorf("got %v", entries)
	}

	if entries[1]["Unit"] != "demo-valkey.service" {
		t.Errorf("got %v, want Podman's keys kept", entries[1])
	}

	pending, err := Load(file)
	if err != nil || len(pending) != 1 || pending[0].ImageID != "a1a1" {
		t.Errorf("loaded %+v and %v, want valkey's image", pending, err)
	}
}

func TestDryRunWithoutUpdatesDoesntListContainers(t *testing.T) {
	var calls []string
	file := filepath.Join(t.TempDir(), "auto-update.json")

	if err := DryRun(context.Background(), fakePodman("", &calls), file); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 {
		t.Errorf("ran %q, want only the dry run", calls)
	}

	if content, _ := os.ReadFile(file); string(content) != "[]\n" {
		t.Errorf("kept %q, want an empty list", content)
	}
}

func TestDryRunKeepsTheLastAnswerWhenPodmanFails(t *testing.T) {
	file := filepath.Join(t.TempDir(), "auto-update.json")
	if err := os.WriteFile(file, []byte(`[{"Updated": "pending", "ImageID": "a1a1"}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	failure := errors.New("podman auto-update: Error: registry down")
	client := podman.Client{Binary: "podman", Run: func(context.Context, string, ...string) ([]byte, error) { return nil, failure }}

	if err := DryRun(context.Background(), client, file); !errors.Is(err, failure) {
		t.Errorf("got %v, want %v", err, failure)
	}

	if pending, _ := Load(file); len(pending) != 1 {
		t.Errorf("got %+v, want the last answer", pending)
	}
}

func TestLoadBeforeTheFirstDryRun(t *testing.T) {
	pending, err := Load(filepath.Join(t.TempDir(), "auto-update.json"))
	if err != nil || pending != nil {
		t.Errorf("got %+v and %v, want none", pending, err)
	}
}

func TestLoadLeavesOutEntriesWithoutAnImage(t *testing.T) {
	file := filepath.Join(t.TempDir(), "auto-update.json")
	if err := os.WriteFile(file, []byte(`[{"Updated": "pending"}, {"Updated": "pending", "ImageID": null}, {"Updated": "false", "ImageID": "c3c3"}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	if pending, err := Load(file); err != nil || len(pending) != 0 {
		t.Errorf("got %+v and %v, want none", pending, err)
	}
}

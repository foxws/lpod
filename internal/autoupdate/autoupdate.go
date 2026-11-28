// Package autoupdate implements "lpod auto-update --dry-run", which asks the
// registries which containers have a newer image, and keeps the answer for
// "lpod images".
//
// A dry run asks every registry, so it's slow. Pods runs it now and then,
// and "lpod images" reads what it kept.
package autoupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/foxws/lpod/internal/podman"
)

// CacheFile is where the answer is kept:
// $XDG_CACHE_HOME/lpod/auto-update.json, or ~/.cache/lpod/auto-update.json.
// The bash script keeps it in the same place, in the same shape.
func CacheFile(getenv func(string) string) string {
	cache := getenv("XDG_CACHE_HOME")
	if !filepath.IsAbs(cache) {
		cache = filepath.Join(getenv("HOME"), ".cache")
	}

	return filepath.Join(cache, "lpod", "auto-update.json")
}

// Pending is a container the registry has a newer image for.
type Pending struct {
	// ImageID is the image the container ran at the dry run. Once a pull,
	// an update or a restart moves the name or the container to another
	// image, the update no longer applies to it.
	ImageID string
}

// DryRun runs "podman auto-update --dry-run" and keeps its answer in file,
// each entry with the ImageID of the container's image.
func DryRun(ctx context.Context, client podman.Client, file string) error {
	updates, err := client.AutoUpdateDryRun(ctx)
	if err != nil {
		return err
	}

	images := map[string]string{}

	if hasPending(updates) {
		containers, err := client.Containers(ctx)
		if err != nil {
			return err
		}

		for _, container := range containers {
			images[container.ID] = container.ImageID
		}
	}

	entries := make([]map[string]json.RawMessage, 0, len(updates))

	for _, update := range updates {
		// A container removed since the dry run has no image, so its
		// ImageID is null...
		imageID := json.RawMessage("null")

		if id := images[update.ContainerID]; update.Updated == "pending" && id != "" {
			imageID, _ = json.Marshal(id)
		}

		entry := map[string]json.RawMessage{}
		for key, value := range update.Raw {
			entry[key] = value
		}

		entry["ImageID"] = imageID
		entries = append(entries, entry)
	}

	content, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return writeFile(file, append(content, '\n'))
}

// hasPending says whether any container has an update.
func hasPending(updates []podman.Update) bool {
	for _, update := range updates {
		if update.Updated == "pending" {
			return true
		}
	}

	return false
}

// Load reads the pending updates the last dry run kept in file. Before the
// first one, there are none. An entry of an earlier lpod, without an
// ImageID, is left out: lpod can't tell whether it still applies.
func Load(file string) ([]Pending, error) {
	content, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var entries []struct {
		Updated string  `json:"Updated"`
		ImageID *string `json:"ImageID"`
	}

	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, fmt.Errorf("can't read %s: %w", file, err)
	}

	var pending []Pending

	for _, entry := range entries {
		if entry.Updated == "pending" && entry.ImageID != nil && *entry.ImageID != "" {
			pending = append(pending, Pending{ImageID: *entry.ImageID})
		}
	}

	return pending, nil
}

// writeFile replaces file with content, through a temporary file next to
// it, so "lpod images" never reads half of it.
func writeFile(file string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(file), ".auto-update-*.json")
	if err != nil {
		return err
	}

	// Removing it fails once it's renamed, which is fine...
	defer func() { _ = os.Remove(temporary.Name()) }()

	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()

		return err
	}

	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(temporary.Name(), file)
}

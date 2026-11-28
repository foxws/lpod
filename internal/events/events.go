// Package events implements "lpod events --json": what happened to the
// containers and images lately, newest first, with the application of each
// container.
package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
)

// timeout is how long List waits for Podman.
const timeout = 30 * time.Second

// Options are the flags of "lpod events".
type Options struct {
	// Since is how far back to look, such as "30m" or "24h".
	Since string
	// Limit is how many events, the newest ones, from 1 to 999.
	Limit int
}

// sincePattern is a duration --since takes.
var sincePattern = regexp.MustCompile(`^[0-9]{1,5}[smh]$`)

// actions are the actions listed for each type. The others, such as a
// container's "exec" or "health_status", happen too often to be news.
var actions = map[string][]string{
	"container": {"start", "stop", "died", "remove", "restart"},
	"image":     {"pull", "remove"},
}

// record is a line of "podman events --format json".
type record struct {
	Type     string            `json:"Type"`
	Status   string            `json:"Status"`
	Name     string            `json:"Name"`
	ID       string            `json:"ID"`
	Image    string            `json:"Image"`
	ExitCode *int              `json:"ContainerExitCode"`
	TimeNano int64             `json:"timeNano"`
	Labels   map[string]string `json:"Attributes"`
}

// List reads the events for "lpod events --json".
func List(ctx context.Context, client podman.Client, options Options) (contract.Events, error) {
	if !sincePattern.MatchString(options.Since) {
		return contract.Events{}, errors.New("--since takes a duration such as 30m or 24h.") //nolint:staticcheck // printed as it is
	}

	if options.Limit < 1 || options.Limit > 999 {
		return contract.Events{}, errors.New("--limit takes a number from 1 to 999.") //nolint:staticcheck // printed as it is
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	out, err := client.Run(ctx, client.Binary, "events", "--stream=false", "--since", options.Since,
		"--filter", "type=container", "--filter", "type=image", "--format", "json")
	if err != nil {
		return contract.Events{}, err
	}

	records, err := parse(out)
	if err != nil {
		return contract.Events{}, err
	}

	// Without the Quadlets, the events still show, without applications...
	quadlets, _ := client.Quadlets(ctx)

	return contract.Events{Contract: contract.Current, Events: Build(records, quadlets, options.Limit)}, nil
}

// parse reads Podman's events, one JSON object per line.
func parse(out []byte) ([]record, error) {
	var records []record
	decoder := json.NewDecoder(bytes.NewReader(out))

	for decoder.More() {
		var r record
		if err := decoder.Decode(&r); err != nil {
			return nil, fmt.Errorf("can't read podman events: %w", err)
		}

		records = append(records, r)
	}

	return records, nil
}

// Build lists the limit newest of records, with the applications of their
// containers' Quadlets.
func Build(records []record, quadlets []podman.Quadlet, limit int) []contract.Event {
	units := podman.ByUnit(quadlets)
	listed := []contract.Event{}

	for _, r := range records {
		if !slices.Contains(actions[r.Type], r.Status) {
			continue
		}

		event := contract.Event{Type: r.Type, Action: r.Status, Name: r.Name}

		if r.TimeNano > 0 {
			event.Time = contract.Time{Time: time.Unix(0, r.TimeNano).Truncate(time.Second)}
		}

		if r.Type == "image" {
			if event.Name == "" {
				event.Name = r.ID
			}
		} else {
			event.Image = contract.String(r.Image)
		}

		if quadlet, ok := units[r.Labels["PODMAN_SYSTEMD_UNIT"]]; ok && r.Labels["PODMAN_SYSTEMD_UNIT"] != "" {
			event.Application = contract.String(quadlet.Application())
		}

		if r.Status == "died" {
			event.ExitCode = r.ExitCode
		}

		listed = append(listed, event)
	}

	slices.SortStableFunc(listed, func(a, b contract.Event) int { return b.Time.Compare(a.Time.Time) })

	if len(listed) > limit {
		listed = listed[:limit]
	}

	return listed
}

package events

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

// eventLines are glance dying, an image pull, an exec, which isn't listed,
// and demo-app starting.
const eventLines = `{"Type": "container", "Status": "died", "Name": "systemd-glance", "Image": "docker.io/glanceapp/glance:latest", "ContainerExitCode": 1, "timeNano": 1795000000123456789, "Attributes": {"PODMAN_SYSTEMD_UNIT": "glance.service"}}
{"Type": "image", "Status": "pull", "Name": "docker.io/library/valkey:9", "ID": "a1a1", "timeNano": 1794990000000000000}
{"Type": "container", "Status": "exec", "Name": "demo-app", "timeNano": 1795003600000000000, "Attributes": {"PODMAN_SYSTEMD_UNIT": "demo-app.service"}}
{"Type": "container", "Status": "start", "Name": "demo-app", "Image": "ghcr.io/example/demo:4", "ContainerExitCode": 0, "timeNano": 1795003500000000000, "Attributes": {"PODMAN_SYSTEMD_UNIT": "demo-app.service"}}
{"Type": "image", "Status": "remove", "ID": "b2b2", "timeNano": 1794980000000000000}
`

// fake answers podman events and quadlet list, and records the events call.
func fake(call *string) podman.Client {
	return podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if args[0] == "quadlet" {
			return []byte(`[{"Name": "glance.container", "UnitName": "glance.service"}, {"Name": "demo-app.container", "UnitName": "demo-app.service", "App": "demo"}]`), nil
		}

		*call = strings.Join(append([]string{name}, args...), " ")

		return []byte(eventLines), nil
	}}
}

func TestList(t *testing.T) {
	var call string

	events, err := List(context.Background(), fake(&call), Options{Since: "24h", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}

	if call != "podman events --stream=false --since 24h --filter type=container --filter type=image --format json" {
		t.Errorf("ran %q", call)
	}

	out, _ := json.Marshal(events)
	want := `{"contract":1,"events":[` +
		`{"time":"2026-11-18T12:05:00+00:00","type":"container","action":"start","name":"demo-app","image":"ghcr.io/example/demo:4","application":"demo","exit_code":null},` +
		`{"time":"2026-11-18T11:06:40+00:00","type":"container","action":"died","name":"systemd-glance","image":"docker.io/glanceapp/glance:latest","application":"glance","exit_code":1},` +
		`{"time":"2026-11-18T08:20:00+00:00","type":"image","action":"pull","name":"docker.io/library/valkey:9","image":null,"application":null,"exit_code":null},` +
		`{"time":"2026-11-18T05:33:20+00:00","type":"image","action":"remove","name":"b2b2","image":null,"application":null,"exit_code":null}]}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestListKeepsTheNewest(t *testing.T) {
	var call string

	events, err := List(context.Background(), fake(&call), Options{Since: "1h", Limit: 2})
	if err != nil || len(events.Events) != 2 || events.Events[0].Action != "start" {
		t.Errorf("got %+v and %v, want the two newest", events.Events, err)
	}
}

func TestListRefusesWrongFlags(t *testing.T) {
	var call string

	cases := map[string]Options{
		"--since takes a duration such as 30m or 24h.": {Since: "--all", Limit: 1},
		"--limit takes a number from 1 to 999.":        {Since: "1h", Limit: 1000},
	}

	for want, options := range cases {
		if _, err := List(context.Background(), fake(&call), options); err == nil || err.Error() != want {
			t.Errorf("%+v: got %v, want %s", options, err, want)
		}
	}

	if call != "" {
		t.Errorf("ran %q", call)
	}
}

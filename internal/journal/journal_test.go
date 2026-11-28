package journal

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

// journalLines is what the fake journalctl prints: glance's error, a line
// systemd's user manager logged about demo-app, and one of a MESSAGE that
// isn't UTF-8.
const journalLines = `{"__REALTIME_TIMESTAMP": "1795000000123456", "_SYSTEMD_USER_UNIT": "glance.service", "PRIORITY": "3", "MESSAGE": "Error: config: glance.yml: no such file or directory\n"}
{"__REALTIME_TIMESTAMP": "1795000001000000", "_SYSTEMD_USER_UNIT": "user@1000.service", "USER_UNIT": "demo-app.service", "PRIORITY": "6", "MESSAGE": "Started demo-app.service."}

{"__REALTIME_TIMESTAMP": "1795003600000000", "_SYSTEMD_USER_UNIT": "podman-auto-update.service", "MESSAGE": [83, 116, 97, 114, 116, 101, 100, 10]}
`

// fake runs quadlet list and journalctl, and records journalctl's call.
func fake(call *string) (podman.Client, Reader) {
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "podman" {
			return []byte(`[{"Name": "glance.container", "UnitName": "glance.service"}, {"Name": "evil.container", "UnitName": "--host=evil.service"}]`), nil
		}

		*call = strings.Join(append([]string{name}, args...), " ")

		return []byte(journalLines), nil
	}

	return podman.Client{Binary: "podman", Run: run}, Reader{User: true, Run: run}
}

func TestList(t *testing.T) {
	var call string
	client, reader := fake(&call)

	journal, err := List(context.Background(), client, reader, Options{Lines: 500, Priority: -1})
	if err != nil {
		t.Fatal(err)
	}

	if call != "journalctl --user --unit=podman-auto-update.service --unit=glance.service --unit=--host=evil.service --lines=500 --output=json --no-pager" {
		t.Errorf("ran %q", call)
	}

	out, _ := json.Marshal(journal)
	want := `{"contract":1,"entries":[` +
		`{"time":"2026-11-18T11:06:40+00:00","unit":"glance","priority":3,"message":"Error: config: glance.yml: no such file or directory"},` +
		`{"time":"2026-11-18T11:06:41+00:00","unit":"demo-app","priority":6,"message":"Started demo-app.service."},` +
		`{"time":"2026-11-18T12:06:40+00:00","unit":"podman-auto-update","priority":6,"message":"Started"}]}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestListPassesTheFilters(t *testing.T) {
	var call string
	client, reader := fake(&call)

	cases := map[string]Options{
		"--lines=10 --boot --priority=3": {Lines: 10, Since: "boot", Priority: 3},
		"--lines=10 --since=-1h":         {Lines: 10, Since: "-1h", Priority: -1},
		"--lines=10 --since=2026-11-18":  {Lines: 10, Since: "2026-11-18", Priority: -1},
	}

	for want, options := range cases {
		if _, err := List(context.Background(), client, reader, options); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(call, want+" --output=json") {
			t.Errorf("%+v: ran %q, want %s", options, call, want)
		}
	}
}

func TestListRefusesWrongFlags(t *testing.T) {
	var call string
	client, reader := fake(&call)

	cases := map[string]Options{
		"--lines takes a number from 1 to 99999.": {Lines: 0, Priority: -1},
		"--priority takes a number from 0 to 7.":  {Lines: 1, Priority: 8},
		"--since takes a time journalctl reads":   {Lines: 1, Since: "--all", Priority: -1},
	}

	for want, options := range cases {
		if _, err := List(context.Background(), client, reader, options); err == nil || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%+v: got %v, want %s", options, err, want)
		}
	}

	if call != "" {
		t.Errorf("ran %q", call)
	}
}

func TestReadAsRoot(t *testing.T) {
	var call string
	_, reader := fake(&call)
	reader.User = false

	if _, err := reader.Read(context.Background(), "--unit=x.service"); err != nil || call != "journalctl --unit=x.service --output=json --no-pager" {
		t.Errorf("ran %q, got %v", call, err)
	}
}

func TestAnEmptyJournalIsAnEmptyList(t *testing.T) {
	entries, err := parse(nil)
	out, _ := json.Marshal(entries)

	if err != nil || string(out) != "[]" {
		t.Errorf("got %s and %v", out, err)
	}
}

func TestLastError(t *testing.T) {
	var call string
	reader := Reader{User: true, Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		call = strings.Join(append([]string{name}, args...), " ")

		return []byte("Error: config: glance.yml: no such file or directory\n"), nil
	}}

	if got := reader.LastError(context.Background(), "glance.service"); got != "Error: config: glance.yml: no such file or directory" {
		t.Errorf("got %q", got)
	}

	if call != "journalctl --user --unit=glance.service --priority=err --lines=1 --output=cat --no-pager" {
		t.Errorf("ran %q", call)
	}
}

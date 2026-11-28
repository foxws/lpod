package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsTheVersion(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-v"}, {"version"}, {"version", "--short"}} {
		var stdout, stderr bytes.Buffer

		code, ported := run(context.Background(), args, &stdout, &stderr)

		if !ported || code != 0 {
			t.Errorf("%q: got ported %v and code %d (%s), want the version", args, ported, code, stderr.String())
		}

		if stdout.String() != "lpod dev\n" {
			t.Errorf("%q: got %q, want lpod dev", args, stdout.String())
		}
	}
}

func TestRunHandsOffWhatIsntPorted(t *testing.T) {
	for _, args := range [][]string{{}, {"apps", "--json"}, {"demo", "version"}, {"help"}, {"completion"}, {"--help"}} {
		var stdout, stderr bytes.Buffer

		if _, ported := run(context.Background(), args, &stdout, &stderr); ported {
			t.Errorf("%q: ported, want it handed to the bash script", args)
		}

		if stdout.Len() != 0 {
			t.Errorf("%q: printed %q, want nothing", args, stdout.String())
		}
	}
}

func TestIsPorted(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"version", "--json"}, true},
		{[]string{"host"}, true},
		{[]string{"host", "--json"}, true},
		{[]string{"--json", "version"}, false},
		{[]string{"hosts"}, false},
		{[]string{"demo", "host"}, false},
		{[]string{"images", "--json"}, true},
		{[]string{"images"}, false},
		{[]string{"images", "status", "--json"}, false},
		{[]string{"images", "prune", "--json"}, false},
		{[]string{"containers", "--json"}, true},
		{[]string{"containers", "prune", "--json"}, false},
		{[]string{"pods", "--json"}, true},
		{[]string{"pods", "status", "--json"}, false},
		{[]string{"volumes", "--json"}, true},
		{[]string{"volumes", "sizes", "--json"}, true},
		{[]string{"volumes", "sizes"}, false},
		{[]string{"volumes", "prune", "--json"}, false},
		{[]string{"networks", "--json"}, true},
		{[]string{"networks", "prune", "--json"}, false},
		{[]string{"auto-update", "--dry-run", "--json"}, true},
		{[]string{"auto-update", "--json"}, true},
		{[]string{"auto-update", "--dry-run"}, false},
	}

	for _, c := range cases {
		if got := isPorted(newRoot(&bytes.Buffer{}), c.args); got != c.want {
			t.Errorf("isPorted(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestRunRefusesArgumentsHostDoesntTake(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, ported := run(context.Background(), []string{"host", "extra"}, &stdout, &stderr)

	if !ported || code != 1 || !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("got ported %v, code %d and %q, want an error", ported, code, stderr.String())
	}
}

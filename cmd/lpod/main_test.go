package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunPrintsTheVersion(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-v"}, {"version"}, {"version", "--short"}} {
		var stdout, stderr bytes.Buffer

		code, ported := run(context.Background(), args, &stdout, &stderr)

		if !ported || code != 0 {
			t.Errorf("%q: got ported %v and code %d, want the version", args, ported, code)
		}

		if stdout.String() != "lpod dev\n" {
			t.Errorf("%q: got %q, want lpod dev", args, stdout.String())
		}
	}
}

func TestRunHandsOffWhatIsntPorted(t *testing.T) {
	for _, args := range [][]string{{}, {"apps", "--json"}, {"stry", "version"}} {
		var stdout, stderr bytes.Buffer

		if _, ported := run(context.Background(), args, &stdout, &stderr); ported {
			t.Errorf("%q: ported, want it handed to the bash script", args)
		}

		if stdout.Len() != 0 {
			t.Errorf("%q: printed %q, want nothing", args, stdout.String())
		}
	}
}

func TestIsVersionJSON(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"version", "--json"}, true},
		{[]string{"version", "--json", "extra"}, true},
		{[]string{"version"}, false},
		{[]string{"--json", "version"}, false},
	}

	for _, c := range cases {
		if got := isVersionJSON(c.args); got != c.want {
			t.Errorf("isVersionJSON(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

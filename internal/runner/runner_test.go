package runner

import (
	"context"
	"strings"
	"testing"
)

func TestExecReturnsStdout(t *testing.T) {
	out, err := Exec(context.Background(), "sh", "-c", "echo out; echo err >&2")
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != "out\n" {
		t.Errorf("got %q, want out", out)
	}
}

func TestExecSaysWhyACommandFailed(t *testing.T) {
	_, err := Exec(context.Background(), "sh", "-c", "echo first >&2; echo 'Error: no such unit' >&2; exit 3")

	if err == nil || err.Error() != "sh -c: Error: no such unit" {
		t.Errorf("got %v, want the last line of stderr", err)
	}
}

func TestExecGivesTheExitCodeWithoutStderr(t *testing.T) {
	_, err := Exec(context.Background(), "sh", "-c", "exit 3")

	if err == nil || err.Error() != "sh -c exited with 3" {
		t.Errorf("got %v, want the exit code", err)
	}
}

func TestExecSaysWhenACommandIsMissing(t *testing.T) {
	_, err := Exec(context.Background(), "lpod-no-such-command")

	if err == nil || !strings.Contains(err.Error(), "lpod-no-such-command") {
		t.Errorf("got %v, want the missing command named", err)
	}
}

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/systemd"
)

// machine is blue's rootless host, whose timer runs at 03:00 and has never
// run, and records the commands it runs.
func machine(calls *[]string, rollbackOff bool) Machine {
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		call := strings.Join(append([]string{name}, args...), " ")
		*calls = append(*calls, call)

		switch call {
		case "systemctl --user show --property=Id,ActiveState,NextElapseUSecRealtime,LastTriggerUSec,TimersCalendar --timestamp=unix -- podman-auto-update.timer":
			return []byte("Id=podman-auto-update.timer\nActiveState=active\nNextElapseUSecRealtime=@1795003600\nLastTriggerUSec=n/a\nTimersCalendar={ OnCalendar=*-*-* 03:00:00 ; next_elapse=@1795003600 }\n"), nil
		case "loginctl show-user --property=Linger --value -- blue":
			return []byte("yes\n"), nil
		}

		return nil, errors.New("unexpected call: " + call)
	}

	return Machine{
		Systemd:  systemd.New(false, run),
		Getenv:   func(name string) string { return map[string]string{"HOME": "/home/blue"}[name] },
		Username: func() (string, error) { return "blue", nil },
		Exists: func(path string) bool {
			return rollbackOff && path == "/home/blue/.config/systemd/user/podman-auto-update.service.d/lpod-rollback.conf"
		},
	}
}

func TestShow(t *testing.T) {
	var calls []string

	out, _ := json.Marshal(Show(context.Background(), machine(&calls, false)))
	want := `{"contract":1,"settings":{"auto_update":{"enabled":true,"schedule":"*-*-* 03:00:00","rollback":true,"next_run":"2026-11-18T12:06:40+00:00","last_run":null},"linger":true,"quadlet_dir":"~/.config/containers/systemd"}}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}

	if len(calls) != 2 {
		t.Errorf("ran %q, want systemctl and loginctl once", calls)
	}
}

func TestShowSaysWhenRollbackIsOff(t *testing.T) {
	var calls []string

	if Show(context.Background(), machine(&calls, true)).Settings.AutoUpdate.Rollback {
		t.Error("got rollback on, with lpod's drop-in that turns it off")
	}
}

func TestShowWithoutTheTimer(t *testing.T) {
	fail := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("no systemd") }
	m := Machine{
		Systemd:  systemd.New(true, fail),
		Rootful:  true,
		Getenv:   func(string) string { return "" },
		Username: func() (string, error) { return "root", nil },
		Exists:   func(string) bool { return false },
	}

	out, _ := json.Marshal(Show(context.Background(), m))
	want := `{"contract":1,"settings":{"auto_update":{"enabled":false,"schedule":"daily","rollback":true,"next_run":null,"last_run":null},"linger":null,"quadlet_dir":"/etc/containers/systemd"}}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

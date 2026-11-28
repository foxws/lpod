package systemd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// recorder returns a Run that records its call and prints out.
func recorder(call *string, out string) func(context.Context, string, ...string) ([]byte, error) {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		*call = strings.Join(append([]string{name}, args...), " ")

		return []byte(out), nil
	}
}

func TestNewPicksTheScope(t *testing.T) {
	if got := New(false, nil).Systemctl; strings.Join(got, " ") != "systemctl --user" {
		t.Errorf("got %q for a user, want systemctl --user", got)
	}

	if got := New(true, nil).Systemctl; strings.Join(got, " ") != "systemctl" {
		t.Errorf("got %q for root, want systemctl", got)
	}
}

func TestShow(t *testing.T) {
	var call string
	client := New(false, recorder(&call, "Id=a.service\nActiveState=active\n\nId=b.timer\nActiveState=inactive\nNextElapseUSecRealtime=@1795003600\n"))

	units, err := client.Show(context.Background(), []string{"a.service", "b.timer"}, "ActiveState", "NextElapseUSecRealtime")
	if err != nil {
		t.Fatal(err)
	}

	if call != "systemctl --user show --property=Id,ActiveState,NextElapseUSecRealtime --timestamp=unix -- a.service b.timer" {
		t.Errorf("ran %q", call)
	}

	if len(units) != 2 || units[0]["ActiveState"] != "active" || ByID(units, "b.timer")["NextElapseUSecRealtime"] != "@1795003600" {
		t.Errorf("got %v", units)
	}

	if ByID(units, "c.service") != nil {
		t.Error("found a unit systemctl didn't print")
	}
}

func TestShowPassesUnitsAfterDashes(t *testing.T) {
	var call string
	client := New(false, recorder(&call, ""))

	if _, err := client.Show(context.Background(), []string{"--host=evil.service"}); err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(call, " -- --host=evil.service") {
		t.Errorf("ran %q, want the unit after --", call)
	}
}

func TestShowRunsNothingForNoUnits(t *testing.T) {
	client := New(false, func(context.Context, string, ...string) ([]byte, error) {
		t.Error("ran systemctl")

		return nil, nil
	})

	if units, err := client.Show(context.Background(), nil); units != nil || err != nil {
		t.Errorf("got %v and %v", units, err)
	}
}

func TestShowKeepsAnEqualsSignInAValue(t *testing.T) {
	units := parseShow("Id=a.timer\nTimersCalendar={ OnCalendar=*-*-* 03:00:00 ; next_elapse=@1 }\n")

	if got := units[0]["TimersCalendar"]; got != "{ OnCalendar=*-*-* 03:00:00 ; next_elapse=@1 }" {
		t.Errorf("got %q", got)
	}
}

func TestTimestamp(t *testing.T) {
	if got := Timestamp("@1795003600"); !got.Equal(time.Unix(1795003600, 0)) || got.Location() != time.UTC {
		t.Errorf("got %v", got)
	}

	for _, none := range []string{"", "n/a", "0", "@0", "soon"} {
		if got := Timestamp(none); !got.IsZero() {
			t.Errorf("Timestamp(%q) = %v, want none", none, got)
		}
	}
}

func TestLinger(t *testing.T) {
	var call string

	for out, want := range map[string]bool{"yes\n": true, "no\n": false} {
		got, err := New(false, recorder(&call, out)).Linger(context.Background(), "blue")
		if err != nil || got != want {
			t.Errorf("%q: got %v and %v, want %v", out, got, err, want)
		}
	}

	if call != "loginctl show-user --property=Linger --value -- blue" {
		t.Errorf("ran %q", call)
	}

	if _, err := New(false, recorder(&call, "")).Linger(context.Background(), "blue"); !errors.Is(err, ErrUnknown) {
		t.Errorf("got %v, want ErrUnknown", err)
	}
}

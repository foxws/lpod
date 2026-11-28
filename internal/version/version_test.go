package version

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/contract"
)

// fakeHost answers like a rootless host with Podman 6.2.0 and systemd 258,
// and records the commands it was asked to run.
func fakeHost(calls *[]string) Host {
	return Host{
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			call := strings.Join(append([]string{name}, args...), " ")
			*calls = append(*calls, call)

			switch call {
			case "podman version --format {{.Client.Version}}":
				return "6.2.0\n", nil
			case "systemctl --version":
				return "systemd 258 (258.1-1.fc45)\n+PAM +AUDIT\n", nil
			}

			return "", errors.New("unexpected call: " + call)
		},
		Getenv:   func(string) string { return "" },
		Geteuid:  func() int { return 1000 },
		Username: func() (string, error) { return "blue", nil },
		Hostname: func() (string, error) { return "podman-01", nil },
	}
}

func TestCollectReadsTheVersionsOfTheHost(t *testing.T) {
	var calls []string

	got := Collect(context.Background(), "v4.3.0", fakeHost(&calls))
	want := contract.Version{
		Contract: 1,
		Lpod:     "4.3.0",
		Podman:   "6.2.0",
		Systemd:  "258",
		Scope:    "rootless",
		User:     "blue",
		Hostname: "podman-01",
	}

	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if len(calls) != 2 {
		t.Errorf("got calls %q, want podman and systemctl once each", calls)
	}
}

func TestCollectRunsThePodmanTheEnvironmentNames(t *testing.T) {
	var calls []string
	host := fakeHost(&calls)
	host.Getenv = func(name string) string {
		if name == "LPOD_PODMAN_BINARY" {
			return "/opt/podman"
		}

		return ""
	}

	Collect(context.Background(), "dev", host)

	if calls[0] != "/opt/podman version --format {{.Client.Version}}" {
		t.Errorf("got %q, want /opt/podman to run", calls[0])
	}
}

func TestCollectSaysRootfulForRoot(t *testing.T) {
	var calls []string
	host := fakeHost(&calls)
	host.Geteuid = func() int { return 0 }

	if got := Collect(context.Background(), "dev", host).Scope; got != "rootful" {
		t.Errorf("got %s, want rootful", got)
	}
}

func TestCollectLeavesOutWhatItCantRead(t *testing.T) {
	failure := errors.New("not there")
	host := Host{
		Run:      func(context.Context, string, ...string) (string, error) { return "", failure },
		Getenv:   func(string) string { return "" },
		Geteuid:  func() int { return 1000 },
		Username: func() (string, error) { return "", failure },
		Hostname: func() (string, error) { return "", failure },
	}

	got := Collect(context.Background(), "dev", host)

	if got.Podman != "" || got.Systemd != "" || got.User != "" || got.Hostname != "" {
		t.Errorf("got %+v, want empty versions and names", got)
	}

	if got.Lpod != "dev" || got.Contract != 1 {
		t.Errorf("got %+v, want lpod dev and contract 1", got)
	}
}

func TestSystemdVersion(t *testing.T) {
	cases := map[string]string{
		"systemd 258 (258.1-1.fc45)\n+PAM": "258",
		"systemd 255\n":                    "255",
		"":                                 "",
		"systemd\n":                        "",
	}

	for out, want := range cases {
		if got := systemdVersion(out); got != want {
			t.Errorf("systemdVersion(%q) = %q, want %q", out, got, want)
		}
	}
}

func TestText(t *testing.T) {
	if got := Text("4.3.0"); got != "lpod 4.3.0" {
		t.Errorf("got %s, want lpod 4.3.0", got)
	}
}

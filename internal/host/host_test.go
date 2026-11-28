package host

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/systemd"
)

// answers are what the fake podman, systemctl and loginctl print, by call.
var answers = map[string]string{
	"podman info --format json":     `{"host": {"networkBackend": "netavark"}, "store": {"graphRoot": "/home/blue/.local/share/containers/storage", "graphDriverName": "overlay"}}`,
	"podman ps --all --format json": `[{"State": "running"}, {"State": "exited"}, {"State": "running"}]`,
	"podman quadlet list --format json": `[
		{"Name": "demo-app.container", "UnitName": "demo-app.service", "App": "demo"},
		{"Name": "demo-data.volume", "UnitName": "demo-data-volume.service", "App": "demo"},
		{"Name": "glance.container", "UnitName": "glance.service"}
	]`,
	"systemctl --user show --property=Id,ActiveState,NextElapseUSecRealtime --timestamp=unix -- podman-auto-update.timer demo-app.service glance.service": "Id=podman-auto-update.timer\nActiveState=active\nNextElapseUSecRealtime=@1795003600\n\nId=demo-app.service\nActiveState=active\n\nId=glance.service\nActiveState=failed\n",
	"loginctl show-user --property=Linger --value -- blue": "yes\n",
}

// machine is a rootless host with blue's home in /home/blue, which records
// the commands it runs in calls.
func machine(calls *[]string) Machine {
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		call := strings.Join(append([]string{name}, args...), " ")
		*calls = append(*calls, call)

		if out, ok := answers[call]; ok {
			return []byte(out), nil
		}

		return nil, errors.New("unexpected call: " + call)
	}

	return Machine{
		Podman:  podman.Client{Binary: "podman", Run: run},
		Systemd: systemd.New(false, run),
		Root: fstest.MapFS{
			"proc/stat":                 {Data: []byte("cpu  100 0 100 700 100 0 0 0 0 0\nbtime 1792743120\n")},
			"proc/loadavg":              {Data: []byte("1.42 0.98 0.87 2/1234 5678\n")},
			"proc/meminfo":              {Data: []byte("MemTotal:       16777216 kB\nMemFree:  100 kB\nMemAvailable:   10066330 kB\nSwapTotal:       8388608 kB\nSwapFree:        8388608 kB\n")},
			"proc/sys/kernel/osrelease": {Data: []byte("7.3.0-100.fc45.x86_64\n")},
			"usr/lib/os-release":        {Data: []byte("NAME=\"Fedora Linux\"\nPRETTY_NAME=\"Fedora Linux 45\"\n")},
		},
		Statfs: func(path string) (uint64, uint64, error) {
			if path != "/home/blue/.local/share/containers/storage" {
				return 0, 0, errors.New("no such disk")
			}

			return 44023414784, 128849018880, nil
		},
		Cores: func() int { return 8 },
		// /proc/stat doesn't change between the samples, so the CPU's use
		// is 0. TestCPUUsage covers the sum...
		Sleep: func(time.Duration) {},
		Getenv: func(name string) string {
			if name == "HOME" {
				return "/home/blue"
			}

			return ""
		},
		Username: func() (string, error) { return "blue", nil },
	}
}

func TestCollectReadsTheHost(t *testing.T) {
	var calls []string
	got := Collect(context.Background(), machine(&calls))

	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"contract":1,` +
		`"cpu":{"cores":8,"usage":0,"load":[1.42,0.98,0.87]},` +
		`"memory":{"used":6871947264,"total":17179869184,"swap_used":0,"swap_total":8589934592},` +
		`"disk":{"path":"/home/blue/.local/share/containers/storage","used":44023414784,"total":128849018880},` +
		`"containers":{"running":2,"total":3},` +
		`"auto_update":{"enabled":true,"next_run":"2026-11-18T12:06:40+00:00"},` +
		`"system":{"os":"Fedora Linux 45","kernel":"7.3.0-100.fc45.x86_64","booted_at":"2026-10-23T08:12:00+00:00","storage_driver":"overlay","network_backend":"netavark","quadlet_dir":"~/.config/containers/systemd","linger":true},` +
		`"units":{"active":1,"failed":1}}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}

	if len(calls) != 5 {
		t.Errorf("ran %q, want each command once", calls)
	}
}

func TestCollectLeavesOutWhatItCantRead(t *testing.T) {
	fail := func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("not there") }
	m := Machine{
		Podman:   podman.Client{Binary: "podman", Run: fail},
		Systemd:  systemd.New(false, fail),
		Root:     fstest.MapFS{},
		Statfs:   func(string) (uint64, uint64, error) { return 0, 0, errors.New("no disk") },
		Cores:    func() int { return 1 },
		Sleep:    func(time.Duration) {},
		Getenv:   func(string) string { return "" },
		Username: func() (string, error) { return "", errors.New("no user") },
	}

	out, err := json.Marshal(Collect(context.Background(), m))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"contract":1,` +
		`"cpu":{"cores":1,"usage":0,"load":[0,0,0]},` +
		`"memory":{"used":0,"total":0,"swap_used":0,"swap_total":0},` +
		`"disk":{"path":"","used":0,"total":0},` +
		`"containers":{"running":0,"total":0},` +
		`"auto_update":{"enabled":false,"next_run":null},` +
		`"system":{"os":null,"kernel":null,"booted_at":null,"storage_driver":null,"network_backend":null,"quadlet_dir":".config/containers/systemd","linger":null},` +
		`"units":{"active":0,"failed":0}}`

	if string(out) != want {
		t.Errorf("got  %s\nwant %s", out, want)
	}
}

func TestCollectAsRoot(t *testing.T) {
	var calls []string
	m := machine(&calls)
	m.Rootful = true
	m.Systemd = systemd.New(true, func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(append([]string{name}, args...), " "))

		return nil, nil
	})

	got := Collect(context.Background(), m)

	if got.System.QuadletDir != "/etc/containers/systemd" || got.System.Linger != nil {
		t.Errorf("got %+v, want the system's Quadlets and no linger", got.System)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "loginctl") || strings.HasPrefix(call, "systemctl --user") {
			t.Errorf("ran %q as root", call)
		}
	}
}

func TestCPUUsage(t *testing.T) {
	// The second sample has 100 more busy and 100 more idle jiffies than the
	// first, so the CPU was half busy...
	root := fstest.MapFS{"proc/stat": {Data: []byte("cpu  100 0 100 700 100 0 0 0 0 0\n")}}
	sleep := func(d time.Duration) {
		if d != sample {
			t.Errorf("slept %v, want %v", d, sample)
		}

		root["proc/stat"] = &fstest.MapFile{Data: []byte("cpu  150 0 150 750 150 0 0 0 0 0\n")}
	}

	if got := cpuUsage(root, sleep); got != 50 {
		t.Errorf("got %v, want 50", got)
	}
}

func TestCPUUsageWithoutProcStat(t *testing.T) {
	if got := cpuUsage(fstest.MapFS{}, func(time.Duration) {}); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestHomePathWritesHomeAsTilde(t *testing.T) {
	cases := map[string]string{
		"/home/blue/.config":  "~/.config",
		"/home/blue":          "~",
		"/home/blue2/.config": "/home/blue2/.config",
		"/etc/containers":     "/etc/containers",
	}

	for path, want := range cases {
		if got := HomePath(path, "/home/blue"); got != want {
			t.Errorf("HomePath(%s) = %s, want %s", path, got, want)
		}
	}

	if got := HomePath("/etc", "/"); got != "/etc" {
		t.Errorf("got %s with / as home, want /etc", got)
	}
}

func TestOSNamePrefersEtc(t *testing.T) {
	root := fstest.MapFS{
		"etc/os-release":     {Data: []byte("PRETTY_NAME='Debian GNU/Linux 13 (trixie)'\n")},
		"usr/lib/os-release": {Data: []byte("PRETTY_NAME=\"Other\"\n")},
	}

	if got := osName(root); got != "Debian GNU/Linux 13 (trixie)" {
		t.Errorf("got %q", got)
	}
}

func TestText(t *testing.T) {
	h := contract.Host{
		CPU:        contract.HostCPU{Cores: 8, Usage: 18.2},
		Memory:     contract.HostMemory{Used: 6871947674, Total: 17179869184},
		Disk:       contract.HostDisk{Path: "/srv/storage", Used: 44023414784, Total: 128849018880},
		Containers: contract.HostContainers{Running: 12, Total: 14},
		AutoUpdate: contract.HostAutoUpdate{Enabled: true, NextRun: contract.Time{Time: time.Unix(1795003600, 0)}},
	}

	want := "CPU: 18.2% of 8 cores\nMemory: 6.4 of 16 GiB\nDisk: 41 of 120 GiB (/srv/storage)\nContainers: 12 of 14 running\nAuto-update: next run 2026-11-18T12:06:40+00:00\n"
	if got := Text(h); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	h.AutoUpdate.Enabled = false
	if got := Text(h); !strings.HasSuffix(got, "Auto-update: off\n") {
		t.Errorf("got %q, want auto-update off", got)
	}
}

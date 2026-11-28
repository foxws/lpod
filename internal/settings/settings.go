// Package settings implements "lpod settings --json": when podman
// auto-update runs, whether a failed update rolls back, lingering and the
// Quadlet directory.
package settings

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/host"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/systemd"
)

// timeout is how long Show waits for systemctl and loginctl.
const timeout = 10 * time.Second

// Machine is what Show reads.
type Machine struct {
	Systemd  systemd.Client
	Rootful  bool
	Getenv   func(string) string
	Username func() (string, error)
	// Exists says whether a file is there.
	Exists func(path string) bool
}

// System returns the Machine lpod runs on.
func System(systemdClient systemd.Client, rootful bool, username func() (string, error)) Machine {
	return Machine{
		Systemd:  systemdClient,
		Rootful:  rootful,
		Getenv:   os.Getenv,
		Username: username,
		Exists: func(path string) bool {
			_, err := os.Stat(path)

			return err == nil
		},
	}
}

// RollbackDropin is the drop-in "lpod settings set --rollback off" writes,
// which runs podman auto-update with --rollback=false.
func RollbackDropin(rootful bool, getenv func(string) string) string {
	return filepath.Join(systemd.UnitDir(rootful, getenv), "podman-auto-update.service.d", "lpod-rollback.conf")
}

// calendarPattern finds the first OnCalendar= in the timer's
// TimersCalendar, such as "{ OnCalendar=*-*-* 03:00:00 ; next_elapse=… }".
var calendarPattern = regexp.MustCompile(`OnCalendar=(.+?)\s*;`)

// Show reads the settings for "lpod settings --json". Like a dashboard, it
// shows what it can read.
func Show(ctx context.Context, m Machine) contract.HostSettings {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	settings := contract.Settings{
		AutoUpdate: contract.AutoUpdateSettings{
			Schedule: "daily",
			Rollback: !m.Exists(RollbackDropin(m.Rootful, m.Getenv)),
		},
		QuadletDir: host.HomePath(quadlet.Dir(m.Rootful, m.Getenv), m.Getenv("HOME")),
	}

	units, _ := m.Systemd.Show(ctx, []string{"podman-auto-update.timer"}, "ActiveState", "NextElapseUSecRealtime", "LastTriggerUSec", "TimersCalendar")
	if timer := systemd.ByID(units, "podman-auto-update.timer"); timer != nil {
		settings.AutoUpdate.Enabled = timer["ActiveState"] == "active"
		settings.AutoUpdate.NextRun = contract.Time{Time: systemd.Timestamp(timer["NextElapseUSecRealtime"])}
		settings.AutoUpdate.LastRun = contract.Time{Time: systemd.Timestamp(timer["LastTriggerUSec"])}

		if match := calendarPattern.FindStringSubmatch(timer["TimersCalendar"]); match != nil {
			settings.AutoUpdate.Schedule = match[1]
		}
	}

	if !m.Rootful {
		if name, err := m.Username(); err == nil {
			if linger, err := m.Systemd.Linger(ctx, name); err == nil {
				settings.Linger = &linger
			}
		}
	}

	return contract.HostSettings{Contract: contract.Current, Settings: settings}
}

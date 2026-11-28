// Package systemd reads units through systemctl, in the scope the Quadlets
// run in: the user's own manager, or the system's as root.
//
// It runs systemctl rather than talking to systemd over D-Bus, so the test
// stand-in for systemctl answers it too (see #59).
package systemd

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/runner"
)

// Client runs systemctl and loginctl.
type Client struct {
	// Systemctl is the command, with "--user" for the user's manager.
	Systemctl []string
	Run       runner.Func
}

// New returns a Client for the system's manager as root, which runs the
// rootful Quadlets, and else for the user's own.
func New(rootful bool, run runner.Func) Client {
	if rootful {
		return Client{Systemctl: []string{"systemctl"}, Run: run}
	}

	return Client{Systemctl: []string{"systemctl", "--user"}, Run: run}
}

// Unit is the properties "systemctl show" printed for one unit, such as
// unit["ActiveState"].
type Unit map[string]string

// Show runs "systemctl show" for units, and returns their properties in the
// order systemctl printed them. It always asks for Id, so a Unit can be
// told apart from the others. The unit names come after "--", so one that
// starts with "-" isn't read as an option.
func (c Client) Show(ctx context.Context, units []string, properties ...string) ([]Unit, error) {
	if len(units) == 0 {
		return nil, nil
	}

	args := append([]string{}, c.Systemctl[1:]...)
	args = append(args, "show", "--property="+strings.Join(append([]string{"Id"}, properties...), ","), "--timestamp=unix", "--")
	args = append(args, units...)

	out, err := c.Run(ctx, c.Systemctl[0], args...)
	if err != nil {
		return nil, err
	}

	return parseShow(string(out)), nil
}

// ByID returns the unit of units with the Id id, or nil.
func ByID(units []Unit, id string) Unit {
	for _, unit := range units {
		if unit["Id"] == id {
			return unit
		}
	}

	return nil
}

// parseShow reads "systemctl show" output: a "Key=value" line per property,
// and a blank line between units.
func parseShow(out string) []Unit {
	var units []Unit

	for block := range strings.SplitSeq(strings.TrimSpace(out), "\n\n") {
		unit := Unit{}

		for line := range strings.SplitSeq(block, "\n") {
			if key, value, ok := strings.Cut(line, "="); ok {
				unit[key] = value
			}
		}

		if unit["Id"] != "" {
			units = append(units, unit)
		}
	}

	return units
}

// Timestamp reads a time of "systemctl show --timestamp=unix", such as
// "@1795003600". It's the zero time when there's none, which systemctl
// prints as "", "n/a" or "0".
func Timestamp(value string) time.Time {
	seconds, err := strconv.ParseInt(strings.TrimPrefix(value, "@"), 10, 64)
	if err != nil || seconds <= 0 {
		return time.Time{}
	}

	return time.Unix(seconds, 0).UTC()
}

// ErrUnknown is what Linger returns when loginctl doesn't say.
var ErrUnknown = errors.New("loginctl didn't say")

// Linger says whether systemd keeps user's services running after they log
// out, which rootless Quadlets need to run without a login.
func (c Client) Linger(ctx context.Context, user string) (bool, error) {
	out, err := c.Run(ctx, "loginctl", "show-user", "--property=Linger", "--value", "--", user)
	if err != nil {
		return false, err
	}

	switch strings.TrimSpace(string(out)) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}

	return false, ErrUnknown
}

// UnitDir is where lpod writes units and drop-ins of its own, such as
// podman-auto-update.timer.d/lpod-schedule.conf: the system's directory as
// root, and else the user's.
func UnitDir(rootful bool, getenv func(string) string) string {
	if rootful {
		return "/etc/systemd/system"
	}

	// The XDG spec says to ignore a relative XDG_CONFIG_HOME...
	config := getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(config) {
		config = filepath.Join(getenv("HOME"), ".config")
	}

	return filepath.Join(config, "systemd", "user")
}

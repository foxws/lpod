// Package journal reads the journal with journalctl, and implements
// "lpod journal --json": the journal of every unit the installed Quadlets
// generate, and of podman-auto-update.service.
package journal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/runner"
)

// timeout is how long Read waits for journalctl.
const timeout = 30 * time.Second

// Reader runs journalctl in the scope the Quadlets run in.
type Reader struct {
	// User reads the user's journal, as a rootless user does.
	User bool
	Run  runner.Func
}

// Read runs journalctl with args and returns its entries, oldest first.
// Pass units as "--unit=NAME", so a name that starts with "-" isn't read as
// an option.
func (r Reader) Read(ctx context.Context, args ...string) ([]contract.JournalEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var command []string
	if r.User {
		command = append(command, "--user")
	}

	command = append(command, args...)
	command = append(command, "--output=json", "--no-pager")

	out, err := r.Run(ctx, "journalctl", command...)
	if err != nil {
		return nil, err
	}

	return parse(out)
}

// record is a line of "journalctl --output=json". Every field is a
// string, except a MESSAGE that isn't valid UTF-8, which is an array of
// bytes.
type record struct {
	Realtime        string          `json:"__REALTIME_TIMESTAMP"`
	SystemdUserUnit string          `json:"_SYSTEMD_USER_UNIT"`
	SystemdUnit     string          `json:"_SYSTEMD_UNIT"`
	UserUnit        string          `json:"USER_UNIT"`
	Unit            string          `json:"UNIT"`
	Priority        string          `json:"PRIORITY"`
	Message         json.RawMessage `json:"MESSAGE"`
}

// parse reads journalctl's JSON, one object per line.
func parse(out []byte) ([]contract.JournalEntry, error) {
	entries := []contract.JournalEntry{}
	lines := bufio.NewScanner(bytes.NewReader(out))
	lines.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	for lines.Scan() {
		if len(bytes.TrimSpace(lines.Bytes())) == 0 {
			continue
		}

		var r record
		if err := json.Unmarshal(lines.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("can't read journalctl: %w", err)
		}

		entries = append(entries, r.entry())
	}

	return entries, lines.Err()
}

// entry is the record as the contract has it.
func (r record) entry() contract.JournalEntry {
	entry := contract.JournalEntry{Unit: r.unit(), Priority: 6, Message: message(r.Message)}

	if micro, err := strconv.ParseInt(r.Realtime, 10, 64); err == nil {
		entry.Time = contract.Time{Time: time.Unix(micro/1_000_000, 0)}
	}

	if priority, err := strconv.Atoi(r.Priority); err == nil {
		entry.Priority = priority
	}

	return entry
}

// unit is the unit that logged the record, without ".service". systemd's
// own lines about a unit come from init.scope or the user's manager
// (user@UID.service), and name the unit in USER_UNIT or UNIT.
func (r record) unit() string {
	unit := r.SystemdUserUnit
	if unit == "" {
		unit = r.SystemdUnit
	}

	if unit == "" || unit == "init.scope" || strings.HasPrefix(unit, "user@") {
		for _, about := range []string{r.UserUnit, r.Unit} {
			if about != "" {
				unit = about

				break
			}
		}
	}

	return strings.TrimSuffix(unit, ".service")
}

// message is MESSAGE as text, without the newline at its end.
func message(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSuffix(text, "\n")
	}

	var data []byte
	var numbers []int

	if err := json.Unmarshal(raw, &numbers); err == nil {
		for _, n := range numbers {
			data = append(data, byte(n))
		}
	}

	return strings.TrimSuffix(string(data), "\n")
}

// Options are the flags of "lpod journal".
type Options struct {
	// Lines is how many entries, the last ones, from 1 to 99999.
	Lines int
	// Since is a time journalctl reads, such as "2026-11-18" or "-1h", or
	// "boot" for this boot. "" is the whole journal.
	Since string
	// Priority keeps the entries of this priority and up, from 0 to 7, or
	// -1 for all.
	Priority int
}

// errLines is the message for a --lines that isn't a number from 1 to
// 99999, as the bash script prints it.
var errLines = errors.New("--lines takes a number from 1 to 99999.") //nolint:staticcheck // printed as it is

// ValidLines checks a --lines value.
func ValidLines(lines int) error {
	if lines < 1 || lines > 99999 {
		return errLines
	}

	return nil
}

// sincePattern is a --since journalctl reads that can't be an option: it
// may start with "-" or "+" only when a number follows, as in "-1h".
var sincePattern = regexp.MustCompile(`^([^-+]|[-+][0-9])`)

// List reads the journal for "lpod journal --json".
func List(ctx context.Context, client podman.Client, reader Reader, options Options) (contract.Journal, error) {
	if err := ValidLines(options.Lines); err != nil {
		return contract.Journal{}, err
	}

	if options.Priority < -1 || options.Priority > 7 {
		return contract.Journal{}, errors.New("--priority takes a number from 0 to 7.") //nolint:staticcheck // printed as it is
	}

	quadlets, err := client.Quadlets(ctx)
	if err != nil {
		return contract.Journal{}, err
	}

	args := []string{"--unit=podman-auto-update.service"}

	for _, quadlet := range quadlets {
		if quadlet.UnitName != "" {
			args = append(args, "--unit="+quadlet.UnitName)
		}
	}

	args = append(args, "--lines="+strconv.Itoa(options.Lines))

	switch {
	case options.Since == "boot":
		args = append(args, "--boot")
	case options.Since != "":
		if !sincePattern.MatchString(options.Since) {
			return contract.Journal{}, fmt.Errorf("--since takes a time journalctl reads, such as 2026-11-18, -1h or boot, not %s.", options.Since) //nolint:staticcheck // printed as it is
		}

		args = append(args, "--since="+options.Since)
	}

	if options.Priority >= 0 {
		args = append(args, "--priority="+strconv.Itoa(options.Priority))
	}

	entries, err := reader.Read(ctx, args...)
	if err != nil {
		return contract.Journal{}, err
	}

	return contract.Journal{Contract: contract.Current, Entries: entries}, nil
}

// Package contract has the JSON payloads that lpod's --json commands print,
// which tools such as Pods parse. tests/fixtures/contract-1 has an example of
// each, and docs/commands.md describes them.
package contract

import (
	"encoding/json"
	"io"
	"time"
)

// Current is the version of the contract the payloads follow. Tools refuse a
// contract they don't know, so a breaking change to a payload bumps it, as
// LPOD_CONTRACT in the bash script.
const Current = 1

// Version is what "lpod version --json" prints (version.json).
type Version struct {
	Contract int    `json:"contract"`
	Lpod     string `json:"lpod"`
	Podman   string `json:"podman"`
	Systemd  string `json:"systemd"`
	Scope    string `json:"scope"`
	User     string `json:"user"`
	Hostname string `json:"hostname"`
}

// Host is what "lpod host --json" prints (host.json): how busy the host is,
// and what runs on it.
type Host struct {
	Contract   int            `json:"contract"`
	CPU        HostCPU        `json:"cpu"`
	Memory     HostMemory     `json:"memory"`
	Disk       HostDisk       `json:"disk"`
	Containers HostContainers `json:"containers"`
	AutoUpdate HostAutoUpdate `json:"auto_update"`
	System     HostSystem     `json:"system"`
	Units      HostUnits      `json:"units"`
}

// HostCPU is the number of cores, the percentage of them in use, and the
// load averages over 1, 5 and 15 minutes.
type HostCPU struct {
	Cores int        `json:"cores"`
	Usage float64    `json:"usage"`
	Load  [3]float64 `json:"load"`
}

// HostMemory is the memory and swap in use, in bytes.
type HostMemory struct {
	Used      uint64 `json:"used"`
	Total     uint64 `json:"total"`
	SwapUsed  uint64 `json:"swap_used"`
	SwapTotal uint64 `json:"swap_total"`
}

// HostDisk is the space in use, in bytes, on the disk Podman stores images
// and containers on, at Path.
type HostDisk struct {
	Path  string `json:"path"`
	Used  uint64 `json:"used"`
	Total uint64 `json:"total"`
}

// HostContainers counts the containers, and the ones that run.
type HostContainers struct {
	Running int `json:"running"`
	Total   int `json:"total"`
}

// HostAutoUpdate says whether podman-auto-update.timer is on, and when it
// runs next.
type HostAutoUpdate struct {
	Enabled bool `json:"enabled"`
	NextRun Time `json:"next_run"`
}

// HostSystem describes the host. A value lpod couldn't read is null.
type HostSystem struct {
	OS             *string `json:"os"`
	Kernel         *string `json:"kernel"`
	BootedAt       Time    `json:"booted_at"`
	StorageDriver  *string `json:"storage_driver"`
	NetworkBackend *string `json:"network_backend"`
	// QuadletDir has the home directory written as "~".
	QuadletDir string `json:"quadlet_dir"`
	// Linger is null as root, where it doesn't apply.
	Linger *bool `json:"linger"`
}

// HostUnits counts the services of the Quadlets that are active, and that
// failed.
type HostUnits struct {
	Active int `json:"active"`
	Failed int `json:"failed"`
}

// Images is what "lpod images --json" prints (images.json).
type Images struct {
	Contract int     `json:"contract"`
	Images   []Image `json:"images"`
}

// Image is an image on the host, with what uses it.
type Image struct {
	// ID is the short ID, its first 12 characters.
	ID string `json:"id"`
	// Repository and Tag are null for a dangling image, which has no name.
	Repository *string `json:"repository"`
	Tag        *string `json:"tag"`
	Size       *int64  `json:"size"`
	CreatedAt  Time    `json:"created_at"`
	// UsedBy is the applications whose containers run it.
	UsedBy []string `json:"used_by"`
	// Containers counts every container that runs it.
	Containers int  `json:"containers"`
	Dangling   bool `json:"dangling"`
	// AutoUpdate is the AutoUpdate= policy of a container that runs it.
	AutoUpdate *string `json:"auto_update"`
	// Update is the tag the registry has a newer image for, from the last
	// "lpod auto-update --dry-run".
	Update *string `json:"update"`
}

// OK is what a --json command that only changes something prints.
type OK struct {
	Contract int  `json:"contract"`
	OK       bool `json:"ok"`
}

// Done is the OK payload of the current contract.
func Done() OK {
	return OK{Contract: Current, OK: true}
}

// Time is a time in the contract, such as "2026-11-15T15:00:00+00:00",
// always in UTC and without a fraction of a second. The zero Time is null.
type Time struct {
	time.Time
}

// TimeLayout is how a Time is written.
const TimeLayout = "2006-01-02T15:04:05+00:00"

// MarshalJSON writes t, or null for the zero Time. Go calls it for every
// Time in a payload, as a cast does in Laravel.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}

	return json.Marshal(t.UTC().Format(TimeLayout))
}

// String returns s as a JSON string, or null when it's empty.
func String(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// Write prints payload as one JSON object, indented as jq prints it.
func Write(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(payload)
}

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

// Containers is what "lpod containers --json" prints (containers.json).
type Containers struct {
	Contract   int         `json:"contract"`
	Containers []Container `json:"containers"`
}

// Container is a container on the host, managed by Quadlet or not. A pod's
// infra container isn't one.
type Container struct {
	// ID is the short ID, its first 12 characters.
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	// State is such as "running", "exited" or "created".
	State string `json:"state"`
	// Health is "healthy", "unhealthy" or "starting", or null without a
	// health check.
	Health *string `json:"health"`
	// ExitCode is null unless the container exited.
	ExitCode *int `json:"exit_code"`
	// Application, Quadlet and Unit are null for a container Quadlet
	// doesn't run.
	Application *string `json:"application"`
	Quadlet     *string `json:"quadlet"`
	Unit        *string `json:"unit"`
	Pod         *string `json:"pod"`
	// Ports are the published ports, such as "0.0.0.0:8000->8080/tcp".
	Ports []string `json:"ports"`
	// CPU is a percentage of one core, and Memory is in bytes. Both are
	// null unless the container runs.
	CPU       *float64 `json:"cpu"`
	Memory    *uint64  `json:"memory"`
	StartedAt Time     `json:"started_at"`
}

// Pods is what "lpod pods --json" prints (pods.json).
type Pods struct {
	Contract int   `json:"contract"`
	Pods     []Pod `json:"pods"`
}

// Pod is a pod on the host.
type Pod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Status is such as "running", "degraded" or "exited".
	Status string `json:"status"`
	// Application, Quadlet and Unit are null for a pod Quadlet doesn't
	// run. A .kube Quadlet's pods are its own.
	Application *string `json:"application"`
	Quadlet     *string `json:"quadlet"`
	Unit        *string `json:"unit"`
	// Containers counts its containers, without the infra container, and
	// Running the ones that run.
	Containers int `json:"containers"`
	Running    int `json:"running"`
	// Ports are what its infra container publishes.
	Ports     []string `json:"ports"`
	Networks  []string `json:"networks"`
	CreatedAt Time     `json:"created_at"`
}

// Volumes is what "lpod volumes --json" prints (volumes.json).
type Volumes struct {
	Contract int      `json:"contract"`
	Volumes  []Volume `json:"volumes"`
}

// Volume is a volume, or a folder of the host that containers mount.
type Volume struct {
	// Name is the volume's, or the folder's path for a bind mount.
	Name string `json:"name"`
	// Quadlet is the .volume that creates it, or the .container that
	// names it in Volume=, or null.
	Quadlet *string `json:"quadlet"`
	Driver  string  `json:"driver"`
	// Mountpoint is where it is on the host, or the device or NFS share
	// that backs it.
	Mountpoint string `json:"mountpoint"`
	// Size is always null here: "lpod volumes sizes --json" works it out.
	Size *int64 `json:"size"`
	// Type is "named", "anonymous", "device", "nfs" or "bind".
	Type string `json:"type"`
	// Containers are the names of the containers that mount it, and
	// Applications their applications.
	Containers   []string `json:"containers"`
	Applications []string `json:"applications"`
}

// VolumeSizes is what "lpod volumes sizes --json" prints (volume-sizes.json).
type VolumeSizes struct {
	Contract int          `json:"contract"`
	Sizes    []VolumeSize `json:"sizes"`
}

// VolumeSize is the disk space a volume uses, in bytes, or null when du
// couldn't measure it.
type VolumeSize struct {
	Name string `json:"name"`
	Size *int64 `json:"size"`
}

// Networks is what "lpod networks --json" prints (networks.json).
type Networks struct {
	Contract int       `json:"contract"`
	Networks []Network `json:"networks"`
}

// Network is a Podman network.
type Network struct {
	Name string `json:"name"`
	// Quadlet is the .network that creates it, or null.
	Quadlet *string `json:"quadlet"`
	Driver  string  `json:"driver"`
	// Subnet and Gateway are its first subnet's, or null.
	Subnet   *string `json:"subnet"`
	Gateway  *string `json:"gateway"`
	DNS      bool    `json:"dns"`
	Internal bool    `json:"internal"`
	IPv6     bool    `json:"ipv6"`
	// Containers counts the containers attached to it.
	Containers int `json:"containers"`
	// Applications are the applications whose containers are attached to
	// it, or whose Quadlets name it in Network=.
	Applications []string `json:"applications"`
	// Default is Podman's own network, "podman".
	Default bool `json:"default"`
}

// Journal is what "lpod journal --json" prints (journal.json).
type Journal struct {
	Contract int            `json:"contract"`
	Entries  []JournalEntry `json:"entries"`
}

// JournalEntry is a line of the journal, oldest first.
type JournalEntry struct {
	Time Time `json:"time"`
	// Unit is the unit that logged it, without ".service".
	Unit string `json:"unit"`
	// Priority is syslog's, from 0 (emergency) to 7 (debug).
	Priority int    `json:"priority"`
	Message  string `json:"message"`
}

// Events is what "lpod events --json" prints (events.json).
type Events struct {
	Contract int     `json:"contract"`
	Events   []Event `json:"events"`
}

// Event is something that happened to a container or an image, newest
// first.
type Event struct {
	Time Time `json:"time"`
	// Type is "container" or "image", and Action such as "start", "died"
	// or "pull".
	Type   string `json:"type"`
	Action string `json:"action"`
	// Name is the container's, or the image's reference or ID.
	Name string `json:"name"`
	// Image is the container's image, and null for an image.
	Image       *string `json:"image"`
	Application *string `json:"application"`
	// ExitCode is null unless a container died.
	ExitCode *int `json:"exit_code"`
}

// HostSettings is what "lpod settings --json" prints (host-settings.json).
type HostSettings struct {
	Contract int      `json:"contract"`
	Settings Settings `json:"settings"`
}

// Settings are the host's settings "lpod settings set" changes.
type Settings struct {
	AutoUpdate AutoUpdateSettings `json:"auto_update"`
	// Linger is null as root, where it doesn't apply.
	Linger *bool `json:"linger"`
	// QuadletDir has the home directory written as "~".
	QuadletDir string `json:"quadlet_dir"`
}

// AutoUpdateSettings are podman-auto-update.timer's.
type AutoUpdateSettings struct {
	Enabled bool `json:"enabled"`
	// Schedule is the timer's OnCalendar=, such as "daily".
	Schedule string `json:"schedule"`
	// Rollback says whether a failed update goes back to the image before.
	Rollback bool `json:"rollback"`
	NextRun  Time `json:"next_run"`
	LastRun  Time `json:"last_run"`
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

// ParseTime reads a time Podman prints, such as
// "2026-11-17T10:00:00.123456789Z". Podman's zero time ("0001-01-01…") and
// one it can't read are the zero Time, which is null.
func ParseTime(s string) Time {
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Time{}
	}

	return Time{Time: parsed}
}

// Unix is seconds since the epoch as a Time, or null for 0.
func Unix(seconds int64) Time {
	if seconds <= 0 {
		return Time{}
	}

	return Time{Time: time.Unix(seconds, 0)}
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

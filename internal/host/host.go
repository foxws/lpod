// Package host implements "lpod host": how busy the host is, and what runs
// on it.
package host

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/systemd"
)

// Machine is what Collect reads the host with. System returns the real one,
// and tests fill it with fakes, as you would bind fakes in Laravel's
// service container.
type Machine struct {
	Podman  podman.Client
	Systemd systemd.Client
	// Root is the file system, for /proc and /etc/os-release.
	Root fs.FS
	// Statfs returns the bytes in use and in all on the disk of path.
	Statfs func(path string) (used, total uint64, err error)
	// Cores is the number of CPU cores lpod may use.
	Cores func() int
	// Sleep waits between the two samples of the CPU's use.
	Sleep    func(time.Duration)
	Getenv   func(string) string
	Rootful  bool
	Username func() (string, error)
}

// System returns the Machine lpod runs on, which asks Podman and systemd with
// the clients given.
func System(podmanClient podman.Client, systemdClient systemd.Client, rootful bool) Machine {
	return Machine{
		Podman:  podmanClient,
		Systemd: systemdClient,
		Root:    os.DirFS("/"),
		Statfs:  statfs,
		Cores:   runtime.NumCPU,
		Sleep:   time.Sleep,
		Getenv:  os.Getenv,
		Rootful: rootful,
		Username: func() (string, error) {
			current, err := user.Current()
			if err != nil {
				return "", err
			}

			return current.Username, nil
		},
	}
}

// timeout is how long Collect waits for podman and systemctl, so a hanging
// Podman doesn't hang the command Pods runs over SSH.
const timeout = 10 * time.Second

// sample is how long Collect watches the CPU for.
const sample = 250 * time.Millisecond

// autoUpdateTimer is the timer that runs "podman auto-update".
const autoUpdateTimer = "podman-auto-update.timer"

// Collect reads the host for "lpod host". Like a dashboard, it shows what it
// can: a value it can't read is 0, or null in the payload.
func Collect(ctx context.Context, m Machine) contract.Host {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload := contract.Host{Contract: contract.Current}

	// The CPU is sampled while podman and systemctl run, so the command
	// doesn't wait for the sample on its own. "wg.Go" starts a goroutine,
	// a function that runs at the same time as the rest, and wg.Wait waits
	// for it, much like a Laravel job you dispatch and then wait on.
	var wg sync.WaitGroup
	var usage float64
	wg.Go(func() { usage = cpuUsage(m.Root, m.Sleep) })

	payload.CPU.Cores = m.Cores()
	payload.CPU.Load = loadAverages(m.Root)
	payload.Memory = memory(m.Root)
	payload.System = system(ctx, m)

	info, err := m.Podman.Info(ctx)
	if err == nil {
		payload.System.StorageDriver = contract.String(info.Store.GraphDriverName)
		payload.System.NetworkBackend = contract.String(info.Host.NetworkBackend)
	}

	payload.Disk = disk(info.Store.GraphRoot, m)

	if containers, err := m.Podman.Containers(ctx); err == nil {
		payload.Containers.Total = len(containers)

		for _, container := range containers {
			if container.State == "running" {
				payload.Containers.Running++
			}
		}
	}

	payload.AutoUpdate, payload.Units = units(ctx, m)

	wg.Wait()
	payload.CPU.Usage = usage

	return payload
}

// units reads the auto-update timer and the services of the Quadlets, with
// one systemctl.
func units(ctx context.Context, m Machine) (contract.HostAutoUpdate, contract.HostUnits) {
	var autoUpdate contract.HostAutoUpdate
	var counts contract.HostUnits

	names := []string{autoUpdateTimer}

	if quadlets, err := m.Podman.Quadlets(ctx); err == nil {
		for _, quadlet := range quadlets {
			if quadlet.RunsAService() && quadlet.UnitName != "" {
				names = append(names, quadlet.UnitName)
			}
		}
	}

	shown, err := m.Systemd.Show(ctx, names, "ActiveState", "NextElapseUSecRealtime")
	if err != nil {
		return autoUpdate, counts
	}

	for _, unit := range shown {
		if unit["Id"] == autoUpdateTimer {
			autoUpdate.Enabled = unit["ActiveState"] == "active"
			autoUpdate.NextRun = contract.Time{Time: systemd.Timestamp(unit["NextElapseUSecRealtime"])}

			continue
		}

		switch unit["ActiveState"] {
		case "active":
			counts.Active++
		case "failed":
			counts.Failed++
		}
	}

	return autoUpdate, counts
}

// system describes the host: its OS, kernel, boot time and where Quadlets
// are installed.
func system(ctx context.Context, m Machine) contract.HostSystem {
	kernel, _ := fs.ReadFile(m.Root, "proc/sys/kernel/osrelease")

	described := contract.HostSystem{
		OS:         contract.String(osName(m.Root)),
		Kernel:     contract.String(strings.TrimSpace(string(kernel))),
		BootedAt:   contract.Time{Time: bootedAt(m.Root)},
		QuadletDir: homePath(quadlet.Dir(m.Rootful, m.Getenv), m.Getenv("HOME")),
	}

	if !m.Rootful {
		if name, err := m.Username(); err == nil {
			if linger, err := m.Systemd.Linger(ctx, name); err == nil {
				described.Linger = &linger
			}
		}
	}

	return described
}

// homePath writes path with the home directory as "~".
func homePath(path, home string) string {
	if home == "" || home == "/" {
		return path
	}

	if path == home {
		return "~"
	}

	if rest, ok := strings.CutPrefix(path, home+"/"); ok {
		return "~/" + rest
	}

	return path
}

// disk is the use of the disk Podman stores images and containers on, or of
// the home directory when Podman didn't say where that is.
func disk(graphRoot string, m Machine) contract.HostDisk {
	path := graphRoot
	if path == "" {
		path = m.Getenv("HOME")
	}

	usage := contract.HostDisk{Path: path}

	if path != "" {
		if used, total, err := m.Statfs(path); err == nil {
			usage.Used, usage.Total = used, total
		}
	}

	return usage
}

// statfs returns the bytes in use and in all on the disk of path, as df
// does.
func statfs(path string) (used, total uint64, err error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	size := uint64(stat.Frsize)

	return (stat.Blocks - stat.Bfree) * size, stat.Blocks * size, nil
}

// cpuUsage is the percentage of the CPU in use, from two samples of
// /proc/stat, rounded to one decimal.
func cpuUsage(root fs.FS, sleep func(time.Duration)) float64 {
	idle1, total1, ok := cpuTimes(root)
	if !ok {
		return 0
	}

	sleep(sample)

	idle2, total2, ok := cpuTimes(root)
	if !ok || total2 <= total1 {
		return 0
	}

	busy := 1 - float64(idle2-idle1)/float64(total2-total1)

	return math.Round(busy*1000) / 10
}

// cpuTimes reads the time the CPU spent idle, and in all, from the "cpu"
// line of /proc/stat. It leaves out guest time, which the user time already
// counts.
func cpuTimes(root fs.FS) (idle, total uint64, ok bool) {
	line := statLine(root, "cpu")
	if len(line) < 8 {
		return 0, 0, false
	}

	// user nice system idle iowait irq softirq steal...
	for i, field := range line[:8] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, false
		}

		total += value

		if i == 3 || i == 4 {
			idle += value
		}
	}

	return idle, total, true
}

// bootedAt is when the host booted, from the "btime" line of /proc/stat.
func bootedAt(root fs.FS) time.Time {
	line := statLine(root, "btime")
	if len(line) < 1 {
		return time.Time{}
	}

	seconds, err := strconv.ParseInt(line[0], 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(seconds, 0).UTC()
}

// statLine returns the fields after name on its line of /proc/stat.
func statLine(root fs.FS, name string) []string {
	stat, err := fs.ReadFile(root, "proc/stat")
	if err != nil {
		return nil
	}

	for line := range strings.SplitSeq(string(stat), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == name {
			return fields[1:]
		}
	}

	return nil
}

// loadAverages reads the load averages over 1, 5 and 15 minutes.
func loadAverages(root fs.FS) [3]float64 {
	var load [3]float64

	content, err := fs.ReadFile(root, "proc/loadavg")
	if err != nil {
		return load
	}

	for i, field := range strings.Fields(string(content)) {
		if i == len(load) {
			break
		}

		load[i], _ = strconv.ParseFloat(field, 64)
	}

	return load
}

// memory reads the memory and swap in use from /proc/meminfo. Memory in use
// is what isn't available, as free shows it.
func memory(root fs.FS) contract.HostMemory {
	content, err := fs.ReadFile(root, "proc/meminfo")
	if err != nil {
		return contract.HostMemory{}
	}

	kib := map[string]uint64{}

	for line := range strings.SplitSeq(string(content), "\n") {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		if fields := strings.Fields(value); len(fields) > 0 {
			kib[name], _ = strconv.ParseUint(fields[0], 10, 64)
		}
	}

	return contract.HostMemory{
		Used:      used(kib["MemTotal"], kib["MemAvailable"]) * 1024,
		Total:     kib["MemTotal"] * 1024,
		SwapUsed:  used(kib["SwapTotal"], kib["SwapFree"]) * 1024,
		SwapTotal: kib["SwapTotal"] * 1024,
	}
}

// used is total less free, and never below 0.
func used(total, free uint64) uint64 {
	if free > total {
		return 0
	}

	return total - free
}

// osName is the PRETTY_NAME of os-release, such as "Fedora Linux 45 (Server
// Edition)". /usr/lib/os-release is the one a distribution ships, which
// /etc/os-release usually links to.
func osName(root fs.FS) string {
	for _, path := range []string{"etc/os-release", "usr/lib/os-release"} {
		file, err := root.Open(path)
		if err != nil {
			continue
		}

		name := prettyName(bufio.NewScanner(file))
		_ = file.Close()

		return name
	}

	return ""
}

// prettyName finds PRETTY_NAME in the lines of an os-release file, without
// its quotes.
func prettyName(lines *bufio.Scanner) string {
	for lines.Scan() {
		if value, ok := strings.CutPrefix(lines.Text(), "PRETTY_NAME="); ok {
			if unquoted, err := strconv.Unquote(value); err == nil {
				return unquoted
			}

			return strings.Trim(value, `"'`)
		}
	}

	return ""
}

// Text is what "lpod host" prints for people.
func Text(h contract.Host) string {
	autoUpdate := "off"
	if h.AutoUpdate.Enabled {
		autoUpdate = "next run " + h.AutoUpdate.NextRun.Format(contract.TimeLayout)
		if h.AutoUpdate.NextRun.IsZero() {
			autoUpdate = "on"
		}
	}

	return fmt.Sprintf("CPU: %s%% of %d cores\nMemory: %s of %s GiB\nDisk: %s of %s GiB (%s)\nContainers: %d of %d running\nAuto-update: %s\n",
		strconv.FormatFloat(h.CPU.Usage, 'f', -1, 64), h.CPU.Cores,
		gib(h.Memory.Used), gib(h.Memory.Total),
		gib(h.Disk.Used), gib(h.Disk.Total), h.Disk.Path,
		h.Containers.Running, h.Containers.Total,
		autoUpdate)
}

// gib writes bytes in GiB, with one decimal.
func gib(bytes uint64) string {
	return strconv.FormatFloat(math.Floor(float64(bytes)/(1<<30)*10)/10, 'f', -1, 64)
}

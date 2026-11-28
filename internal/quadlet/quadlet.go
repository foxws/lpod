// Package quadlet reads installed Quadlet files the way Quadlet does: with
// Podman's own parser, and with their drop-ins merged in.
//
// A directive is looked up in its section, such as VolumeName= in [Volume].
// The parser joins continued lines, allows spaces around "=", and lets an
// empty value reset a list, which reading the lines by hand missed (see
// "Found in phase 0" in #59).
package quadlet

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.podman.io/podman/v6/pkg/systemd/parser"

	"github.com/foxws/lpod/internal/podman"
)

// Dir is where Quadlet looks for the units "podman quadlet install"
// installs, and their drop-ins.
func Dir(rootful bool, getenv func(string) string) string {
	if rootful {
		return "/etc/containers/systemd"
	}

	// The XDG spec says to ignore a relative XDG_CONFIG_HOME...
	config := getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(config) {
		config = filepath.Join(getenv("HOME"), ".config")
	}

	return filepath.Join(config, "containers", "systemd")
}

// Unit is an installed Quadlet with its file, drop-ins merged in.
type Unit struct {
	podman.Quadlet
	File *parser.UnitFile
}

// Load reads the file of each of quadlets, with its drop-ins from its own
// directory and from dir. A file that can't be read is empty, so a broken
// Quadlet still shows up, without what it says.
func Load(quadlets []podman.Quadlet, dir string) []Unit {
	units := make([]Unit, 0, len(quadlets))

	for _, quadlet := range quadlets {
		file, err := read(quadlet.Path, []string{filepath.Dir(quadlet.Path), dir})
		if err != nil {
			file = parser.NewUnitFile()
			file.Filename = quadlet.Name
		}

		units = append(units, Unit{Quadlet: quadlet, File: file})
	}

	return units
}

// read parses the unit at path, and merges in its drop-ins from dirs. Like
// Quadlet, it searches each directory for the drop-in directories of the
// unit, from the most specific ("demo-app.container.d") to the broadest
// ("container.d"). The first drop-in of a name wins, and they're applied
// in the order of their names.
func read(path string, dirs []string) (*parser.UnitFile, error) {
	file, err := parser.ParseUnitFile(path)
	if err != nil {
		return nil, err
	}

	dropins := map[string]string{}

	for _, dir := range slices.Compact(dirs) {
		for _, dropinDir := range file.GetUnitDropinPaths() {
			entries, err := os.ReadDir(filepath.Join(dir, dropinDir))
			if err != nil {
				continue
			}

			for _, entry := range entries {
				name := entry.Name()
				if _, seen := dropins[name]; seen || entry.IsDir() || !strings.HasSuffix(name, ".conf") {
					continue
				}

				dropins[name] = filepath.Join(dir, dropinDir, name)
			}
		}
	}

	names := make([]string, 0, len(dropins))
	for name := range dropins {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		dropin, err := parser.ParseUnitFile(dropins[name])
		if err != nil {
			continue
		}

		file.Merge(dropin)
	}

	return file, nil
}

// Section is the section of the Quadlet's own directives, such as
// "Container" for a .container.
func (u Unit) Section() string {
	ext := u.Ext()
	if ext == "" {
		return ""
	}

	return strings.ToUpper(ext[:1]) + ext[1:]
}

// Last is the last value of key in the Quadlet's own section, or "".
func (u Unit) Last(key string) string {
	value, _ := u.File.Lookup(u.Section(), key)

	return value
}

// All is every value of key in the Quadlet's own section. An empty value
// resets the ones before it, as in systemd.
func (u Unit) All(key string) []string {
	values := u.File.LookupAll(u.Section(), key)
	for i, value := range values {
		values[i] = strings.TrimSpace(value)
	}

	return values
}

// ResourceName is the name of the volume or network a .volume or .network
// creates: its VolumeName= or NetworkName=, or else "systemd-" and its
// base name, as Quadlet names it.
func (u Unit) ResourceName() string {
	var key string

	switch u.Ext() {
	case "volume":
		key = "VolumeName"
	case "network":
		key = "NetworkName"
	default:
		return ""
	}

	if name := u.Last(key); name != "" {
		return name
	}

	return "systemd-" + u.Base()
}

// Parse is the Unit of quadlet with content as its file, for tests that
// have no files.
func Parse(quadlet podman.Quadlet, content string) Unit {
	file := parser.NewUnitFile()
	file.Filename = quadlet.Name
	_ = file.Parse(content)

	return Unit{Quadlet: quadlet, File: file}
}

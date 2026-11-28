// Parts of this file (read) are adapted from loadUnitDropins in Podman's
// cmd/quadlet/main.go, commit 3471162efb28760529d852706c493c29245def6d
// (v6.2.1). Copyright The Podman Authors. Licensed under the Apache License,
// Version 2.0: https://www.apache.org/licenses/LICENSE-2.0

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
	podmanquadlet "go.podman.io/podman/v6/pkg/systemd/quadlet"

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

// SearchDirs are the directories Quadlet reads units and drop-ins from,
// with their subdirectories, as Podman's own Quadlet generator lists them.
// For a user, they include $XDG_CONFIG_HOME/containers/systemd and
// /etc/containers/systemd/users.
func SearchDirs(rootful bool) []string {
	return podmanquadlet.GetUnitDirs(!rootful, true)
}

// Load reads the file of each of quadlets, with its drop-ins from dirs
// (see SearchDirs) and from the directory the file is in. A file that
// can't be read is empty, so a broken Quadlet still shows up, without what
// it says.
func Load(quadlets []podman.Quadlet, dirs []string) []Unit {
	units := make([]Unit, 0, len(quadlets))

	for _, quadlet := range quadlets {
		file, err := read(quadlet.Path, dirs)
		if err != nil {
			file = parser.NewUnitFile()
			file.Filename = quadlet.Name
		}

		units = append(units, Unit{Quadlet: quadlet, File: file})
	}

	return units
}

// read parses the unit at path, and merges in its drop-ins, as Quadlet's
// loadUnitDropins does (cmd/quadlet in Podman, which can't be imported).
// The drop-in directories of the unit go from the most specific
// ("demo-app.container.d") to the broadest ("container.d"), and each is
// looked for in every one of dirs, and then next to the file itself, after
// following a symlink to it. The first drop-in of a name wins, and they're
// applied in the order of their names.
func read(path string, dirs []string) (*parser.UnitFile, error) {
	file, err := parser.ParseUnitFile(path)
	if err != nil {
		return nil, err
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}

	dropins := map[string]string{}
	searched := map[string]bool{}

	for _, dropinDir := range file.GetUnitDropinPaths() {
		for _, dir := range append(slices.Clone(dirs), filepath.Dir(resolved)) {
			candidate := filepath.Join(dir, dropinDir)
			if searched[candidate] {
				continue
			}

			searched[candidate] = true

			entries, err := os.ReadDir(candidate)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				name := entry.Name()
				if _, seen := dropins[name]; seen || entry.IsDir() || filepath.Ext(name) != ".conf" {
					continue
				}

				dropins[name] = filepath.Join(candidate, name)
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

// Labels are the Label= values in the Quadlet's own section whose keys
// start with prefix, read as Quadlet reads them: quoted, and several to a
// line.
func (u Unit) Labels(prefix string) map[string]string {
	labels := map[string]string{}
	all, _ := u.File.LookupAllKeyVal(u.Section(), "Label")

	for key, value := range all {
		if strings.HasPrefix(key, prefix) && value != nil {
			labels[key] = *value
		}
	}

	return labels
}

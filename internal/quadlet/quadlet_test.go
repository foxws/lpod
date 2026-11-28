package quadlet

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

// write creates the files of files under dir, by their path.
func write(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// load writes files to a Quadlet directory, and loads the Quadlet name
// from its subdirectory "demo", as "podman quadlet install" puts it.
func load(t *testing.T, name string, files map[string]string) Unit {
	t.Helper()

	dir := t.TempDir()
	write(t, dir, files)

	return Load([]podman.Quadlet{{Name: name, Path: filepath.Join(dir, "demo", name)}}, []string{dir})[0]
}

func TestTheParserReadsWhatLinesByHandMissed(t *testing.T) {
	unit := load(t, "demo-app.container", map[string]string{
		"demo/demo-app.container": "[Container]\n" +
			"Image = docker.io/library/alpine:3\n" +
			"Volume=old:/old\n" +
			"Volume=\n" +
			"Volume=demo-data.volume:/data\n" +
			"Exec=sleep \\\n  infinity\n" +
			"[Service]\n" +
			"Volume=not-the-container's\n",
	})

	if got := unit.Last("Image"); got != "docker.io/library/alpine:3" {
		t.Errorf("Image is %q, want it with spaces around =", got)
	}

	if got := unit.All("Volume"); !slices.Equal(got, []string{"demo-data.volume:/data"}) {
		t.Errorf("Volume is %q, want the list after its reset, from [Container] only", got)
	}

	if got := unit.Last("Exec"); got != "sleep infinity" {
		t.Errorf("Exec is %q, want the continued line joined", got)
	}
}

func TestDropinsAreMergedInTheOrderOfTheirNames(t *testing.T) {
	unit := load(t, "demo-app.container", map[string]string{
		"demo/demo-app.container":              "[Container]\nImage=one\nEnvironment=A=1\n",
		"demo/demo-app.container.d/b.conf":     "[Container]\nImage=three\n",
		"demo/demo-app.container.d/a.conf":     "[Container]\nImage=two\nEnvironment=B=2\n",
		"demo/demo-app.container.d/x.txt":      "[Container]\nImage=not-a-drop-in\n",
		"demo/demo-.container.d/c.conf":        "[Container]\nEnvironment=C=3\n",
		"container.d/d.conf":                   "[Container]\nEnvironment=D=4\n",
		"demo-app.container.d/lpod-x.conf":     "[Container]\nEnvironment=E=5\n",
		"demo/demo-app.container.d/sub/x.conf": "[Container]\nImage=in-a-subdirectory\n",
	})

	if got := unit.Last("Image"); got != "three" {
		t.Errorf("Image is %q, want b.conf's", got)
	}

	if got := unit.All("Environment"); !slices.Equal(got, []string{"A=1", "B=2", "C=3", "D=4", "E=5"}) {
		t.Errorf("Environment is %q, want every drop-in's", got)
	}
}

func TestTheMostSpecificDropinOfANameWins(t *testing.T) {
	unit := load(t, "demo-app.container", map[string]string{
		"demo/demo-app.container":               "[Container]\nImage=one\n",
		"demo/container.d/memory.conf":          "[Container]\nImage=broad\n",
		"demo/demo-app.container.d/memory.conf": "[Container]\nImage=specific\n",
	})

	if got := unit.Last("Image"); got != "specific" {
		t.Errorf("Image is %q, want the specific drop-in's", got)
	}
}

func TestASpecificDropinWinsOverABroadOneOfTheSameNameElsewhere(t *testing.T) {
	// The broad one is next to the Quadlet, and the specific one in the
	// Quadlet directory. Quadlet looks for the specific directory in every
	// place before the broad one...
	unit := load(t, "demo-app.container", map[string]string{
		"demo/demo-app.container":          "[Container]\nImage=one\n",
		"demo/container.d/memory.conf":     "[Container]\nImage=broad\n",
		"demo-app.container.d/memory.conf": "[Container]\nImage=specific\n",
	})

	if got := unit.Last("Image"); got != "specific" {
		t.Errorf("Image is %q, want the specific drop-in's", got)
	}
}

func TestDropinsNextToASymlinkedQuadletsFileCount(t *testing.T) {
	real := t.TempDir()
	write(t, real, map[string]string{
		"demo-app.container":              "[Container]\nImage=one\n",
		"demo-app.container.d/image.conf": "[Container]\nImage=next-to-the-file\n",
	})

	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(real, "demo-app.container"), filepath.Join(dir, "demo-app.container")); err != nil {
		t.Fatal(err)
	}

	unit := Load([]podman.Quadlet{{Name: "demo-app.container", Path: filepath.Join(dir, "demo-app.container")}}, []string{dir})[0]

	if got := unit.Last("Image"); got != "next-to-the-file" {
		t.Errorf("Image is %q, want the drop-in next to the real file", got)
	}
}

func TestSearchDirsAreQuadlets(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/srv/config")

	if dirs := SearchDirs(false); !slices.Contains(dirs, "/srv/config/containers/systemd") {
		t.Errorf("got %q, want the user's Quadlet directory", dirs)
	}

	if dirs := SearchDirs(true); !slices.Contains(dirs, "/etc/containers/systemd") {
		t.Errorf("got %q, want the system's Quadlet directory", dirs)
	}
}

func TestAQuadletThatCantBeReadIsEmpty(t *testing.T) {
	unit := Load([]podman.Quadlet{{Name: "gone.container", Path: filepath.Join(t.TempDir(), "gone.container")}}, nil)[0]

	if unit.Name != "gone.container" || unit.Last("Image") != "" || unit.All("Volume") != nil && len(unit.All("Volume")) != 0 {
		t.Errorf("got %+v, want an empty unit", unit)
	}
}

func TestResourceName(t *testing.T) {
	named := load(t, "demo-data.volume", map[string]string{"demo/demo-data.volume": "[Volume]\nVolumeName=demo-data\n"})
	if got := named.ResourceName(); got != "demo-data" {
		t.Errorf("got %s, want VolumeName=", got)
	}

	unnamed := load(t, "demo.network", map[string]string{"demo/demo.network": "[Network]\n"})
	if got := unnamed.ResourceName(); got != "systemd-demo" {
		t.Errorf("got %s, want Quadlet's own name", got)
	}

	container := load(t, "demo-app.container", map[string]string{"demo/demo-app.container": "[Container]\n"})
	if got := container.ResourceName(); got != "" {
		t.Errorf("got %s for a container, want none", got)
	}
}

func TestDir(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(name string) string { return vars[name] }
	}

	cases := []struct {
		rootful bool
		vars    map[string]string
		want    string
	}{
		{true, nil, "/etc/containers/systemd"},
		{false, map[string]string{"HOME": "/home/blue"}, "/home/blue/.config/containers/systemd"},
		{false, map[string]string{"HOME": "/home/blue", "XDG_CONFIG_HOME": "/srv/config"}, "/srv/config/containers/systemd"},
		{false, map[string]string{"HOME": "/home/blue", "XDG_CONFIG_HOME": "config"}, "/home/blue/.config/containers/systemd"},
	}

	for _, c := range cases {
		if got := Dir(c.rootful, env(c.vars)); got != c.want {
			t.Errorf("Dir(%v, %v) = %s, want %s", c.rootful, c.vars, got, c.want)
		}
	}
}

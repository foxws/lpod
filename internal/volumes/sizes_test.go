package volumes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/podman"
)

// sizesHost answers "volume inspect --all" with a named, an anonymous, an
// NFS and a device volume, and du with what it measured, and records the
// du it ran.
func sizesHost(du *string, duOut string, duErr error) podman.Client {
	return podman.Client{Binary: "podman", Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "podman" && args[0] == "volume" {
			return []byte(`[
				{"Name": "demo-data", "Driver": "local", "Mountpoint": "/volumes/demo-data/_data"},
				{"Name": "f00dcafe", "Driver": "local", "Mountpoint": "/volumes/f00dcafe/_data", "Anonymous": true},
				{"Name": "media", "Driver": "local", "Mountpoint": "/volumes/media/_data", "Options": {"type": "nfs", "device": "nas:/media"}},
				{"Name": "plugin", "Driver": "other", "Mountpoint": "/plugin"},
				{"Name": "bound", "Driver": "local", "Mountpoint": "/volumes/bound/_data", "Options": {"device": "/srv/media", "o": "bind"}}
			]`), nil
		}

		*du = strings.Join(append([]string{name}, args...), " ")

		return []byte(duOut), duErr
	}}
}

func TestSizesRunsDuInTheUsersNamespace(t *testing.T) {
	var du string
	client := sizesHost(&du, "4096\t/volumes/f00dcafe/_data\n48318382080\t/volumes/demo-data/_data\n", nil)

	sizes, err := Sizes(context.Background(), client, false)
	if err != nil {
		t.Fatal(err)
	}

	if du != "podman unshare du -s --block-size=1 -- /volumes/demo-data/_data /volumes/f00dcafe/_data" {
		t.Errorf("ran %q", du)
	}

	out, _ := json.Marshal(sizes)
	if string(out) != `{"contract":1,"sizes":[{"name":"demo-data","size":48318382080},{"name":"f00dcafe","size":4096}]}` {
		t.Errorf("got %s", out)
	}
}

func TestSizesRunsDuOnItsOwnAsRoot(t *testing.T) {
	var du string

	if _, err := Sizes(context.Background(), sizesHost(&du, "", nil), true); err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(du, "du -s --block-size=1 -- ") {
		t.Errorf("ran %q, want du without podman unshare", du)
	}
}

func TestSizesKeepsWhatDuMeasuredWhenItFails(t *testing.T) {
	var du string
	client := sizesHost(&du, "4096\t/volumes/f00dcafe/_data\n", errors.New("podman unshare: du: cannot read directory: Permission denied"))

	sizes, err := Sizes(context.Background(), client, false)
	if err != nil {
		t.Fatal(err)
	}

	if sizes.Sizes[0].Size != nil || *sizes.Sizes[1].Size != 4096 {
		t.Errorf("got %+v, want demo-data null and f00dcafe measured", sizes.Sizes)
	}
}

func TestSizesWithoutVolumesRunsNoDu(t *testing.T) {
	client := podman.Client{Binary: "podman", Run: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] != "volume" {
			t.Errorf("ran %q", args)
		}

		return []byte("[]"), nil
	}}

	sizes, err := Sizes(context.Background(), client, false)
	out, _ := json.Marshal(sizes)

	if err != nil || string(out) != `{"contract":1,"sizes":[]}` {
		t.Errorf("got %s and %v", out, err)
	}
}

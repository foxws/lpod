package volumes

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/podman"
)

// sizesTimeout is how long Sizes waits. du walks every file of every
// volume, which takes a while on big ones.
const sizesTimeout = 10 * time.Minute

// Sizes measures the disk space of each named and anonymous volume with du,
// for "lpod volumes sizes --json". A volume backed by a folder, a device or
// an NFS share is left out: it can be a whole media library, or remote.
//
// A rootless user's volumes hold files owned by its subordinate IDs, which
// only "podman unshare" can read all of, so du runs there. As root, it runs
// on its own.
func Sizes(ctx context.Context, client podman.Client, rootful bool) (contract.VolumeSizes, error) {
	ctx, cancel := context.WithTimeout(ctx, sizesTimeout)
	defer cancel()

	volumes, err := client.Volumes(ctx)
	if err != nil {
		return contract.VolumeSizes{}, err
	}

	var measured []podman.Volume
	var paths []string

	for _, volume := range volumes {
		if cmp.Or(volume.Driver, "local") == "local" && volume.Options["device"] == "" && volume.Mountpoint != "" {
			measured = append(measured, volume)
			paths = append(paths, volume.Mountpoint)
		}
	}

	var usage map[string]int64

	if len(paths) > 0 {
		du := append([]string{"du", "-s", "--block-size=1", "--"}, paths...)
		if !rootful {
			du = append([]string{client.Binary, "unshare"}, du...)
		}

		// du still prints the totals it could work out when it can't read
		// some files, and fails then, so a failure isn't one here: a
		// volume it didn't measure is null...
		out, _ := client.Run(ctx, du[0], du[1:]...)
		usage = parseDu(out)
	}

	sizes := make([]contract.VolumeSize, 0, len(measured))

	for _, volume := range measured {
		size := contract.VolumeSize{Name: volume.Name}

		if bytes, ok := usage[volume.Mountpoint]; ok {
			size.Size = &bytes
		}

		sizes = append(sizes, size)
	}

	slices.SortFunc(sizes, func(a, b contract.VolumeSize) int { return cmp.Compare(a.Name, b.Name) })

	return contract.VolumeSizes{Contract: contract.Current, Sizes: sizes}, nil
}

// parseDu reads the "SIZE<tab>PATH" lines of "du -s", by path.
func parseDu(out []byte) map[string]int64 {
	usage := map[string]int64{}
	lines := bufio.NewScanner(bytes.NewReader(out))

	for lines.Scan() {
		size, path, ok := strings.Cut(lines.Text(), "\t")
		if !ok {
			continue
		}

		if bytes, err := strconv.ParseInt(size, 10, 64); err == nil {
			usage[path] = bytes
		}
	}

	return usage
}

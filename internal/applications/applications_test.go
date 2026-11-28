package applications

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/foxws/lpod/internal/contract"
	"github.com/foxws/lpod/internal/journal"
	"github.com/foxws/lpod/internal/podman"
	"github.com/foxws/lpod/internal/quadlet"
	"github.com/foxws/lpod/internal/systemd"
)

// encode writes v as JSON on one line, without escaping ">" as
// contract.Write doesn't.
func encode(t *testing.T, v any) string {
	t.Helper()

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		t.Fatal(err)
	}

	return strings.TrimSuffix(out.String(), "\n")
}

func unit(name, app, unitName, content string) quadlet.Unit {
	return quadlet.Parse(podman.Quadlet{Name: name, UnitName: unitName, App: app, Path: "/q/" + app + "/" + name}, content)
}

func text(s string) *string { return &s }

func bytesOf(n uint64) *uint64 { return &n }

// host has immich, a .pod with a server built from a .build and a valkey,
// and paperless, a .kube.
func host() Host {
	units := []quadlet.Unit{
		unit("immich.pod", "immich", "immich-pod.service", "[Pod]\nPublishPort=2283:2283\nLabel=io.foxws.pods.url=https://photos.test\n"),
		unit("immich-server.container", "immich", "immich-server.service", "[Container]\nImage=immich-server.build\nContainerName=immich-server\nPod=immich.pod\n"+
			"AutoUpdate=registry\nSecret=immich-db-password,type=env\nSecret=immich-db-password,type=mount\n"+
			"Label=\"io.foxws.pods.description=Self-hosted photos\" io.foxws.pods.url=https://ignored.test\n"),
		unit("immich-valkey.container", "immich", "immich-valkey.service", "[Container]\nImage=docker.io/valkey/valkey:8\nPod=immich.pod\nPublishPort=127.0.0.1:6379:6379/tcp\nPublishPort=6380\n"),
		unit("immich-server.build", "immich", "immich-server-build.service", "[Build]\nImageTag=localhost/immich-server:dev\n"),
		unit("immich-library.volume", "immich", "immich-library-volume.service", "[Volume]\nVolumeName=immich-library\n"),
		unit("paperless.kube", "paperless", "paperless.service", "[Kube]\nYaml=paperless.yaml\n"),
		unit("notes.volume", "notes", "notes-volume.service", "[Volume]\n"),
	}

	return Host{
		Units: units,
		All:   units,
		States: []systemd.Unit{
			{"Id": "immich-pod.service", "ActiveState": "active", "Result": "success", "StateChangeTimestamp": "@1795000000"},
			{"Id": "immich-server.service", "ActiveState": "active"},
			{"Id": "immich-valkey.service", "ActiveState": "inactive"},
		},
		Containers: []contract.Container{
			{Name: "immich-server", Image: "localhost/immich-server:dev", State: "running", Unit: text("immich-server.service"), CPU: new(float64), Memory: bytesOf(100)},
			{Name: "paperless-web", Image: "paperless:2", State: "running", Unit: text("paperless.service"), Memory: bytesOf(50)},
			{Name: "paperless-db", Image: "postgres:16", State: "exited", Unit: text("paperless.service")},
		},
	}
}

func TestBuild(t *testing.T) {
	apps := Build(host())

	if len(apps) != 2 {
		t.Fatalf("got %d applications, want immich and paperless, without notes", len(apps))
	}

	want := `{"name":"immich","unit":"immich-pod.service","kind":"pod","status":"running","result":"success","since":"2026-11-18T11:06:40+00:00",` +
		`"image":"localhost/immich-server:dev","pods":1,"containers":2,"ports":["0.0.0.0:2283->2283/tcp","127.0.0.1:6379->6379/tcp"],"memory":100,"auto_update":"registry",` +
		`"labels":{"io.foxws.pods.description":"Self-hosted photos","io.foxws.pods.url":"https://photos.test"},` +
		`"path":"/q/immich","volumes":[{"name":"immich-library","size":null}],"secrets":["immich-db-password"],` +
		`"items":[{"name":"immich-server","image":"localhost/immich-server:dev","state":"running","cpu":0,"memory":100},` +
		`{"name":"systemd-immich-valkey","image":"docker.io/valkey/valkey:8","state":"inactive","cpu":null,"memory":null}]}`

	if got := encode(t, apps[0].ApplicationDetail); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}

	if got := strings.Join(apps[0].Services, " "); got != "immich-server.service immich-valkey.service immich-pod.service" {
		t.Errorf("got services %s", got)
	}
}

func TestAKubeCountsTheContainersItRuns(t *testing.T) {
	paperless := Build(host())[1]

	if paperless.Kind != "kube" || paperless.Containers != 2 || paperless.Image != nil || *paperless.Memory != 50 || paperless.Status != "unknown" {
		t.Errorf("got %+v", paperless.Application)
	}

	if !paperless.Error.IsZero() {
		t.Error("got an error for an application that didn't fail")
	}
}

func TestAFailedApplicationGetsItsLastError(t *testing.T) {
	var call string
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch {
		case name == "journalctl":
			call = strings.Join(args, " ")

			return []byte("Error: no secret\n"), nil
		case name == "systemctl":
			return []byte("Id=glance.service\nActiveState=failed\nResult=exit-code\n"), nil
		case args[0] == "quadlet":
			return []byte(`[{"Name": "glance.container", "UnitName": "glance.service"}]`), nil
		}

		return []byte("[]"), nil
	}

	sources := Sources{
		Podman:  podman.Client{Binary: "podman", Run: run},
		Systemd: systemd.New(false, run),
		Journal: journal.Reader{User: true, Run: run},
	}

	apps, err := List(context.Background(), sources)
	if err != nil {
		t.Fatal(err)
	}

	if got := encode(t, apps[0].Application); !strings.HasSuffix(got, `"labels":{},"error":"Error: no secret"}`) {
		t.Errorf("got %s", got)
	}

	if !strings.Contains(call, "--unit=glance.service --priority=err") {
		t.Errorf("ran journalctl %s", call)
	}

	found, err := Find(context.Background(), sources, "glance")
	if err != nil || found.Name != "glance" {
		t.Errorf("got %+v and %v", found.Application, err)
	}

	if _, err := Find(context.Background(), sources, "ghost"); err == nil || err.Error() != "No application or Quadlet named ghost is installed." {
		t.Errorf("got %v for an unknown application", err)
	}
}

func TestFindTakesAQuadletOfTheApplication(t *testing.T) {
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "podman" && args[0] == "quadlet" {
			return []byte(`[{"Name": "demo-app.container", "UnitName": "demo-app.service", "App": "demo"}, {"Name": "demo-data.volume", "UnitName": "demo-data-volume.service", "App": "demo"}, {"Name": "lone.volume", "UnitName": "lone-volume.service"}]`), nil
		}

		return []byte("[]"), nil
	}

	sources := Sources{Podman: podman.Client{Binary: "podman", Run: run}, Systemd: systemd.New(false, run), Journal: journal.Reader{Run: run}}

	if found, err := Find(context.Background(), sources, "demo-data"); err != nil || found.Name != "demo" {
		t.Errorf("got %q and %v, want demo", found.Name, err)
	}

	if _, err := Find(context.Background(), sources, "lone"); err == nil || err.Error() != "lone has no container, pod or kube Quadlet." {
		t.Errorf("got %v", err)
	}
}

package handoff

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptPathUsesTheScriptTheEnvironmentNames(t *testing.T) {
	getenv := func(name string) string {
		if name == ScriptEnv {
			return "/src/lpod/bin/lpod"
		}

		return ""
	}

	script, err := ScriptPath("/usr/local/bin/lpod", getenv)
	if err != nil {
		t.Fatal(err)
	}

	if script != "/src/lpod/bin/lpod" {
		t.Errorf("got %s, want /src/lpod/bin/lpod", script)
	}
}

func TestScriptPathFindsTheScriptNextToTheBinaryASymlinkPointsAt(t *testing.T) {
	installed := t.TempDir()
	binary := filepath.Join(installed, "lpod")
	writeFile(t, binary, "binary")

	onPath := t.TempDir()
	link := filepath.Join(onPath, "lpod")
	if err := os.Symlink(binary, link); err != nil {
		t.Fatal(err)
	}

	script, err := ScriptPath(link, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}

	// The temporary directory itself may be behind a symlink.
	resolved, err := filepath.EvalSymlinks(installed)
	if err != nil {
		t.Fatal(err)
	}

	if want := filepath.Join(resolved, ScriptName); script != want {
		t.Errorf("got %s, want %s", script, want)
	}
}

func TestExecSaysHowToFixAMissingScript(t *testing.T) {
	t.Setenv(ScriptEnv, filepath.Join(t.TempDir(), ScriptName))

	err := Exec([]string{"apps", "--json"})

	if err == nil || !strings.Contains(err.Error(), "Install lpod again") {
		t.Errorf("got %v, want an error that says to install lpod again", err)
	}
}

func TestExecHandsArgumentsEnvironmentAndStdinToTheScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), ScriptName)
	writeFile(t, script, "#!/bin/sh\nprintf '%s\\n' \"$0\" \"$@\" \"probe=$LPOD_HANDOFF_PROBE\"\ncat\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "secret", "create", "a b", "--stdin")
	cmd.Env = append(os.Environ(), "LPOD_HANDOFF_HELPER=1", ScriptEnv+"="+script, "LPOD_HANDOFF_PROBE=kept")
	cmd.Stdin = strings.NewReader("from stdin\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}

	want := strings.Join([]string{script, "secret", "create", "a b", "--stdin", "probe=kept", "from stdin", ""}, "\n")
	if string(output) != want {
		t.Errorf("got %q, want %q", output, want)
	}
}

// TestHelperProcess is the lpod binary for TestExecHandsArgumentsEnvironmentAndStdinToTheScript:
// it hands the arguments after "--" to the script, as main does.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("LPOD_HANDOFF_HELPER") != "1" {
		return
	}

	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]

			break
		}
	}

	err := Exec(args)

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

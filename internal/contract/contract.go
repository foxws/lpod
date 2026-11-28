// Package contract has the JSON payloads that lpod's --json commands print,
// which tools such as Pods parse. tests/fixtures/contract-1 has an example of
// each, and docs/commands.md describes them.
package contract

import (
	"encoding/json"
	"io"
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

// Write prints payload as one JSON object, indented as jq prints it.
func Write(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	return encoder.Encode(payload)
}

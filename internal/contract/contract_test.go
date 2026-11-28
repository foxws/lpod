package contract

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestTimeIsUTCWithoutAFraction(t *testing.T) {
	at := time.Date(2026, 11, 15, 16, 0, 0, 500, time.FixedZone("CET", 3600))

	out, err := json.Marshal(Time{at})
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != `"2026-11-15T15:00:00+00:00"` {
		t.Errorf("got %s", out)
	}
}

func TestTheZeroTimeIsNull(t *testing.T) {
	out, err := json.Marshal(struct {
		At Time `json:"at"`
	}{})
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != `{"at":null}` {
		t.Errorf("got %s", out)
	}
}

func TestString(t *testing.T) {
	if String("") != nil {
		t.Error("got a string for empty, want nil")
	}

	if got := String("overlay"); got == nil || *got != "overlay" {
		t.Errorf("got %v, want overlay", got)
	}
}

func TestWriteIndentsAsJqDoes(t *testing.T) {
	var out bytes.Buffer

	if err := Write(&out, map[string]any{"contract": 1, "name": "<a&b>"}); err != nil {
		t.Fatal(err)
	}

	if out.String() != "{\n  \"contract\": 1,\n  \"name\": \"<a&b>\"\n}\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestParseTime(t *testing.T) {
	out, _ := json.Marshal(ParseTime("2026-11-17T11:00:00.123456789+01:00"))
	if string(out) != `"2026-11-17T10:00:00+00:00"` {
		t.Errorf("got %s", out)
	}

	for _, none := range []string{"", "0001-01-01T00:00:00Z", "yesterday"} {
		if !ParseTime(none).IsZero() {
			t.Errorf("ParseTime(%q) isn't null", none)
		}
	}
}

func TestUnix(t *testing.T) {
	if !Unix(0).IsZero() || Unix(1795000000).Unix() != 1795000000 {
		t.Error("got the wrong time")
	}
}

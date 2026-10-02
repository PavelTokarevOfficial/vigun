package httpapi

import (
	"testing"
	"time"
)

func TestParseClipWindowIncludesEndDate(t *testing.T) {
	startedAt, endedAt, err := parseClipWindow("2026-09-01", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC); !startedAt.Equal(want) {
		t.Fatalf("unexpected start: %s", startedAt)
	}
	if want := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC); !endedAt.Equal(want) {
		t.Fatalf("unexpected end: %s", endedAt)
	}
}

func TestParseClipWindowRejectsInvalidRange(t *testing.T) {
	if _, _, err := parseClipWindow("2026-09-08", "2026-09-01"); err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestParseClipWindowPreservesRFC3339Timezone(t *testing.T) {
	startedAt, endedAt, err := parseClipWindow(
		"2020-10-02T21:00:00Z",
		"2020-10-02T22:00:00Z",
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2020, 10, 2, 21, 0, 0, 0, time.UTC); !startedAt.Equal(want) {
		t.Fatalf("unexpected start: %s", startedAt)
	}
	if want := time.Date(2020, 10, 2, 22, 0, 0, 0, time.UTC); !endedAt.Equal(want) {
		t.Fatalf("unexpected end: %s", endedAt)
	}
}

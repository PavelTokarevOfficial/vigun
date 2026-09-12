package processing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/finde-clip/finde-v2/back/internal/composition"
)

func TestApplyTimelineToSRTRetimesAndDropsCues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subtitles.srt")
	input := "1\n00:00:01,000 --> 00:00:02,000\nDrop me\n\n2\n00:00:05,500 --> 00:00:06,500\nFirst kept\n\n3\n00:00:10,000 --> 00:00:12,000\nSecond kept\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	segments := []composition.Segment{{ID: "a", Start: 5, End: 7}, {ID: "b", Start: 10, End: 11}}
	if err := applyTimelineToSRT(path, segments); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, expected := range []string{"00:00:00,500 --> 00:00:01,500", "First kept", "00:00:02,000 --> 00:00:03,000", "Second kept"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("retimed SRT does not contain %q:\n%s", expected, got)
		}
	}
	if strings.Contains(got, "Drop me") {
		t.Fatalf("cue outside selected segments was kept:\n%s", got)
	}
}

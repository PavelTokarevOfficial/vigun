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

func TestApplyTimelineToSRTLeavesGapForInsertedAsset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subtitles.srt")
	input := "1\n00:00:01,000 --> 00:00:02,000\nBefore ad\n\n2\n00:00:06,000 --> 00:00:07,000\nAfter ad\n"
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	segments := []composition.Segment{
		{ID: "before", Source: "clip", Start: 0, End: 4},
		{ID: "ad", Source: "asset", AssetID: "asset-1", Start: 0, End: 3},
		{ID: "after", Source: "clip", Start: 4, End: 8},
	}
	if err := applyTimelineToSRT(path, segments); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, expected := range []string{"00:00:01,000 --> 00:00:02,000", "00:00:09,000 --> 00:00:10,000"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("retimed SRT does not contain %q:\n%s", expected, got)
		}
	}
}

func TestLayerSubtitleFilesClipsCuesToLayerRange(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "subtitles.srt")
	input := "1\n00:00:01,000 --> 00:00:03,000\nFirst\n\n2\n00:00:05,000 --> 00:00:07,000\nSecond\n"
	if err := os.WriteFile(source, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	paths, err := layerSubtitleFiles(source, dir, []composition.Layer{{ID: "captions", Type: "subtitles", Visible: true, StartTime: 2, EndTime: 6}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(paths["captions"])
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, expected := range []string{"00:00:02,000 --> 00:00:03,000", "00:00:05,000 --> 00:00:06,000"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("layer SRT does not contain %q:\n%s", expected, got)
		}
	}
}

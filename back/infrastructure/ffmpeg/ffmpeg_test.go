package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/processing"
)

func TestRenderCreatesVerticalVideoWithBurnedSubtitles(t *testing.T) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	filters, err := exec.Command(bin, "-hide_banner", "-filters").CombinedOutput()
	if err != nil {
		t.Fatalf("inspect ffmpeg filters: %v: %s", err, filters)
	}
	if !strings.Contains(string(filters), "subtitles") {
		t.Skip("ffmpeg was built without the subtitles/libass filter")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	subtitles := filepath.Join(dir, "subtitles.srt")
	output := filepath.Join(dir, "with-subtitles.mp4")

	makeSource := exec.Command(
		bin,
		"-y",
		"-f", "lavfi", "-i", "testsrc2=size=160x90:rate=10",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=44100",
		"-t", "1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
		source,
	)
	if log, err := makeSource.CombinedOutput(); err != nil {
		t.Fatalf("create source video: %v: %s", err, log)
	}
	if err := os.WriteFile(subtitles, []byte("1\n00:00:00,000 --> 00:00:00,900\nTest subtitle\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = New(bin).Render(context.Background(), processing.RenderInput{
		SourcePath:   source,
		SubtitlePath: subtitles,
		OutputPath:   output,
		Width:        1080,
		Height:       1920,
		Blur:         10,
		Preset:       "ultrafast",
	})
	if err != nil {
		t.Fatalf("render video: %v", err)
	}

	probe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed")
	}
	check := exec.Command(probe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", output)
	got, err := check.Output()
	if err != nil {
		t.Fatalf("inspect rendered video: %v", err)
	}
	if strings.TrimSpace(string(got)) != "1080,1920" {
		t.Fatalf("unexpected rendered dimensions %q", strings.TrimSpace(string(got)))
	}
}

func TestRenderInsertsSilentAssetBetweenSourceFragments(t *testing.T) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	asset := filepath.Join(dir, "ad.mp4")
	output := filepath.Join(dir, "montage.mp4")

	makeMedia := func(path, color string, withAudio bool) {
		args := []string{"-y", "-f", "lavfi", "-i", "color=c=" + color + ":s=64x96:r=10"}
		if withAudio {
			args = append(args, "-f", "lavfi", "-i", "sine=frequency=800:sample_rate=48000")
		}
		args = append(args, "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p")
		if withAudio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
		args = append(args, path)
		if log, runErr := exec.Command(bin, args...).CombinedOutput(); runErr != nil {
			t.Fatalf("create %s media: %v: %s", color, runErr, log)
		}
	}
	makeMedia(source, "red", true)
	makeMedia(asset, "blue", false)

	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 64, Height: 96, FPS: 10, Background: "#000000"},
		Timeline: &composition.Timeline{Segments: []composition.Segment{
			{ID: "before", Source: "clip", Start: 0, End: 0.3},
			{ID: "ad", Source: "asset", AssetID: "ad", Start: 0, End: 0.4},
			{ID: "after", Source: "clip", Start: 0.7, End: 1},
		}},
		Layers: []composition.Layer{{ID: "video", Type: "video", Source: "clip", Width: 64, Height: 96, Visible: true, Opacity: 1, Fit: "stretch"}},
	}
	err = New(bin).Render(context.Background(), processing.RenderInput{
		SourcePath:  source,
		OutputPath:  output,
		Preset:      "ultrafast",
		Composition: config,
		AssetPaths:  map[string]string{"ad": asset},
	})
	if err != nil {
		t.Fatalf("render montage: %v", err)
	}

	colorAt := func(second string) []byte {
		command := exec.Command(bin, "-v", "error", "-ss", second, "-i", output, "-frames:v", "1", "-vf", "scale=1:1", "-f", "rawvideo", "-pix_fmt", "rgb24", "pipe:1")
		pixels, readErr := command.Output()
		if readErr != nil || len(pixels) < 3 {
			t.Fatalf("read frame at %s: %v", second, readErr)
		}
		return pixels
	}
	before := colorAt("0.1")
	inserted := colorAt("0.5")
	if before[0] <= before[2] {
		t.Fatalf("first source fragment is not red: rgb=%v", before[:3])
	}
	if inserted[2] <= inserted[0] {
		t.Fatalf("inserted asset fragment is not blue: rgb=%v", inserted[:3])
	}
}

func TestRenderDrawsLinkedTimelineAssetAsMovableLayer(t *testing.T) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	asset := filepath.Join(dir, "ad.mp4")
	output := filepath.Join(dir, "linked-montage.mp4")
	makeColorVideo := func(path, color string) {
		command := exec.Command(bin, "-y", "-f", "lavfi", "-i", "color=c="+color+":s=64x96:r=10", "-f", "lavfi", "-i", "sine=frequency=800:sample_rate=48000", "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", path)
		if log, runErr := command.CombinedOutput(); runErr != nil {
			t.Fatalf("create %s media: %v: %s", color, runErr, log)
		}
	}
	makeColorVideo(source, "red")
	makeColorVideo(asset, "blue")

	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 64, Height: 96, FPS: 10, Background: "#000000"},
		Timeline: &composition.Timeline{Segments: []composition.Segment{
			{ID: "before", Source: "clip", Start: 0, End: 0.3},
			{ID: "ad", Source: "asset", AssetID: "ad", Start: 0, End: 0.4},
			{ID: "after", Source: "clip", Start: 0.7, End: 1},
		}},
		Layers: []composition.Layer{
			{ID: "clip", Type: "video", Source: "clip", Width: 64, Height: 96, Visible: true, Opacity: 1, Fit: "stretch"},
			{ID: "ad-layer", Type: "video", Source: "asset", AssetID: "ad", TimelineSegmentID: "ad", Width: 64, Height: 96, Visible: true, Opacity: 1, Fit: "stretch", StartTime: 0.3, EndTime: 0.7},
		},
	}
	err = New(bin).Render(context.Background(), processing.RenderInput{
		SourcePath:  source,
		OutputPath:  output,
		Preset:      "ultrafast",
		Composition: config,
		AssetPaths:  map[string]string{"ad": asset},
	})
	if err != nil {
		t.Fatalf("render linked montage: %v", err)
	}

	command := exec.Command(bin, "-v", "error", "-ss", "0.5", "-i", output, "-frames:v", "1", "-vf", "scale=1:1", "-f", "rawvideo", "-pix_fmt", "rgb24", "pipe:1")
	pixel, err := command.Output()
	if err != nil || len(pixel) < 3 {
		t.Fatalf("read inserted frame: %v", err)
	}
	if pixel[2] <= pixel[0] {
		t.Fatalf("linked asset layer is not blue: rgb=%v", pixel[:3])
	}
}

func TestEscapeFilterPath(t *testing.T) {
	got := escapeFilterPath("/tmp/it's: a\\file.srt")
	if got != "/tmp/it\\'s\\: a\\\\file.srt" {
		t.Fatalf("unexpected escaped path %q", got)
	}
}

func TestScaleFilterRoundsOddContainDimensionsDown(t *testing.T) {
	filter := scaleFilter(composition.Layer{Width: 1080, Height: 607, Fit: "contain", Opacity: 1})
	if !strings.Contains(filter, "pad=1080:606") {
		t.Fatalf("odd dimension was not normalized: %s", filter)
	}
}

func TestBuildFilterSupportsVideoSourceAndBlurBlock(t *testing.T) {
	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 1080, Height: 1920, FPS: 30, Background: "#000000"},
		Layers: []composition.Layer{
			{ID: "video", Type: "video", Source: "clip", Width: 1080, Height: 1920, Visible: true, Opacity: 1, Fit: "cover"},
			{ID: "blur", Type: "blur", X: 100, Y: 200, Width: 400, Height: 300, Visible: true, Opacity: 0.7, Filters: composition.Filters{Blur: 18, Brightness: -0.3}},
		},
	}
	filter, video, _, err := buildFilter(config, "", nil, map[string]int{}, map[int]bool{0: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"[0:v]scale=", "split=2", "crop=400:300:100:200", "boxblur=18:10", "eq=brightness=-0.3", "colorchannelmixer=aa=0.7"} {
		if !strings.Contains(filter, fragment) {
			t.Fatalf("filter does not contain %q: %s", fragment, filter)
		}
	}
	if video != "[base2]" {
		t.Fatalf("unexpected output label %q", video)
	}
}

func TestBuildFilterConcatenatesTimelineSegments(t *testing.T) {
	config := composition.Config{
		Version:  composition.CurrentVersion,
		Canvas:   composition.Canvas{Width: 1080, Height: 1920, FPS: 30, Background: "#000000"},
		Timeline: &composition.Timeline{Segments: []composition.Segment{{ID: "a", Start: 1, End: 3}, {ID: "b", Start: 5, End: 8}}},
		Layers:   []composition.Layer{{ID: "video", Type: "video", Source: "clip", Width: 1080, Height: 1920, Visible: true, Opacity: 1}},
	}
	filter, _, audio, err := buildFilter(config, "", nil, map[string]int{}, map[int]bool{0: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"[0:v]trim=start=1:end=3", "[0:v]trim=start=5:end=8", "concat=n=2:v=1:a=0", "[0:a]atrim=start=1:end=3", "concat=n=2:v=0:a=1[clipaudio]"} {
		if !strings.Contains(filter, fragment) {
			t.Fatalf("timeline filter does not contain %q: %s", fragment, filter)
		}
	}
	if audio != "[clipaudio]" {
		t.Fatalf("unexpected timeline audio label %q", audio)
	}
}

func TestBuildFilterInsertsAssetVideoIntoTimeline(t *testing.T) {
	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 1080, Height: 1920, FPS: 30, Background: "#000000"},
		Timeline: &composition.Timeline{Segments: []composition.Segment{
			{ID: "before", Source: "clip", Start: 0, End: 4},
			{ID: "ad", Source: "asset", AssetID: "asset-1", Start: 0, End: 3},
			{ID: "after", Source: "clip", Start: 4, End: 8},
		}},
		Layers: []composition.Layer{{ID: "video", Type: "video", Source: "clip", Width: 1080, Height: 1920, Visible: true, Opacity: 1}},
	}
	filter, _, audio, err := buildFilter(config, "", nil, map[string]int{"asset-1": 1}, map[int]bool{0: true, 1: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"[0:v]trim=start=0:end=4", "[1:v]trim=start=0:end=3", "[0:v]trim=start=4:end=8", "concat=n=3:v=1:a=0", "[1:a]atrim=start=0:end=3", "concat=n=3:v=0:a=1[clipaudio]"} {
		if !strings.Contains(filter, fragment) {
			t.Fatalf("asset timeline filter does not contain %q: %s", fragment, filter)
		}
	}
	if audio != "[clipaudio]" {
		t.Fatalf("unexpected timeline audio label %q", audio)
	}
}

func TestBuildFilterAppliesLayerTimelineRanges(t *testing.T) {
	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 1080, Height: 1920, FPS: 30, Background: "#000000"},
		Layers: []composition.Layer{
			{ID: "image", Type: "image", AssetID: "asset-1", X: 10, Y: 20, Width: 300, Height: 200, Visible: true, Opacity: 1, StartTime: 2, EndTime: 5},
			{ID: "text", Type: "text", Text: "Hello", X: 10, Y: 20, Width: 300, Height: 100, Visible: true, Opacity: 1, StartTime: 1, EndTime: 4},
		},
	}
	filter, _, _, err := buildFilter(config, "", nil, map[string]int{"asset-1": 1}, map[int]bool{0: true, 1: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"setpts=PTS-STARTPTS+2/TB", "enable='between(t,2,5)'", "enable='between(t,1,4)'"} {
		if !strings.Contains(filter, fragment) {
			t.Fatalf("layer timing filter does not contain %q: %s", fragment, filter)
		}
	}
}

func TestBuildFilterUsesSilenceForTimelineVideoWithoutAudio(t *testing.T) {
	config := composition.Config{
		Version: composition.CurrentVersion,
		Canvas:  composition.Canvas{Width: 1080, Height: 1920, FPS: 30, Background: "#000000"},
		Timeline: &composition.Timeline{Segments: []composition.Segment{
			{ID: "clip", Source: "clip", Start: 0, End: 2},
			{ID: "silent-ad", Source: "asset", AssetID: "asset-1", Start: 0, End: 3},
		}},
		Layers: []composition.Layer{{ID: "video", Type: "video", Source: "clip", Width: 1080, Height: 1920, Visible: true, Opacity: 1}},
	}
	filter, _, _, err := buildFilter(config, "", nil, map[string]int{"asset-1": 1}, map[int]bool{0: true, 1: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filter, "anullsrc=r=48000:cl=stereo,atrim=duration=3") {
		t.Fatalf("silent timeline asset did not receive a silence track: %s", filter)
	}
}

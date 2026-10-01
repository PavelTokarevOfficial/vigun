package processing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
)

type memoryStorage struct{ objects map[string][]byte }

func newMemoryStorage() *memoryStorage { return &memoryStorage{objects: map[string][]byte{}} }
func (s *memoryStorage) Put(_ context.Context, key string, body io.Reader, _ string) error {
	b, err := io.ReadAll(body)
	if err == nil {
		s.objects[key] = b
	}
	return err
}
func (s *memoryStorage) Get(_ context.Context, key string) (media.Object, error) {
	b, ok := s.objects[key]
	if !ok {
		return media.Object{}, os.ErrNotExist
	}
	return media.Object{Key: key, Size: int64(len(b)), Body: io.NopCloser(bytes.NewReader(b))}, nil
}
func (s *memoryStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}
func (s *memoryStorage) Exists(_ context.Context, key string) (bool, error) {
	_, ok := s.objects[key]
	return ok, nil
}
func (s *memoryStorage) PresignGet(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

type fakeDownloader struct{ calls int }

func (d *fakeDownloader) Download(_ context.Context, _ string) (io.ReadCloser, string, error) {
	d.calls++
	return io.NopCloser(bytes.NewReader([]byte("source"))), "video/mp4", nil
}

type fakeMedia struct {
	audioCalls, cutCalls, renderCalls int
	lastRenderInput                   RenderInput
}

func (m *fakeMedia) ExtractAudio(_ context.Context, _, out string) error {
	m.audioCalls++
	return os.WriteFile(out, []byte("audio"), 0o600)
}
func (m *fakeMedia) Cut(_ context.Context, _, out string, _ composition.Timeline, _ string) error {
	m.cutCalls++
	return os.WriteFile(out, []byte("fragment"), 0o600)
}
func (m *fakeMedia) Render(_ context.Context, in RenderInput) error {
	m.renderCalls++
	m.lastRenderInput = in
	return os.WriteFile(in.OutputPath, []byte("render"), 0o600)
}

type fakeTranscriber struct {
	calls int
	empty bool
}

func (t *fakeTranscriber) Transcribe(_ context.Context, _, outputBase, _ string) error {
	t.calls++
	if t.empty {
		return os.WriteFile(outputBase+".srt", nil, 0o600)
	}
	return os.WriteFile(outputBase+".srt", []byte("1\n00:00:00,000 --> 00:00:01,000\ntext\n"), 0o600)
}

func TestProcessSkipsExistingArtifactsOnRetry(t *testing.T) {
	store, downloader, processor, transcriber := newMemoryStorage(), &fakeDownloader{}, &fakeMedia{}, &fakeTranscriber{}
	runner := Runner{Storage: store, Downloader: downloader, Media: processor, Transcriber: transcriber}
	in := Input{ClipID: "clip-1", ClipURL: "https://example.test/clip", Width: 1080, Height: 1920, Blur: 25, Preset: "veryfast"}

	first, err := runner.Process(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.objects[first.RenderKey]; !ok {
		t.Fatalf("render %q was not stored", first.RenderKey)
	}
	if _, ok := store.objects[first.SubtitleKey]; !ok {
		t.Fatalf("rendered subtitles %q were not stored", first.SubtitleKey)
	}
	if downloader.calls != 1 || processor.audioCalls != 1 || transcriber.calls != 1 || processor.renderCalls != 1 {
		t.Fatalf("unexpected first-run calls: download=%d audio=%d transcribe=%d render=%d", downloader.calls, processor.audioCalls, transcriber.calls, processor.renderCalls)
	}

	second, err := runner.Process(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("artifact keys changed between retries: %#v != %#v", first, second)
	}
	if downloader.calls != 1 || processor.audioCalls != 1 || transcriber.calls != 1 || processor.renderCalls != 1 {
		t.Fatalf("retry repeated completed work: download=%d audio=%d transcribe=%d render=%d", downloader.calls, processor.audioCalls, transcriber.calls, processor.renderCalls)
	}

}

func TestProcessRendersWithoutSubtitlesWhenTranscriptionIsEmpty(t *testing.T) {
	store, downloader, processor := newMemoryStorage(), &fakeDownloader{}, &fakeMedia{}
	transcriber := &fakeTranscriber{empty: true}
	runner := Runner{Storage: store, Downloader: downloader, Media: processor, Transcriber: transcriber}

	result, err := runner.Process(context.Background(), Input{ClipID: "silent-clip", ClipURL: "https://example.test/clip", Width: 1080, Height: 1920, Blur: 25, Preset: "veryfast"})
	if err != nil {
		t.Fatal(err)
	}
	if processor.renderCalls != 1 {
		t.Fatalf("expected a render without subtitles, got %d render calls", processor.renderCalls)
	}
	if processor.lastRenderInput.SubtitlePath != "" {
		t.Fatalf("expected an empty subtitle path, got %q", processor.lastRenderInput.SubtitlePath)
	}
	if _, ok := store.objects[result.RenderKey]; !ok {
		t.Fatalf("render %q was not stored", result.RenderKey)
	}
}

func TestProcessSkipsAudioAndWhisperWithoutSubtitleLayer(t *testing.T) {
	store, downloader, processor, transcriber := newMemoryStorage(), &fakeDownloader{}, &fakeMedia{}, &fakeTranscriber{}
	runner := Runner{Storage: store, Downloader: downloader, Media: processor, Transcriber: transcriber}
	steps := []string{}
	config := composition.Default(1080, 1920, 25)
	layers := make([]composition.Layer, 0, len(config.Layers))
	for _, layer := range config.Layers {
		if layer.Type != "subtitles" {
			layers = append(layers, layer)
		}
	}
	config.Layers = layers
	snapshot, err := json.Marshal(composition.Snapshot{Version: composition.CurrentVersion, TemplateID: "without-subtitles", TemplateName: "Without subtitles", Config: config})
	if err != nil {
		t.Fatal(err)
	}

	result, err := runner.Process(context.Background(), Input{ClipID: "no-subtitles", ClipURL: "https://example.test/clip", Width: 1080, Height: 1920, Blur: 25, Preset: "veryfast", TemplateSnapshot: snapshot, Progress: func(step, _ string, _ int) error {
		steps = append(steps, step)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if processor.audioCalls != 0 || transcriber.calls != 0 {
		t.Fatalf("audio or Whisper ran without a subtitle layer: audio=%d transcribe=%d", processor.audioCalls, transcriber.calls)
	}
	if result.AudioKey != "" || result.SubtitleKey != "" {
		t.Fatalf("unexpected subtitle artifacts: audio=%q subtitle=%q", result.AudioKey, result.SubtitleKey)
	}
	if processor.renderCalls != 1 || processor.lastRenderInput.SubtitlePath != "" {
		t.Fatalf("expected one render without subtitles, calls=%d path=%q", processor.renderCalls, processor.lastRenderInput.SubtitlePath)
	}
	for _, step := range steps {
		if step == "extracting_audio" || step == "transcribing" {
			t.Fatalf("unexpected subtitle step %q in %#v", step, steps)
		}
	}
	if !containsStep(steps, "rendering_without_subtitles") {
		t.Fatalf("missing explicit render status without subtitles: %#v", steps)
	}
}

func containsStep(steps []string, expected string) bool {
	for _, step := range steps {
		if step == expected {
			return true
		}
	}
	return false
}

func TestCreateFragmentCopiesUnchangedSourceWithoutFFmpeg(t *testing.T) {
	store := newMemoryStorage()
	store.objects["assets/source.mp4"] = []byte("original-video")
	processor := &fakeMedia{}
	runner := Runner{Storage: store, Media: processor}
	steps := []string{}

	result, err := runner.CreateFragment(context.Background(), Input{ClipID: "fragment-copy", SourceKey: "assets/source.mp4", Preset: "veryfast", Progress: func(step, _ string, _ int) error {
		steps = append(steps, step)
		return nil
	}}, composition.Timeline{Segments: []composition.Segment{{ID: "full", Source: "clip", Start: 0, End: 42, SourceDuration: 42}}})
	if err != nil {
		t.Fatal(err)
	}
	if processor.cutCalls != 0 {
		t.Fatalf("FFmpeg cut was called for an unchanged source")
	}
	if string(store.objects[result.SourceKey]) != "original-video" {
		t.Fatalf("fragment copy does not match source")
	}
	if !containsStep(steps, "copying_fragment") {
		t.Fatalf("missing copy progress step: %#v", steps)
	}
}

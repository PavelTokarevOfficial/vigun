package processing

import (
	"context"
	"fmt"
	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/media"
	"io"
	"os"
	"path/filepath"
)

// Runner orchestrates steps; it does not know Twitch, Rod, S3 or CLI details.
type Runner struct {
	Storage     media.Storage
	Downloader  Downloader
	Media       MediaProcessor
	Transcriber Transcriber
}
type Input struct {
	JobID, ClipID, ClipURL string
	SourceKey              string
	Width, Height, Blur    int
	Preset                 string
	TemplateSnapshot       []byte
	WhisperModel           string
	Progress               func(step, clipStatus string, percent int) error
}
type Result struct{ SourceKey, AudioKey, SubtitleKey, RenderKey string }

func (r *Runner) CreateFragment(ctx context.Context, in Input, timeline composition.Timeline) (Result, error) {
	key := "sources/" + in.ClipID + "/source.mp4"
	result := Result{SourceKey: key}
	if exists, err := r.Storage.Exists(ctx, key); err != nil {
		return result, err
	} else if exists {
		return result, nil
	}
	if in.SourceKey == "" {
		return result, fmt.Errorf("fragment source is missing")
	}
	if isUnchangedFragment(timeline) {
		if err := report(in, "copying_fragment", "rendering", 50); err != nil {
			return result, err
		}
		object, err := r.Storage.Get(ctx, in.SourceKey)
		if err != nil {
			return result, fmt.Errorf("open fragment source: %w", err)
		}
		defer object.Body.Close()
		contentType := object.ContentType
		if contentType == "" {
			contentType = "video/mp4"
		}
		if err = r.Storage.Put(ctx, key, object.Body, contentType); err != nil {
			return result, fmt.Errorf("copy unchanged fragment: %w", err)
		}
		return result, nil
	}
	dir, err := os.MkdirTemp("", "finde-fragment-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)
	if err = report(in, "preparing_fragment_source", "rendering", 10); err != nil {
		return Result{}, err
	}
	source := filepath.Join(dir, "source")
	if err = r.ensureLocal(ctx, in.SourceKey, source); err != nil {
		return Result{}, err
	}
	if err = report(in, "rendering_fragment", "rendering", 35); err != nil {
		return Result{}, err
	}
	output := filepath.Join(dir, "fragment.mp4")
	if err = r.Media.Cut(ctx, source, output, timeline, in.Preset); err != nil {
		return Result{}, fmt.Errorf("render fragment: %w", err)
	}
	if err = report(in, "saving_fragment", "rendering", 90); err != nil {
		return Result{}, err
	}
	if err = r.putFile(ctx, key, output, "video/mp4"); err != nil {
		return Result{}, err
	}
	return Result{SourceKey: key}, nil
}

func isUnchangedFragment(timeline composition.Timeline) bool {
	if len(timeline.Segments) != 1 {
		return false
	}
	segment := timeline.Segments[0]
	return segment.Source != "asset" && segment.Start <= 0.001 && segment.SourceDuration > 0 && segment.End >= segment.SourceDuration-0.05
}

func (r *Runner) Process(ctx context.Context, in Input) (Result, error) {
	d, e := os.MkdirTemp("", "finde-job-")
	if e != nil {
		return Result{}, e
	}
	defer os.RemoveAll(d)
	renderName := in.JobID
	if renderName == "" {
		renderName = "vertical"
	}
	transcriptKey := "subtitles/" + in.ClipID + "/subtitles.srt"
	if in.WhisperModel != "" && in.WhisperModel != "tiny" {
		transcriptKey = "subtitles/" + in.ClipID + "/" + in.WhisperModel + ".srt"
	}
	out := Result{SourceKey: "sources/" + in.ClipID + "/source.mp4", AudioKey: "audio/" + in.ClipID + "/audio.wav", SubtitleKey: "subtitles/" + in.ClipID + "/renders/" + renderName + ".srt", RenderKey: "renders/" + in.ClipID + "/" + renderName + ".mp4"}
	requiresSubtitles, e := requiresSubtitleLayer(in.TemplateSnapshot, in.Width, in.Height, in.Blur)
	if e != nil {
		return out, e
	}
	if !requiresSubtitles {
		out.AudioKey = ""
		out.SubtitleKey = ""
	}
	src := filepath.Join(d, "source.mp4")
	sourceKey := out.SourceKey
	if in.SourceKey != "" {
		sourceKey = in.SourceKey
	}
	if e = report(in, "preparing_source", "downloaded", 5); e != nil {
		return out, e
	}
	if e = r.ensureSource(ctx, in.ClipURL, sourceKey, src); e != nil {
		return out, e
	}
	audio := filepath.Join(d, "audio.wav")
	if requiresSubtitles {
		if ok, e := r.Storage.Exists(ctx, out.AudioKey); e != nil {
			return out, e
		} else if !ok {
			if e = report(in, "extracting_audio", "transcribing", 15); e != nil {
				return out, e
			}
			if e = r.Media.ExtractAudio(ctx, src, audio); e != nil {
				return out, fmt.Errorf("extract audio: %w", e)
			}
			if e = r.putFile(ctx, out.AudioKey, audio, "audio/wav"); e != nil {
				return out, e
			}
		}
		if ok, e := r.Storage.Exists(ctx, transcriptKey); e != nil {
			return out, e
		} else if !ok {
			if e = report(in, "transcribing", "transcribing", 30); e != nil {
				return out, e
			}
			if e = r.ensureLocal(ctx, out.AudioKey, audio); e != nil {
				return out, e
			}
			base := filepath.Join(d, "subtitles")
			if e = r.Transcriber.Transcribe(ctx, audio, base, in.WhisperModel); e != nil {
				return out, fmt.Errorf("transcribe: %w", e)
			}
			if e = r.putFile(ctx, transcriptKey, base+".srt", "application/x-subrip"); e != nil {
				return out, e
			}
		}
	}
	if ok, e := r.Storage.Exists(ctx, out.RenderKey); e != nil {
		return out, e
	} else if !ok {
		preparingStep := "preparing_render_without_subtitles"
		if requiresSubtitles {
			preparingStep = "preparing_render_with_subtitles"
		}
		if e = report(in, preparingStep, "rendering", 50); e != nil {
			return out, e
		}
		config, assets, e := r.templateFiles(ctx, in, d)
		if e != nil {
			return out, e
		}
		sub := filepath.Join(d, "subtitles.srt")
		hasSubtitles := false
		if requiresSubtitles {
			if e = r.ensureLocal(ctx, transcriptKey, sub); e != nil {
				return out, e
			}
			hasSubtitles, e = srtHasCues(sub)
			if e != nil {
				return out, fmt.Errorf("inspect subtitles: %w", e)
			}
			if hasSubtitles && config.Timeline != nil {
				if e = applyTimelineToSRT(sub, config.Timeline.Segments); e != nil {
					return out, fmt.Errorf("retime subtitles: %w", e)
				}
				hasSubtitles, e = srtHasCues(sub)
				if e != nil {
					return out, fmt.Errorf("inspect retimed subtitles: %w", e)
				}
			}
		}
		subtitlePaths := map[string]string{}
		subtitlePath := ""
		if hasSubtitles {
			subtitlePath = sub
			subtitlePaths, e = layerSubtitleFiles(sub, d, config)
			if e != nil {
				return out, fmt.Errorf("prepare subtitle tracks: %w", e)
			}
		}
		if requiresSubtitles {
			if e = r.putFile(ctx, out.SubtitleKey, sub, "application/x-subrip"); e != nil {
				return out, fmt.Errorf("save rendered subtitles: %w", e)
			}
		}
		renderingStep := "rendering_without_subtitles"
		if requiresSubtitles {
			renderingStep = "rendering_with_subtitles"
		}
		if e = report(in, renderingStep, "rendering", 60); e != nil {
			return out, e
		}
		render := filepath.Join(d, "final.mp4")
		if e = r.Media.Render(ctx, RenderInput{SourcePath: src, SubtitlePath: subtitlePath, SubtitlePaths: subtitlePaths, OutputPath: render, Width: in.Width, Height: in.Height, Blur: in.Blur, Preset: in.Preset, Composition: config, AssetPaths: assets}); e != nil {
			return out, fmt.Errorf("render: %w", e)
		}
		if e = report(in, "saving_render", "rendering", 90); e != nil {
			return out, e
		}
		if e = r.putFile(ctx, out.RenderKey, render, "video/mp4"); e != nil {
			return out, e
		}
	}
	return out, nil
}

func requiresSubtitleLayer(raw []byte, width, height, blur int) (bool, error) {
	config := composition.Default(width, height, blur)
	if len(raw) > 0 {
		snapshot, err := composition.ParseSnapshot(raw)
		if err != nil {
			return false, err
		}
		config = snapshot.Config
	}
	for _, layer := range config.Layers {
		if layer.Type == "subtitles" && layer.Visible {
			return true, nil
		}
	}
	return false, nil
}

// templateFiles resolves the immutable asset keys from the job snapshot, never
// from the mutable template record. The FFmpeg adapter only receives local paths.
func (r *Runner) templateFiles(ctx context.Context, in Input, dir string) (composition.Config, map[string]string, error) {
	config := composition.Default(in.Width, in.Height, in.Blur)
	if len(in.TemplateSnapshot) == 0 {
		return config, map[string]string{}, nil
	}
	snapshot, err := composition.ParseSnapshot(in.TemplateSnapshot)
	if err != nil {
		return composition.Config{}, nil, err
	}
	paths := make(map[string]string, len(snapshot.Assets))
	for index, asset := range snapshot.Assets {
		if asset.ID == "" || asset.StorageKey == "" {
			return composition.Config{}, nil, fmt.Errorf("template snapshot contains an invalid asset")
		}
		path := filepath.Join(dir, fmt.Sprintf("asset-%d", index))
		if err := r.ensureLocal(ctx, asset.StorageKey, path); err != nil {
			return composition.Config{}, nil, fmt.Errorf("download template asset %s: %w", asset.ID, err)
		}
		paths[asset.ID] = path
	}
	return snapshot.Config, paths, nil
}

func report(in Input, step, clipStatus string, percent int) error {
	if in.Progress == nil {
		return nil
	}
	return in.Progress(step, clipStatus, percent)
}
func (r *Runner) ensureSource(ctx context.Context, url, key, local string) error {
	ok, e := r.Storage.Exists(ctx, key)
	if e != nil {
		return e
	}
	if ok {
		return r.ensureLocal(ctx, key, local)
	}
	body, mime, e := r.Downloader.Download(ctx, url)
	if e != nil {
		return fmt.Errorf("download: %w", e)
	}
	defer body.Close()
	if mime == "" {
		mime = "video/mp4"
	}
	if e = r.Storage.Put(ctx, key, body, mime); e != nil {
		return e
	}
	return r.ensureLocal(ctx, key, local)
}
func (r *Runner) ensureLocal(ctx context.Context, key, path string) error {
	o, e := r.Storage.Get(ctx, key)
	if e != nil {
		return e
	}
	defer o.Body.Close()
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = io.Copy(f, o.Body)
	return e
}
func (r *Runner) putFile(ctx context.Context, key, path, mime string) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return r.Storage.Put(ctx, key, f, mime)
}

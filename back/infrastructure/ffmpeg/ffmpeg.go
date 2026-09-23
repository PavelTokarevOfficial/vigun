package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/finde-clip/finde-v2/back/internal/composition"
	"github.com/finde-clip/finde-v2/back/internal/processing"
)

type Adapter struct{ Bin string }

func New(bin string) *Adapter { return &Adapter{bin} }
func (a *Adapter) run(ctx context.Context, args ...string) error {
	out, e := exec.CommandContext(ctx, a.Bin, args...).CombinedOutput()
	if e != nil {
		return fmt.Errorf("ffmpeg: %w: %s", e, string(out))
	}
	return nil
}
func (a *Adapter) ExtractAudio(ctx context.Context, source, out string) error {
	return a.run(ctx, "-y", "-i", source, "-ar", "16000", "-ac", "1", out)
}
func (a *Adapter) Render(ctx context.Context, in processing.RenderInput) error {
	config := in.Composition
	if config.Version == 0 {
		config = composition.Default(in.Width, in.Height, in.Blur)
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid render composition: %w", err)
	}

	// Large vertical compositions may fan one source into several timeline and
	// blur branches. Limit filter and encoder parallelism so FFmpeg does not get
	// killed by a short-lived memory spike inside a constrained worker container.
	args := []string{"-y", "-filter_threads", "1", "-filter_complex_threads", "1", "-i", in.SourcePath}
	assetInputs, err := appendAssetInputs(&args, config, in.AssetPaths)
	if err != nil {
		return err
	}
	audioInputs, err := a.detectAudioInputs(ctx, in.SourcePath, in.AssetPaths, assetInputs)
	if err != nil {
		return err
	}
	if err := appendTimelineInputs(&args, config, in.SourcePath, in.AssetPaths, assetInputs, audioInputs); err != nil {
		return err
	}
	filter, videoLabel, audioLabel, err := buildFilter(config, in.SubtitlePath, in.SubtitlePaths, assetInputs, audioInputs)
	if err != nil {
		return err
	}
	args = append(args, "-filter_complex", filter, "-map", videoLabel)
	if audioLabel != "" {
		args = append(args, "-map", audioLabel)
	} else {
		args = append(args, "-map", "0:a?")
	}
	args = append(args, "-c:v", "libx264", "-threads", "2", "-pix_fmt", "yuv420p", "-preset", in.Preset, "-crf", "20", "-c:a", "aac", "-shortest", "-movflags", "+faststart", in.OutputPath)
	return a.run(ctx, args...)
}

// appendTimelineInputs gives every timeline segment its own seeked input. A
// single input fanned out into trim filters makes FFmpeg push all branches at
// once. When segments are reordered (for example 20s..23s followed by 0s..2s),
// concat then buffers the later branch as full-size raw frames. With multiple
// visual layers that can exhaust the worker's memory. Independent inputs are
// demand-driven by concat and start decoding at the requested segment.
func appendTimelineInputs(args *[]string, config composition.Config, sourcePath string, paths map[string]string, inputs map[string]int, audioInputs map[int]bool) error {
	if config.Timeline == nil {
		return nil
	}
	nextIndex := 1
	for _, index := range inputs {
		if index >= nextIndex {
			nextIndex = index + 1
		}
	}
	for _, segment := range config.Timeline.Segments {
		duration := segment.End - segment.Start
		if duration <= 0 {
			return fmt.Errorf("timeline segment %s has invalid duration", segment.ID)
		}
		path := sourcePath
		hasAudio := audioInputs[0]
		if segment.Source == "asset" {
			path = paths[segment.AssetID]
			assetIndex, ok := inputs[segment.AssetID]
			if !ok || path == "" {
				return fmt.Errorf("timeline segment %s has no resolved asset", segment.ID)
			}
			hasAudio = audioInputs[assetIndex]
		}
		*args = append(*args, "-ss", fmt.Sprintf("%g", segment.Start), "-t", fmt.Sprintf("%g", duration), "-i", path)
		inputs[timelineInputKey(segment.ID)] = nextIndex
		audioInputs[nextIndex] = hasAudio
		nextIndex++
	}
	return nil
}

func timelineInputKey(segmentID string) string { return "timeline:" + segmentID }

func (a *Adapter) detectAudioInputs(ctx context.Context, source string, paths map[string]string, assetInputs map[string]int) (map[int]bool, error) {
	inputs := map[int]bool{}
	hasAudio, err := a.hasAudioStream(ctx, source)
	if err != nil {
		return nil, err
	}
	inputs[0] = hasAudio
	for id, index := range assetInputs {
		hasAudio, err = a.hasAudioStream(ctx, paths[id])
		if err != nil {
			return nil, err
		}
		inputs[index] = hasAudio
	}
	return inputs, nil
}

func (a *Adapter) hasAudioStream(ctx context.Context, path string) (bool, error) {
	probe := "ffprobe"
	if dir := filepath.Dir(a.Bin); dir != "." {
		probe = filepath.Join(dir, "ffprobe")
	}
	out, err := exec.CommandContext(ctx, probe, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=index", "-of", "csv=p=0", path).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ffprobe audio stream: %w: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func appendAssetInputs(args *[]string, config composition.Config, paths map[string]string) (map[string]int, error) {
	indices := map[string]int{}
	nextIndex := 1 // source Twitch clip is input 0
	appendAsset := func(id string, loop bool) error {
		if id == "" || indices[id] != 0 {
			return nil
		}
		path := paths[id]
		if path == "" {
			return fmt.Errorf("render asset %s is missing from job snapshot", id)
		}
		if loop {
			*args = append(*args, "-stream_loop", "-1")
		}
		*args = append(*args, "-i", path)
		indices[id] = nextIndex
		nextIndex++
		return nil
	}
	for _, layer := range config.Layers {
		if !layer.Visible || layer.AssetID == "" {
			continue
		}
		if err := appendAsset(layer.AssetID, layer.Type == "image" || layer.Type == "gif"); err != nil {
			return nil, err
		}
	}
	if config.Timeline != nil {
		for _, segment := range config.Timeline.Segments {
			if segment.Source == "asset" {
				if err := appendAsset(segment.AssetID, false); err != nil {
					return nil, err
				}
			}
		}
	}
	return indices, nil
}

func buildFilter(config composition.Config, subtitlePath string, subtitlePaths map[string]string, assetInputs map[string]int, audioInputs map[int]bool) (string, string, string, error) {
	baseFilter := fmt.Sprintf("color=c=%s:s=%dx%d:r=%d", safeColor(config.Canvas.Background, "#000000"), config.Canvas.Width, config.Canvas.Height, config.Canvas.FPS)
	if duration := timelineOutputDuration(config); duration > 0 {
		baseFilter += fmt.Sprintf(":d=%g", duration)
	}
	filters := []string{baseFilter + "[base0]"}
	timelineAudio, err := appendTimelineAudioFilters(&filters, config, assetInputs, audioInputs)
	if err != nil {
		return "", "", "", err
	}
	base := "base0"
	step := 0
	audioLabels := []string{}
	for _, layer := range config.Layers {
		if !layer.Visible {
			continue
		}
		if layer.Type == "subtitles" {
			layerPath, hasLayerPath := subtitlePaths[layer.ID]
			if (hasLayerPath && layerPath == "") || (!hasLayerPath && subtitlePath == "") {
				continue
			}
		}
		if layer.Type == "audio" {
			index, ok := assetInputs[layer.AssetID]
			if !ok {
				return "", "", "", fmt.Errorf("audio layer %s has no resolved asset", layer.ID)
			}
			if !audioInputs[index] {
				return "", "", "", fmt.Errorf("audio layer %s asset has no audio stream", layer.ID)
			}
			label := fmt.Sprintf("audio%d", len(audioLabels))
			filter := fmt.Sprintf("[%d:a]volume=1", index)
			if layer.StartTime > 0 || layer.EndTime > 0 {
				duration := layer.EndTime - layer.StartTime
				if layer.EndTime == 0 {
					duration = maxFloat(0.1, timelineOutputDuration(config)-layer.StartTime)
				}
				filter += fmt.Sprintf(",atrim=duration=%g,asetpts=PTS-STARTPTS+%g/TB", duration, layer.StartTime)
			}
			filters = append(filters, filter+"["+label+"]")
			audioLabels = append(audioLabels, "["+label+"]")
			continue
		}
		next := fmt.Sprintf("base%d", step+1)
		switch layer.Type {
		case "subtitles":
			layerSubtitlePath := subtitlePath
			if path, ok := subtitlePaths[layer.ID]; ok {
				layerSubtitlePath = path
			}
			style := layer.Style
			fontSize := positiveOr(style.FontSize, 8)
			outline := nonNegativeOr(style.Outline, 2)
			primary := safeASSColor(style.PrimaryColor, "&H00FFFFFF")
			outlineColor := safeASSColor(style.OutlineColor, "&H00000000")
			// This is the point where Whisper's local SRT is burned into the render.
			filters = append(filters, fmt.Sprintf("[%s]subtitles=filename='%s':force_style='Fontsize=%d,PrimaryColour=%s,OutlineColour=%s,BorderStyle=1,Outline=%d'[%s]", base, escapeFilterPath(layerSubtitlePath), fontSize, primary, outlineColor, outline, next))
		case "text":
			fontSize := positiveOr(layer.Style.FontSize, max(18, layer.Height/5))
			filters = append(filters, fmt.Sprintf("[%s]drawtext=text='%s':x=%d:y=%d:fontsize=%d:fontcolor=white:borderw=%d:bordercolor=black%s[%s]", base, escapeDrawText(layer.Text), layer.X, layer.Y, fontSize, nonNegativeOr(layer.Style.Outline, 2), filterEnable(layer), next))
		case "color":
			visual := fmt.Sprintf("layer%d", step)
			filters = append(filters, fmt.Sprintf("color=c=%s:s=%dx%d:r=%d[%s]", safeColor(layer.Color, "#000000"), videoDimension(layer.Width), videoDimension(layer.Height), config.Canvas.FPS, visual))
			filters = append(filters, overlayFilter(base, visual, next, layer))
		case "blur":
			baseCopy := fmt.Sprintf("basecopy%d", step)
			blurSource := fmt.Sprintf("blursource%d", step)
			visual := fmt.Sprintf("layer%d", step)
			width := min(videoDimension(layer.Width), max(2, config.Canvas.Width-max(0, layer.X)))
			height := min(videoDimension(layer.Height), max(2, config.Canvas.Height-max(0, layer.Y)))
			filters = append(filters, fmt.Sprintf("[%s]split=2[%s][%s]", base, baseCopy, blurSource))
			blur := fmt.Sprintf("[%s]crop=%d:%d:%d:%d", blurSource, width, height, max(0, layer.X), max(0, layer.Y))
			if layer.Filters.Blur > 0 {
				blur += fmt.Sprintf(",boxblur=%d:10", layer.Filters.Blur)
			}
			if layer.Filters.Brightness != 0 {
				blur += fmt.Sprintf(",eq=brightness=%g", layer.Filters.Brightness)
			}
			if layer.Opacity < 1 {
				blur += fmt.Sprintf(",format=rgba,colorchannelmixer=aa=%g", layer.Opacity)
			}
			filters = append(filters, blur+"["+visual+"]")
			filters = append(filters, overlayFilter(baseCopy, visual, next, layer))
		default:
			input := "[0:v]"
			usesClip := layer.Type == "input_video" || (layer.Type == "video" && layer.Source == "clip")
			visual := fmt.Sprintf("layer%d", step)
			effectiveLayer := layer
			if usesClip && config.Timeline != nil && len(config.Timeline.Segments) > 0 {
				visual, err = appendTimelineVideoFilters(&filters, config, layer, step, assetInputs)
				if err != nil {
					return "", "", "", err
				}
			} else {
				if !usesClip {
					index, ok := assetInputs[layer.AssetID]
					if !ok {
						return "", "", "", fmt.Errorf("layer %s has no resolved asset", layer.ID)
					}
					input = fmt.Sprintf("[%d:v]", index)
				}
				filter := input
				if !usesClip && layer.TimelineSegmentID != "" {
					segment, outputStart, ok := timelineSegmentLayout(config, layer.TimelineSegmentID)
					if !ok || (segment.Source != "asset" && segment.Source != "") {
						return "", "", "", fmt.Errorf("layer %s has no linked asset timeline segment", layer.ID)
					}
					filter += fmt.Sprintf("trim=start=%g:end=%g,setpts=PTS-STARTPTS+%g/TB,%s", segment.Start, segment.End, outputStart, scaleFilter(layer))
					if effectiveLayer.EndTime == 0 {
						effectiveLayer.StartTime = outputStart
						effectiveLayer.EndTime = outputStart + segment.End - segment.Start
					}
				} else {
					filter += scaleFilter(layer)
					if !usesClip && layer.StartTime > 0 {
						filter += fmt.Sprintf(",setpts=PTS-STARTPTS+%g/TB", layer.StartTime)
					}
				}
				filters = append(filters, filter+"["+visual+"]")
			}
			filters = append(filters, overlayFilter(base, visual, next, effectiveLayer))
		}
		base = next
		step++
	}
	audio := ""
	if len(audioLabels) > 0 {
		audio = "[audioout]"
		baseAudio := "[0:a]"
		if timelineAudio != "" {
			baseAudio = timelineAudio
		}
		filters = append(filters, baseAudio+strings.Join(audioLabels, "")+fmt.Sprintf("amix=inputs=%d:duration=first:dropout_transition=0[audioout]", len(audioLabels)+1))
	} else if timelineAudio != "" {
		audio = timelineAudio
	}
	return strings.Join(filters, ";"), "[" + base + "]", audio, nil
}

func appendTimelineVideoFilters(filters *[]string, config composition.Config, layer composition.Layer, step int, assetInputs map[string]int) (string, error) {
	inputs := ""
	for index, segment := range config.Timeline.Segments {
		label := fmt.Sprintf("clipv%d_%d", step, index)
		if (segment.Source == "asset") && timelineSegmentHasVisualLayer(config, segment.ID) {
			*filters = append(*filters, fmt.Sprintf("color=c=black@0:s=%dx%d:r=%d:d=%g,format=rgba[%s]", videoDimension(layer.Width), videoDimension(layer.Height), config.Canvas.FPS, segment.End-segment.Start, label))
		} else {
			input, err := timelineSegmentInput(segment, assetInputs, "v")
			if err != nil {
				return "", err
			}
			trim := fmt.Sprintf("trim=start=%g:end=%g,", segment.Start, segment.End)
			if _, dedicated := assetInputs[timelineInputKey(segment.ID)]; dedicated {
				trim = ""
			}
			pixelFormat := "yuv420p"
			if layer.Opacity < 1 {
				pixelFormat = "rgba"
			}
			*filters = append(*filters, fmt.Sprintf("%s%ssetpts=PTS-STARTPTS,%s,fps=%d,setsar=1,format=%s[%s]", input, trim, scaleFilter(layer), config.Canvas.FPS, pixelFormat, label))
		}
		inputs += "[" + label + "]"
	}
	output := fmt.Sprintf("clipv%d", step)
	*filters = append(*filters, inputs+fmt.Sprintf("concat=n=%d:v=1:a=0[%s]", len(config.Timeline.Segments), output))
	return output, nil
}

func timelineSegmentHasVisualLayer(config composition.Config, segmentID string) bool {
	for _, layer := range config.Layers {
		if layer.TimelineSegmentID == segmentID {
			return true
		}
	}
	return false
}

func timelineSegmentLayout(config composition.Config, segmentID string) (composition.Segment, float64, bool) {
	if config.Timeline == nil {
		return composition.Segment{}, 0, false
	}
	outputStart := float64(0)
	for _, segment := range config.Timeline.Segments {
		if segment.ID == segmentID {
			return segment, outputStart, true
		}
		outputStart += maxFloat(0, segment.End-segment.Start)
	}
	return composition.Segment{}, 0, false
}

func appendTimelineAudioFilters(filters *[]string, config composition.Config, assetInputs map[string]int, audioInputs map[int]bool) (string, error) {
	if config.Timeline == nil || len(config.Timeline.Segments) == 0 {
		return "", nil
	}
	inputs := ""
	for index, segment := range config.Timeline.Segments {
		inputIndex, err := timelineSegmentIndex(segment, assetInputs)
		if err != nil {
			return "", err
		}
		label := fmt.Sprintf("clipa%d", index)
		if audioInputs[inputIndex] {
			trim := fmt.Sprintf("atrim=start=%g:end=%g,", segment.Start, segment.End)
			if _, dedicated := assetInputs[timelineInputKey(segment.ID)]; dedicated {
				trim = ""
			}
			*filters = append(*filters, fmt.Sprintf("[%d:a]%sasetpts=PTS-STARTPTS,aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo[%s]", inputIndex, trim, label))
		} else {
			*filters = append(*filters, fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%g,asetpts=PTS-STARTPTS[%s]", segment.End-segment.Start, label))
		}
		inputs += "[" + label + "]"
	}
	*filters = append(*filters, inputs+fmt.Sprintf("concat=n=%d:v=0:a=1[clipaudio]", len(config.Timeline.Segments)))
	return "[clipaudio]", nil
}

func timelineSegmentIndex(segment composition.Segment, assetInputs map[string]int) (int, error) {
	if index, ok := assetInputs[timelineInputKey(segment.ID)]; ok {
		return index, nil
	}
	if segment.Source == "" || segment.Source == "clip" {
		return 0, nil
	}
	index, ok := assetInputs[segment.AssetID]
	if !ok {
		return 0, fmt.Errorf("timeline segment %s has no resolved asset", segment.ID)
	}
	return index, nil
}

func timelineSegmentInput(segment composition.Segment, assetInputs map[string]int, stream string) (string, error) {
	index, err := timelineSegmentIndex(segment, assetInputs)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("[%d:%s]", index, stream), nil
}

func timelineOutputDuration(config composition.Config) float64 {
	if config.Timeline == nil {
		return 0
	}
	duration := float64(0)
	for _, segment := range config.Timeline.Segments {
		duration += maxFloat(0, segment.End-segment.Start)
	}
	return duration
}

func scaleFilter(layer composition.Layer) string {
	width, height := videoDimension(layer.Width), videoDimension(layer.Height)
	fit := layer.Fit
	if fit == "" {
		fit = "contain"
	}
	base := "scale="
	switch fit {
	case "cover":
		base += fmt.Sprintf("%d:%d:force_original_aspect_ratio=increase:force_divisible_by=2,crop=%d:%d", width, height, width, height)
	case "stretch":
		base += fmt.Sprintf("%d:%d", width, height)
	default:
		if layer.Type == "image" || layer.Type == "gif" {
			base = fmt.Sprintf("format=rgba,scale=%d:%d:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black@0", width, height, width, height)
		} else {
			base += fmt.Sprintf("%d:%d:force_original_aspect_ratio=decrease:force_divisible_by=2,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=black", width, height, width, height)
		}
	}
	if layer.Filters.Blur > 0 {
		base += fmt.Sprintf(",boxblur=%d:10", layer.Filters.Blur)
	}
	if layer.Filters.Brightness != 0 {
		base += fmt.Sprintf(",eq=brightness=%g", layer.Filters.Brightness)
	}
	if layer.Opacity < 1 {
		base += fmt.Sprintf(",format=rgba,colorchannelmixer=aa=%g", layer.Opacity)
	}
	return base
}

// H.264 and the scale/pad chain are most reliable with even intermediate
// dimensions. The editor may contain an odd user-entered value (e.g. 607);
// rendering it as 606 avoids FFmpeg rounding it up past the pad target.
func videoDimension(value int) int {
	if value < 2 {
		return 2
	}
	if value%2 != 0 {
		return value - 1
	}
	return value
}

func overlayFilter(base, visual, next string, layer composition.Layer) string {
	return fmt.Sprintf("[%s][%s]overlay=%d:%d:eof_action=pass:shortest=0%s[%s]", base, visual, layer.X, layer.Y, filterEnable(layer), next)
}

func filterEnable(layer composition.Layer) string {
	if layer.EndTime > 0 {
		return fmt.Sprintf(":enable='between(t,%g,%g)'", layer.StartTime, layer.EndTime)
	}
	if layer.StartTime > 0 {
		return fmt.Sprintf(":enable='gte(t,%g)'", layer.StartTime)
	}
	return ""
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func positiveOr(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
func nonNegativeOr(value, fallback int) int {
	if value >= 0 {
		return value
	}
	return fallback
}
func safeColor(value, fallback string) string {
	if strings.HasPrefix(value, "#") && len(value) == 7 {
		return value
	}
	return fallback
}
func safeASSColor(value, fallback string) string {
	if strings.HasPrefix(value, "&H") && len(value) == 10 {
		return value
	}
	return fallback
}
func escapeDrawText(value string) string {
	return strings.NewReplacer("\\", "\\\\", "'", "\\'", ":", "\\:", "\n", "\\n").Replace(value)
}

func escapeFilterPath(path string) string {
	return strings.NewReplacer("\\", "\\\\", ":", "\\:", "'", "\\'").Replace(path)
}

// Package composition owns the serializable video-template contract.
// It deliberately has no dependency on HTTP, PostgreSQL, S3 or FFmpeg.
package composition

import (
	"encoding/json"
	"fmt"
	"strings"
)

const CurrentVersion = 1

type Config struct {
	Version  int       `json:"version"`
	Canvas   Canvas    `json:"canvas"`
	Layers   []Layer   `json:"layers"`
	Timeline *Timeline `json:"timeline,omitempty"`
}

type Timeline struct {
	Segments []Segment `json:"segments"`
}

type Segment struct {
	ID    string  `json:"id"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Canvas struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FPS        int    `json:"fps"`
	Background string `json:"background"`
}

type Layer struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Visible    bool    `json:"visible"`
	Opacity    float64 `json:"opacity"`
	Source     string  `json:"source,omitempty"`
	Fit        string  `json:"fit,omitempty"`
	AssetID    string  `json:"assetId,omitempty"`
	Text       string  `json:"text,omitempty"`
	TextSource string  `json:"textSource,omitempty"`
	Color      string  `json:"color,omitempty"`
	Filters    Filters `json:"filters,omitempty"`
	Style      Style   `json:"style,omitempty"`
}

type Filters struct {
	Blur       int     `json:"blur,omitempty"`
	Brightness float64 `json:"brightness,omitempty"`
}

type Style struct {
	FontSize     int    `json:"fontSize,omitempty"`
	Alignment    int    `json:"alignment,omitempty"`
	MarginV      int    `json:"marginV,omitempty"`
	Outline      int    `json:"outline,omitempty"`
	PrimaryColor string `json:"primaryColor,omitempty"`
	OutlineColor string `json:"outlineColor,omitempty"`
}

type AssetSnapshot struct {
	ID         string `json:"id"`
	StorageKey string `json:"storageKey"`
	Kind       string `json:"kind"`
	MIMEType   string `json:"mimeType"`
}

// Snapshot is copied into a processing job. It is intentionally self-contained
// so a later template edit or asset rename cannot alter an already queued render.
type Snapshot struct {
	Version      int             `json:"version"`
	TemplateID   string          `json:"templateId"`
	TemplateName string          `json:"templateName"`
	Config       Config          `json:"config"`
	Assets       []AssetSnapshot `json:"assets"`
}

func ParseConfig(raw []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return Config{}, fmt.Errorf("decode template config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func ParseSnapshot(raw []byte) (Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode template snapshot: %w", err)
	}
	if snapshot.Version != CurrentVersion {
		return Snapshot{}, fmt.Errorf("unsupported template snapshot version %d", snapshot.Version)
	}
	if strings.TrimSpace(snapshot.TemplateID) == "" {
		return Snapshot{}, fmt.Errorf("template snapshot has no template id")
	}
	if err := snapshot.Config.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (c Config) JSON() ([]byte, error) { return json.Marshal(c) }

func (c Config) AssetIDs() []string {
	ids := make([]string, 0)
	seen := map[string]bool{}
	for _, layer := range c.Layers {
		if layer.AssetID != "" && !seen[layer.AssetID] {
			seen[layer.AssetID] = true
			ids = append(ids, layer.AssetID)
		}
	}
	return ids
}

func (c Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported template version %d", c.Version)
	}
	if c.Canvas.Width < 16 || c.Canvas.Width > 4320 || c.Canvas.Height < 16 || c.Canvas.Height > 7680 {
		return fmt.Errorf("canvas width and height must be between 16 and 7680")
	}
	if c.Canvas.FPS < 1 || c.Canvas.FPS > 120 {
		return fmt.Errorf("canvas fps must be between 1 and 120")
	}
	if len(c.Layers) == 0 || len(c.Layers) > 50 {
		return fmt.Errorf("template must contain from 1 to 50 layers")
	}
	if c.Timeline != nil {
		if len(c.Timeline.Segments) == 0 || len(c.Timeline.Segments) > 100 {
			return fmt.Errorf("timeline must contain from 1 to 100 segments")
		}
		previousEnd := float64(-1)
		for index, segment := range c.Timeline.Segments {
			if strings.TrimSpace(segment.ID) == "" || segment.Start < 0 || segment.End <= segment.Start {
				return fmt.Errorf("timeline segment %d is invalid", index+1)
			}
			if segment.Start < previousEnd {
				return fmt.Errorf("timeline segments must be ordered and not overlap")
			}
			previousEnd = segment.End
		}
	}
	ids := map[string]bool{}
	for i, layer := range c.Layers {
		if strings.TrimSpace(layer.ID) == "" || ids[layer.ID] {
			return fmt.Errorf("layer %d must have a unique id", i+1)
		}
		ids[layer.ID] = true
		if !validLayerType(layer.Type) {
			return fmt.Errorf("layer %q has unsupported type %q", layer.ID, layer.Type)
		}
		if layer.Opacity < 0 || layer.Opacity > 1 {
			return fmt.Errorf("layer %q opacity must be between 0 and 1", layer.ID)
		}
		if layer.Type != "audio" && (layer.Width < 1 || layer.Height < 1) {
			return fmt.Errorf("visual layer %q must have a positive size", layer.ID)
		}
		if layer.Type == "asset_video" || layer.Type == "image" || layer.Type == "gif" || layer.Type == "audio" || (layer.Type == "video" && layer.Source == "asset") {
			if strings.TrimSpace(layer.AssetID) == "" {
				return fmt.Errorf("layer %q requires assetId", layer.ID)
			}
		}
		if layer.Type == "video" && layer.Source != "clip" && layer.Source != "asset" {
			return fmt.Errorf("video layer %q has unsupported source %q", layer.ID, layer.Source)
		}
		if layer.Type == "text" && layer.TextSource != "streamer_name" && strings.TrimSpace(layer.Text) == "" {
			return fmt.Errorf("text layer %q cannot be empty", layer.ID)
		}
		if layer.Type == "text" && layer.TextSource != "" && layer.TextSource != "custom" && layer.TextSource != "streamer_name" {
			return fmt.Errorf("text layer %q has unsupported text source %q", layer.ID, layer.TextSource)
		}
		if layer.Type == "color" && strings.TrimSpace(layer.Color) == "" {
			return fmt.Errorf("color layer %q requires color", layer.ID)
		}
		if layer.Fit != "" && layer.Fit != "cover" && layer.Fit != "contain" && layer.Fit != "stretch" {
			return fmt.Errorf("layer %q has unsupported fit %q", layer.ID, layer.Fit)
		}
	}
	return nil
}

func validLayerType(v string) bool {
	switch v {
	case "video", "image", "subtitles", "text", "blur", "input_video", "asset_video", "gif", "audio", "color":
		return true
	default:
		return false
	}
}

func Default(width, height, blur int) Config {
	return Config{
		Version: CurrentVersion,
		Canvas:  Canvas{Width: width, Height: height, FPS: 30, Background: "#000000"},
		Layers: []Layer{
			{ID: "background", Name: "Видео на фоне", Type: "video", Source: "clip", Width: width, Height: height, Visible: true, Opacity: 1, Fit: "cover"},
			{ID: "background-blur", Name: "Блюр фона", Type: "blur", Width: width, Height: height, Visible: true, Opacity: 1, Filters: Filters{Blur: blur, Brightness: -0.2}},
			{ID: "clip", Name: "Видео", Type: "video", Source: "clip", Width: width, Height: height, Visible: true, Opacity: 1, Fit: "contain"},
			{ID: "subtitles", Name: "Субтитры", Type: "subtitles", X: 90, Y: height - 380, Width: width - 180, Height: 240, Visible: true, Opacity: 1, Style: Style{FontSize: 8, Alignment: 2, MarginV: 100, Outline: 2, PrimaryColor: "&H00FFFFFF", OutlineColor: "&H00000000"}},
		},
	}
}

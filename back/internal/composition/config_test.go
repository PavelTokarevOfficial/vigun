package composition

import "testing"

func TestValidateEditorLayerTypes(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Layers = append(config.Layers,
		Layer{ID: "asset-video", Type: "video", Source: "asset", AssetID: "asset-1", Width: 500, Height: 500, Visible: true, Opacity: 1},
		Layer{ID: "channel", Type: "text", TextSource: "streamer_name", Width: 500, Height: 100, Visible: true, Opacity: 1},
	)
	if err := config.Validate(); err != nil {
		t.Fatalf("new editor config should be valid: %v", err)
	}
}

func TestValidateAssetVideoRequiresAsset(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Layers = []Layer{{ID: "video", Type: "video", Source: "asset", Width: 500, Height: 500, Visible: true, Opacity: 1}}
	if err := config.Validate(); err == nil {
		t.Fatal("asset video without assetId should be rejected")
	}
}

func TestValidateTimelineRejectsOverlappingSegments(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Timeline = &Timeline{Segments: []Segment{
		{ID: "first", Start: 0, End: 5},
		{ID: "second", Start: 4, End: 8},
	}}
	if err := config.Validate(); err == nil {
		t.Fatal("overlapping timeline segments should be rejected")
	}
}

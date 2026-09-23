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

func TestValidateTimelineAllowsReorderedSourceRanges(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Timeline = &Timeline{Segments: []Segment{
		{ID: "first", Source: "clip", Start: 5, End: 8},
		{ID: "second", Source: "clip", Start: 0, End: 4},
	}}
	if err := config.Validate(); err != nil {
		t.Fatalf("reordered source ranges should be valid: %v", err)
	}
}

func TestValidateTimelineAssetAndCollectReference(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Timeline = &Timeline{Segments: []Segment{{ID: "ad", Source: "asset", AssetID: "asset-1", Start: 0, End: 5, SourceDuration: 5}}}
	if err := config.Validate(); err != nil {
		t.Fatalf("asset segment should be valid: %v", err)
	}
	ids := config.AssetIDs()
	if len(ids) != 1 || ids[0] != "asset-1" {
		t.Fatalf("timeline asset was not collected: %#v", ids)
	}
}

func TestTrainTransitionAssetsAreSnapshottedOnce(t *testing.T) {
	config := Default(1080, 1920, 25)
	config.Train = &Train{Enabled: true, TransitionAssetIDs: []string{"transition-1", "transition-2", "transition-1"}}
	if err := config.Validate(); err != nil {
		t.Fatalf("train config should be valid: %v", err)
	}
	ids := config.AssetIDs()
	if len(ids) != 2 || ids[0] != "transition-1" || ids[1] != "transition-2" {
		t.Fatalf("transition assets were not collected uniquely: %#v", ids)
	}
}

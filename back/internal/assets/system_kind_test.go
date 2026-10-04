package assets

import "testing"

func TestSystemDisplayKindUsesOriginForLegacyMIME(t *testing.T) {
	if got := systemDisplayKind("source", "binary/octet-stream"); got != "video" {
		t.Fatalf("source kind = %q, want video", got)
	}
	if got := systemDisplayKind("subtitle", "binary/octet-stream"); got != "file" {
		t.Fatalf("subtitle kind = %q, want file", got)
	}
}

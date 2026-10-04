package media

import "testing"

func TestSystemFolderID(t *testing.T) {
	tests := map[string]string{
		"audio":    "00000000-0000-0000-0000-000000000101",
		"subtitle": "00000000-0000-0000-0000-000000000102",
		"source":   "00000000-0000-0000-0000-000000000103",
		"render":   "00000000-0000-0000-0000-000000000103",
	}
	for kind, want := range tests {
		if got := systemFolderID(kind); got != want {
			t.Fatalf("kind %q: got %q, want %q", kind, got, want)
		}
	}
}

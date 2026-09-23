package instagram

import "testing"

func TestNewSelectsGraphHostForTokenFamily(t *testing.T) {
	tests := []struct {
		token string
		want  string
	}{
		{"IGAA-test", "https://graph.instagram.com/v22.0"},
		{"EAA-test", "https://graph.facebook.com/v22.0"},
	}
	for _, test := range tests {
		if got := New("v22.0", "user", test.token).baseURL; got != test.want {
			t.Fatalf("token %q: got %q, want %q", test.token[:3], got, test.want)
		}
	}
}

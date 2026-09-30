package instagram

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

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

func TestListMediaMapsPageAndCursor(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v22.0/user/media" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("after") != "cursor-1" || r.URL.Query().Get("access_token") != "token" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.Query().Get("fields"), "media_product_type") {
			t.Fatalf("media_product_type is not requested")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"data":[{"id":"reel-1","caption":"Demo","media_type":"VIDEO","media_product_type":"REELS","media_url":"https://cdn.example/reel.mp4","thumbnail_url":"https://cdn.example/reel.jpg","permalink":"https://instagram.com/reel/demo","timestamp":"2026-09-30T00:00:00+0000"}],"paging":{"cursors":{"after":"cursor-2"},"next":"https://next"}}`,
			)),
		}, nil
	})

	client := New("v22.0", "user", "token")
	client.baseURL = "https://graph.example/v22.0"
	client.http = &http.Client{Transport: transport}
	page, err := client.ListMedia(context.Background(), "cursor-1")
	if err != nil {
		t.Fatalf("list media: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "reel-1" || page.NextCursor != "cursor-2" {
		t.Fatalf("unexpected page: %#v", page)
	}
}

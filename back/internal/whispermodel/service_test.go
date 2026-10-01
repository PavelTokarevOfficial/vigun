package whispermodel

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type brokenBody struct{ sent bool }

func (b *brokenBody) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("connection lost")
}
func (b *brokenBody) Close() error { return nil }

func TestDownloadPublishesOnlyCompleteFile(t *testing.T) {
	dir := t.TempDir()
	service := New(&Repository{Dir: dir}, nil)
	service.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", ContentLength: 4, Body: io.NopCloser(strings.NewReader("data"))}, nil
	})}
	model := Catalog[0]
	service.download(model)
	info, err := os.Stat(dir + "/" + model.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 4 {
		t.Fatalf("unexpected size %d", info.Size())
	}
	if _, err = os.Stat(dir + "/" + model.Filename + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial file remains")
	}
}

func TestInterruptedDownloadIsNotInstalled(t *testing.T) {
	dir := t.TempDir()
	service := New(&Repository{Dir: dir}, nil)
	service.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", ContentLength: 100, Body: &brokenBody{}}, nil
	})}
	model := Catalog[0]
	service.download(model)
	if _, err := os.Stat(dir + "/" + model.Filename); !os.IsNotExist(err) {
		t.Fatal("interrupted download was installed")
	}
	if _, err := os.Stat(dir + "/" + model.Filename + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial file remains")
	}
	if service.states[model.ID].State != "error" {
		t.Fatalf("unexpected state %q", service.states[model.ID].State)
	}
}

func TestPresetArguments(t *testing.T) {
	for preset, want := range map[string]string{"fast": "-bs 1 -bo 1 -nf", "balanced": "-ml 10 -sow", "maximum": "-bs 8", "subtitles": "-ml 42 -sow"} {
		if got := strings.Join(presetOptions(preset).args(), " "); got != want {
			t.Errorf("%s: got %q want %q", preset, got, want)
		}
	}
}

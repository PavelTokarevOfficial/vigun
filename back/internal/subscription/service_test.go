package subscription

import (
	"context"
	"testing"
	"time"
)

type repositoryStub struct {
	target        Target
	identity      RemoteUser
	savedClips    []RemoteClip
	windowStarted time.Time
	windowEnded   time.Time
	viewedID      string
}

func (r *repositoryStub) List(context.Context) ([]Feed, error) { return nil, nil }
func (r *repositoryStub) Target(context.Context, string) (Target, error) {
	return r.target, nil
}
func (r *repositoryStub) UpdateIdentity(_ context.Context, _ string, user RemoteUser) error {
	r.identity = user
	return nil
}
func (r *repositoryStub) SaveWindow(_ context.Context, _ string, clips []RemoteClip, startedAt, endedAt time.Time) error {
	r.savedClips = clips
	r.windowStarted = startedAt
	r.windowEnded = endedAt
	return nil
}
func (r *repositoryStub) MarkViewed(_ context.Context, clipID string) error {
	r.viewedID = clipID
	return nil
}

type clipSourceStub struct {
	user          RemoteUser
	clips         []RemoteClip
	broadcasterID string
}

func (s *clipSourceStub) User(context.Context, string) (RemoteUser, error) {
	return s.user, nil
}
func (s *clipSourceStub) Clips(_ context.Context, broadcasterID string, _, _ time.Time) ([]RemoteClip, error) {
	s.broadcasterID = broadcasterID
	return s.clips, nil
}

func TestSyncResolvesTwitchIdentityAndSavesSevenDayWindow(t *testing.T) {
	repository := &repositoryStub{target: Target{StreamerID: "streamer", TwitchLogin: "login"}}
	source := &clipSourceStub{
		user:  RemoteUser{ID: "twitch-user", DisplayName: "Streamer"},
		clips: []RemoteClip{{ID: "clip"}},
	}
	service := New(repository, source)

	count, err := service.Sync(context.Background(), "streamer")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || source.broadcasterID != "twitch-user" {
		t.Fatalf("unexpected sync result: count=%d broadcaster=%q", count, source.broadcasterID)
	}
	if repository.identity.ID != "twitch-user" || len(repository.savedClips) != 1 {
		t.Fatalf("identity or clips were not persisted: %#v %#v", repository.identity, repository.savedClips)
	}
	window := repository.windowEnded.Sub(repository.windowStarted)
	if window < 7*24*time.Hour-time.Minute || window > 7*24*time.Hour+time.Minute {
		t.Fatalf("expected a seven day window, got %s", window)
	}
}

func TestMarkViewedDelegatesToRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := New(repository, &clipSourceStub{})
	if err := service.MarkViewed(context.Background(), "clip-id"); err != nil {
		t.Fatal(err)
	}
	if repository.viewedID != "clip-id" {
		t.Fatalf("expected clip-id, got %q", repository.viewedID)
	}
}

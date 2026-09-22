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
	cleared       bool
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
func (r *repositoryStub) Clear(context.Context) (int64, error) {
	r.cleared = true
	return 3, nil
}

type clipSourceStub struct {
	user          RemoteUser
	clips         []RemoteClip
	broadcasterID string
	startedAt     time.Time
	endedAt       time.Time
}

func (s *clipSourceStub) User(context.Context, string) (RemoteUser, error) {
	return s.user, nil
}
func (s *clipSourceStub) Clips(_ context.Context, broadcasterID string, startedAt, endedAt time.Time) ([]RemoteClip, error) {
	s.broadcasterID = broadcasterID
	s.startedAt = startedAt
	s.endedAt = endedAt
	return s.clips, nil
}

func TestSyncResolvesTwitchIdentityAndSavesSelectedWindow(t *testing.T) {
	repository := &repositoryStub{target: Target{StreamerID: "streamer", TwitchLogin: "login"}}
	source := &clipSourceStub{
		user:  RemoteUser{ID: "twitch-user", DisplayName: "Streamer"},
		clips: []RemoteClip{{ID: "clip"}},
	}
	service := New(repository, source)

	startedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	count, err := service.Sync(context.Background(), "streamer", startedAt, endedAt)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || source.broadcasterID != "twitch-user" {
		t.Fatalf("unexpected sync result: count=%d broadcaster=%q", count, source.broadcasterID)
	}
	if repository.identity.ID != "twitch-user" || len(repository.savedClips) != 1 {
		t.Fatalf("identity or clips were not persisted: %#v %#v", repository.identity, repository.savedClips)
	}
	if !repository.windowStarted.Equal(startedAt) || !repository.windowEnded.Equal(endedAt) {
		t.Fatalf("unexpected repository window: %s - %s", repository.windowStarted, repository.windowEnded)
	}
	if !source.startedAt.Equal(startedAt) || !source.endedAt.Equal(endedAt) {
		t.Fatalf("unexpected source window: %s - %s", source.startedAt, source.endedAt)
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

func TestClearDelegatesToRepository(t *testing.T) {
	repository := &repositoryStub{}
	service := New(repository, &clipSourceStub{})
	count, err := service.Clear(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || !repository.cleared {
		t.Fatalf("expected three cleared clips, got %d", count)
	}
}

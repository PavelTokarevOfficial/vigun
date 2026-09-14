package subscription

import (
	"context"
	"time"
)

const WindowDays = 7

type Clip struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     float64   `json:"duration"`
	CreatedAt    time.Time `json:"created_at"`
	Saved        bool      `json:"saved"`
	Viewed       bool      `json:"viewed"`
}

type Feed struct {
	StreamerID   string     `json:"streamerId"`
	TwitchLogin  string     `json:"twitchLogin"`
	DisplayName  string     `json:"displayName"`
	LastSyncedAt *time.Time `json:"lastSyncedAt"`
	Clips        []Clip     `json:"clips"`
}

type Target struct {
	StreamerID  string
	TwitchID    string
	TwitchLogin string
}

type RemoteUser struct {
	ID          string
	DisplayName string
}

type RemoteClip struct {
	ID           string
	Title        string
	URL          string
	ThumbnailURL string
	Duration     float64
	CreatedAt    time.Time
}

type Repository interface {
	List(context.Context) ([]Feed, error)
	Target(context.Context, string) (Target, error)
	UpdateIdentity(context.Context, string, RemoteUser) error
	SaveWindow(context.Context, string, []RemoteClip, time.Time, time.Time) error
	MarkViewed(context.Context, string) error
}

type ClipSource interface {
	User(context.Context, string) (RemoteUser, error)
	Clips(context.Context, string, time.Time, time.Time) ([]RemoteClip, error)
}

type Service struct {
	repository Repository
	source     ClipSource
}

func New(repository Repository, source ClipSource) *Service {
	return &Service{repository: repository, source: source}
}

func (s *Service) List(ctx context.Context) ([]Feed, error) {
	return s.repository.List(ctx)
}

func (s *Service) Sync(ctx context.Context, streamerID string) (int, error) {
	target, err := s.repository.Target(ctx, streamerID)
	if err != nil {
		return 0, err
	}
	if target.TwitchID == "" {
		user, lookupErr := s.source.User(ctx, target.TwitchLogin)
		if lookupErr != nil {
			return 0, lookupErr
		}
		target.TwitchID = user.ID
		if err = s.repository.UpdateIdentity(ctx, streamerID, user); err != nil {
			return 0, err
		}
	}

	endedAt := time.Now().UTC()
	startedAt := endedAt.AddDate(0, 0, -WindowDays)
	clips, err := s.source.Clips(ctx, target.TwitchID, startedAt, endedAt)
	if err != nil {
		return 0, err
	}
	if err = s.repository.SaveWindow(ctx, streamerID, clips, startedAt, endedAt); err != nil {
		return 0, err
	}
	return len(clips), nil
}

func (s *Service) MarkViewed(ctx context.Context, clipID string) error {
	return s.repository.MarkViewed(ctx, clipID)
}

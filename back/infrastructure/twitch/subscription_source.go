package twitch

import (
	"context"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/subscription"
)

type SubscriptionSource struct {
	client *Client
}

func NewSubscriptionSource(client *Client) *SubscriptionSource {
	return &SubscriptionSource{client: client}
}

func (s *SubscriptionSource) User(ctx context.Context, login string) (subscription.RemoteUser, error) {
	user, err := s.client.User(ctx, login)
	return subscription.RemoteUser{ID: user.ID, DisplayName: user.DisplayName}, err
}

func (s *SubscriptionSource) Clips(ctx context.Context, broadcasterID string, startedAt, endedAt time.Time) ([]subscription.RemoteClip, error) {
	clips, err := s.client.Clips(ctx, broadcasterID, startedAt, endedAt)
	if err != nil {
		return nil, err
	}
	result := make([]subscription.RemoteClip, 0, len(clips))
	for _, clip := range clips {
		result = append(result, subscription.RemoteClip{
			ID:           clip.ID,
			Title:        clip.Title,
			URL:          clip.URL,
			ThumbnailURL: clip.ThumbnailURL,
			Duration:     clip.Duration,
			CreatedAt:    clip.CreatedAt,
		})
	}
	return result, nil
}

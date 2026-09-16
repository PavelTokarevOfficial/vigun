// Package instagram owns the application flow for publishing rendered videos.
// It deliberately does not depend on HTTP or Meta's Graph API transport.
package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type CreateInput struct {
	VideoURL      string
	Caption       string
	ShareToFeed   bool
	Collaborators []string
	CoverURL      string
	AudioName     string
	LocationID    string
	ThumbOffset   *int
}

type Container struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type PublishedMedia struct {
	ID string `json:"id"`
}

type Client interface {
	CreateReel(context.Context, CreateInput) (string, error)
	ContainerStatus(context.Context, string) (Container, error)
	Publish(context.Context, string) (string, error)
}

type Service struct{ client Client }

func New(client Client) *Service { return &Service{client: client} }

func (s *Service) Create(ctx context.Context, input CreateInput) (Container, error) {
	if err := validateHTTPSURL("videoUrl", input.VideoURL); err != nil {
		return Container{}, err
	}
	if input.CoverURL != "" {
		if err := validateHTTPSURL("coverUrl", input.CoverURL); err != nil {
			return Container{}, err
		}
	}
	if input.ThumbOffset != nil && *input.ThumbOffset < 0 {
		return Container{}, fmt.Errorf("thumbOffset must be non-negative")
	}
	for i, username := range input.Collaborators {
		input.Collaborators[i] = strings.TrimPrefix(strings.TrimSpace(username), "@")
		if input.Collaborators[i] == "" {
			return Container{}, fmt.Errorf("collaborators must not contain empty usernames")
		}
	}
	id, err := s.client.CreateReel(ctx, input)
	if err != nil {
		return Container{}, err
	}
	return Container{ID: id, Status: "IN_PROGRESS"}, nil
}

func (s *Service) Status(ctx context.Context, id string) (Container, error) {
	if strings.TrimSpace(id) == "" {
		return Container{}, fmt.Errorf("container id is required")
	}
	return s.client.ContainerStatus(ctx, id)
}

func (s *Service) Publish(ctx context.Context, id string) (PublishedMedia, error) {
	if strings.TrimSpace(id) == "" {
		return PublishedMedia{}, fmt.Errorf("container id is required")
	}
	id, err := s.client.Publish(ctx, id)
	return PublishedMedia{ID: id}, err
}

func CollaboratorsJSON(values []string) string {
	encoded, _ := json.Marshal(values)
	return string(encoded)
}

func validateHTTPSURL(field, raw string) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be a public https URL", field)
	}
	return nil
}

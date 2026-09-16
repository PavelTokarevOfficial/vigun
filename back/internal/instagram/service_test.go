package instagram

import (
	"context"
	"testing"
)

type clientStub struct{ input CreateInput }

func (c *clientStub) CreateReel(_ context.Context, input CreateInput) (string, error) {
	c.input = input
	return "container-1", nil
}
func (c *clientStub) ContainerStatus(_ context.Context, id string) (Container, error) {
	return Container{ID: id, Status: "FINISHED"}, nil
}
func (c *clientStub) Publish(_ context.Context, _ string) (string, error) {
	return "media-1", nil
}

func TestCreateRequiresPublicHTTPSVideo(t *testing.T) {
	service := New(&clientStub{})
	for _, raw := range []string{"", "http://localhost:8080/video.mp4", "https:///video.mp4"} {
		if _, err := service.Create(context.Background(), CreateInput{VideoURL: raw}); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestCreateNormalizesCollaborators(t *testing.T) {
	client := &clientStub{}
	service := New(client)
	container, err := service.Create(context.Background(), CreateInput{
		VideoURL:      "https://example.trycloudflare.com/api/videos/1/content",
		Collaborators: []string{" @first ", "second"},
	})
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	if container.ID != "container-1" || client.input.Collaborators[0] != "first" {
		t.Fatalf("unexpected result: %#v %#v", container, client.input.Collaborators)
	}
}

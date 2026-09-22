package instagram

import (
	"context"
	"testing"
	"time"
)

type clientStub struct{ input CreateInput }

type accountRepositoryStub struct{}

func (accountRepositoryStub) List(context.Context) ([]Account, error) { return nil, nil }
func (accountRepositoryStub) Create(_ context.Context, input AccountInput) (Account, error) {
	return Account{ID: "account-1", Nickname: input.Nickname, InstagramUserID: input.InstagramUserID, HasAccessToken: true}, nil
}
func (accountRepositoryStub) Update(_ context.Context, id string, input AccountInput) (Account, error) {
	return Account{ID: id, Nickname: input.Nickname, InstagramUserID: input.InstagramUserID, HasAccessToken: true}, nil
}
func (accountRepositoryStub) Delete(context.Context, string) error { return nil }
func (accountRepositoryStub) Credentials(context.Context, string) (Credentials, error) {
	return Credentials{InstagramUserID: "user-1", AccessToken: "token-1"}, nil
}
func (accountRepositoryStub) UpdateToken(_ context.Context, id, _ string, expiresAt time.Time) (Account, error) {
	return Account{ID: id, TokenExpiresAt: &expiresAt}, nil
}
func (accountRepositoryStub) MarkTokenChecked(_ context.Context, id string, profile TokenProfile) (Account, error) {
	return Account{ID: id, VerifiedUsername: profile.Username, AccountType: profile.AccountType}, nil
}

type tokenManagerStub struct{}

func (tokenManagerStub) Exchange(context.Context, string, string) (TokenResult, error) {
	return TokenResult{AccessToken: "long-token", ExpiresIn: 60 * 24 * time.Hour}, nil
}
func (tokenManagerStub) Refresh(context.Context, string) (TokenResult, error) {
	return TokenResult{AccessToken: "fresh-token", ExpiresIn: 60 * 24 * time.Hour}, nil
}
func (tokenManagerStub) Inspect(context.Context, string) (TokenProfile, error) {
	return TokenProfile{ID: "user-1", Username: "account", AccountType: "BUSINESS"}, nil
}

func testService(client *clientStub) *Service {
	return New(accountRepositoryStub{}, func(_, _ string) Client { return client }, tokenManagerStub{}, "app-secret")
}

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
	service := testService(&clientStub{})
	for _, raw := range []string{"", "http://localhost:8080/video.mp4", "https:///video.mp4"} {
		if _, err := service.Create(context.Background(), "account-1", CreateInput{VideoURL: raw}); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestCreateNormalizesCollaborators(t *testing.T) {
	client := &clientStub{}
	service := testService(client)
	container, err := service.Create(context.Background(), "account-1", CreateInput{
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

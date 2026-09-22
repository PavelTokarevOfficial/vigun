// Package instagram owns the application flow for publishing rendered videos.
// It deliberately does not depend on HTTP or Meta's Graph API transport.
package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
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

type Account struct {
	ID                 string     `json:"id"`
	Nickname           string     `json:"nickname"`
	InstagramUserID    string     `json:"instagramUserId"`
	HasAccessToken     bool       `json:"hasAccessToken"`
	TokenUpdatedAt     time.Time  `json:"tokenUpdatedAt"`
	TokenExpiresAt     *time.Time `json:"tokenExpiresAt"`
	TokenLastCheckedAt *time.Time `json:"tokenLastCheckedAt"`
	VerifiedUsername   string     `json:"verifiedUsername"`
	AccountType        string     `json:"accountType"`
}

type AccountInput struct {
	Nickname        string `json:"nickname"`
	InstagramUserID string `json:"instagramUserId"`
	AccessToken     string `json:"accessToken"`
}

type Credentials struct {
	InstagramUserID string
	AccessToken     string
}

type AccountRepository interface {
	List(context.Context) ([]Account, error)
	Create(context.Context, AccountInput) (Account, error)
	Update(context.Context, string, AccountInput) (Account, error)
	Delete(context.Context, string) error
	Credentials(context.Context, string) (Credentials, error)
	UpdateToken(context.Context, string, string, time.Time) (Account, error)
	MarkTokenChecked(context.Context, string, TokenProfile) (Account, error)
}

type TokenResult struct {
	AccessToken string
	ExpiresIn   time.Duration
}

type TokenProfile struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	AccountType string `json:"accountType"`
}

type TokenManager interface {
	Exchange(context.Context, string, string) (TokenResult, error)
	Refresh(context.Context, string) (TokenResult, error)
	Inspect(context.Context, string) (TokenProfile, error)
}

type Client interface {
	CreateReel(context.Context, CreateInput) (string, error)
	ContainerStatus(context.Context, string) (Container, error)
	Publish(context.Context, string) (string, error)
}

type ClientFactory func(userID, accessToken string) Client

type Service struct {
	accounts  AccountRepository
	clients   ClientFactory
	tokens    TokenManager
	appSecret string
}

func New(accounts AccountRepository, clients ClientFactory, tokens TokenManager, appSecret string) *Service {
	return &Service{accounts: accounts, clients: clients, tokens: tokens, appSecret: strings.TrimSpace(appSecret)}
}

func (s *Service) Accounts(ctx context.Context) ([]Account, error) {
	return s.accounts.List(ctx)
}

func (s *Service) CreateAccount(ctx context.Context, input AccountInput) (Account, error) {
	input = normalizeAccountInput(input)
	if err := validateAccountInput(input, false); err != nil {
		return Account{}, err
	}
	return s.accounts.Create(ctx, input)
}

func (s *Service) UpdateAccount(ctx context.Context, id string, input AccountInput) (Account, error) {
	input = normalizeAccountInput(input)
	if strings.TrimSpace(id) == "" {
		return Account{}, fmt.Errorf("account id is required")
	}
	if err := validateAccountInput(input, true); err != nil {
		return Account{}, err
	}
	return s.accounts.Update(ctx, id, input)
}

func (s *Service) DeleteAccount(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("account id is required")
	}
	return s.accounts.Delete(ctx, id)
}

func (s *Service) ExchangeToken(ctx context.Context, id string) (Account, error) {
	if s.appSecret == "" {
		return Account{}, fmt.Errorf("INSTAGRAM_APP_SECRET is not configured")
	}
	credentials, err := s.accounts.Credentials(ctx, id)
	if err != nil {
		return Account{}, err
	}
	result, err := s.tokens.Exchange(ctx, credentials.AccessToken, s.appSecret)
	if err != nil {
		return Account{}, err
	}
	return s.saveToken(ctx, id, result)
}

func (s *Service) RefreshToken(ctx context.Context, id string) (Account, error) {
	credentials, err := s.accounts.Credentials(ctx, id)
	if err != nil {
		return Account{}, err
	}
	result, err := s.tokens.Refresh(ctx, credentials.AccessToken)
	if err != nil {
		return Account{}, err
	}
	return s.saveToken(ctx, id, result)
}

func (s *Service) CheckToken(ctx context.Context, id string) (Account, error) {
	credentials, err := s.accounts.Credentials(ctx, id)
	if err != nil {
		return Account{}, err
	}
	profile, err := s.tokens.Inspect(ctx, credentials.AccessToken)
	if err != nil {
		return Account{}, err
	}
	if profile.ID != credentials.InstagramUserID {
		return Account{}, fmt.Errorf("token belongs to Instagram user %s, expected %s", profile.ID, credentials.InstagramUserID)
	}
	return s.accounts.MarkTokenChecked(ctx, id, profile)
}

func (s *Service) saveToken(ctx context.Context, id string, result TokenResult) (Account, error) {
	if strings.TrimSpace(result.AccessToken) == "" || result.ExpiresIn <= 0 {
		return Account{}, fmt.Errorf("Instagram returned an invalid token response")
	}
	return s.accounts.UpdateToken(ctx, id, result.AccessToken, time.Now().UTC().Add(result.ExpiresIn))
}

func (s *Service) client(ctx context.Context, accountID string) (Client, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, fmt.Errorf("Instagram account is required")
	}
	credentials, err := s.accounts.Credentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.clients(credentials.InstagramUserID, credentials.AccessToken), nil
}

func (s *Service) Create(ctx context.Context, accountID string, input CreateInput) (Container, error) {
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
	client, err := s.client(ctx, accountID)
	if err != nil {
		return Container{}, err
	}
	id, err := client.CreateReel(ctx, input)
	if err != nil {
		return Container{}, err
	}
	return Container{ID: id, Status: "IN_PROGRESS"}, nil
}

func (s *Service) Status(ctx context.Context, accountID, id string) (Container, error) {
	if strings.TrimSpace(id) == "" {
		return Container{}, fmt.Errorf("container id is required")
	}
	client, err := s.client(ctx, accountID)
	if err != nil {
		return Container{}, err
	}
	return client.ContainerStatus(ctx, id)
}

func (s *Service) Publish(ctx context.Context, accountID, id string) (PublishedMedia, error) {
	if strings.TrimSpace(id) == "" {
		return PublishedMedia{}, fmt.Errorf("container id is required")
	}
	client, err := s.client(ctx, accountID)
	if err != nil {
		return PublishedMedia{}, err
	}
	publishedID, err := client.Publish(ctx, id)
	return PublishedMedia{ID: publishedID}, err
}

func normalizeAccountInput(input AccountInput) AccountInput {
	input.Nickname = strings.TrimSpace(strings.TrimPrefix(input.Nickname, "@"))
	input.InstagramUserID = strings.TrimSpace(input.InstagramUserID)
	input.AccessToken = strings.TrimSpace(input.AccessToken)
	return input
}

func validateAccountInput(input AccountInput, tokenOptional bool) error {
	if input.Nickname == "" {
		return fmt.Errorf("nickname is required")
	}
	if input.InstagramUserID == "" {
		return fmt.Errorf("Instagram user ID is required")
	}
	if !tokenOptional && input.AccessToken == "" {
		return fmt.Errorf("Instagram access token is required")
	}
	return nil
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

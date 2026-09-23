package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	app "github.com/finde-clip/finde-v2/back/internal/instagram"
)

type Client struct {
	baseURL, userID, accessToken string
	http                         *http.Client
}

type Factory struct {
	version string
	http    *http.Client
}

func NewFactory(version string) *Factory {
	return &Factory{version: version, http: &http.Client{Timeout: 30 * time.Second}}
}

func (f *Factory) New(userID, accessToken string) app.Client {
	return New(f.version, userID, accessToken)
}

func (f *Factory) Exchange(ctx context.Context, accessToken, appSecret string) (app.TokenResult, error) {
	query := url.Values{
		"grant_type":    {"ig_exchange_token"},
		"client_secret": {appSecret},
		"access_token":  {accessToken},
	}
	return f.tokenRequest(ctx, "https://graph.instagram.com/access_token?"+query.Encode())
}

func (f *Factory) Refresh(ctx context.Context, accessToken string) (app.TokenResult, error) {
	query := url.Values{
		"grant_type":   {"ig_refresh_token"},
		"access_token": {accessToken},
	}
	return f.tokenRequest(ctx, "https://graph.instagram.com/refresh_access_token?"+query.Encode())
}

func (f *Factory) Inspect(ctx context.Context, accessToken string) (app.TokenProfile, error) {
	query := url.Values{
		"fields":       {"id,username,account_type"},
		"access_token": {accessToken},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.instagram.com/me?"+query.Encode(), nil)
	if err != nil {
		return app.TokenProfile{}, err
	}
	var response struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		AccountType string `json:"account_type"`
	}
	if err = (&Client{http: f.http}).do(request, &response); err != nil {
		return app.TokenProfile{}, err
	}
	return app.TokenProfile{ID: response.ID, Username: response.Username, AccountType: response.AccountType}, nil
}

func (f *Factory) tokenRequest(ctx context.Context, endpoint string) (app.TokenResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return app.TokenResult{}, err
	}
	var response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err = (&Client{http: f.http}).do(request, &response); err != nil {
		return app.TokenResult{}, err
	}
	return app.TokenResult{
		AccessToken: response.AccessToken,
		ExpiresIn:   time.Duration(response.ExpiresIn) * time.Second,
	}, nil
}

func New(version, userID, accessToken string) *Client {
	version = strings.Trim(strings.TrimSpace(version), "/")
	host := "https://graph.facebook.com/"
	if strings.HasPrefix(accessToken, "IGA") {
		host = "https://graph.instagram.com/"
	}
	return &Client{
		baseURL:     host + version,
		userID:      userID,
		accessToken: accessToken,
		http:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) CreateReel(ctx context.Context, input app.CreateInput) (string, error) {
	form := url.Values{
		"media_type":    {"REELS"},
		"video_url":     {input.VideoURL},
		"caption":       {input.Caption},
		"share_to_feed": {fmt.Sprintf("%t", input.ShareToFeed)},
	}
	if len(input.Collaborators) > 0 {
		form.Set("collaborators", app.CollaboratorsJSON(input.Collaborators))
	}
	if input.CoverURL != "" {
		form.Set("cover_url", input.CoverURL)
	}
	if input.AudioName != "" {
		form.Set("audio_name", input.AudioName)
	}
	if input.LocationID != "" {
		form.Set("location_id", input.LocationID)
	}
	if input.ThumbOffset != nil {
		form.Set("thumb_offset", fmt.Sprintf("%d", *input.ThumbOffset))
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, "/"+c.userID+"/media", form, &response); err != nil {
		return "", err
	}
	return response.ID, nil
}

func (c *Client) ContainerStatus(ctx context.Context, id string) (app.Container, error) {
	query := url.Values{"fields": {"status_code,status"}, "access_token": {c.accessToken}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/"+url.PathEscape(id)+"?"+query.Encode(), nil)
	if err != nil {
		return app.Container{}, err
	}
	var response struct {
		ID         string `json:"id"`
		StatusCode string `json:"status_code"`
		Status     string `json:"status"`
	}
	if err = c.do(request, &response); err != nil {
		return app.Container{}, err
	}
	return app.Container{ID: response.ID, Status: response.StatusCode, Error: response.Status}, nil
}

func (c *Client) Publish(ctx context.Context, containerID string) (string, error) {
	var response struct {
		ID string `json:"id"`
	}
	err := c.post(ctx, "/"+c.userID+"/media_publish", url.Values{"creation_id": {containerID}}, &response)
	return response.ID, err
}

func (c *Client) post(ctx context.Context, path string, form url.Values, target any) error {
	form.Set("access_token", c.accessToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(request, target)
}

func (c *Client) do(request *http.Request, target any) error {
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("instagram request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read instagram response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &failure)
		if failure.Error.Message == "" {
			failure.Error.Message = response.Status
		}
		return fmt.Errorf("instagram: %s", failure.Error.Message)
	}
	if err = json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode instagram response: %w", err)
	}
	return nil
}

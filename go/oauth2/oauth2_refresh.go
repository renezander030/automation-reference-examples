package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// TokenStore holds OAuth2 credentials and manages automatic refresh.
type TokenStore struct {
	client       *http.Client
	mu           sync.RWMutex
	clientID     string
	clientSecret string
	refreshToken string
	tokenURI     string
	accessToken  string
	expiresAt    time.Time
}

func NewTokenStore(clientID, clientSecret, refreshToken, tokenURI string, clients ...*http.Client) *TokenStore {
	client := &http.Client{Timeout: 10 * time.Second}
	if len(clients) > 0 && clients[0] != nil {
		client = clients[0]
	}
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}
	return &TokenStore{
		client:       client,
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		tokenURI:     tokenURI,
	}
}

// Token returns a valid access token, refreshing if expired.
func (t *TokenStore) Token() (string, error) { return t.TokenContext(context.Background()) }

func (t *TokenStore) TokenContext(ctx context.Context) (string, error) {
	t.mu.RLock()
	if time.Now().Before(t.expiresAt) {
		token := t.accessToken
		t.mu.RUnlock()
		return token, nil
	}
	t.mu.RUnlock()

	return t.refresh(ctx)
}

func (t *TokenStore) refresh(parent context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(t.expiresAt) {
		return t.accessToken, nil
	}

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	form := url.Values{
		"client_id":     {t.clientID},
		"client_secret": {t.clientSecret},
		"refresh_token": {t.refreshToken},
		"grant_type":    {"refresh_token"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("invalid token endpoint")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token refresh request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil {
		return "", fmt.Errorf("token response read failed")
	}
	if len(body) > 65536 {
		return "", fmt.Errorf("token response too large")
	}
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("invalid token response JSON")
	}
	if result.Error != "" {
		return "", fmt.Errorf("token refresh rejected")
	}

	if result.AccessToken == "" || result.ExpiresIn <= 0 || result.ExpiresIn > 86400*365 {
		return "", fmt.Errorf("invalid access token or expires_in")
	}
	ttl := time.Duration(result.ExpiresIn) * time.Second
	margin := 60 * time.Second
	if ttl/10 < margin {
		margin = ttl / 10
	}
	t.accessToken = result.AccessToken
	t.expiresAt = time.Now().Add(ttl - margin)

	return t.accessToken, nil
}

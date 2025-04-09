// Package providers implements authentication providers for the notification service.

// AuthProvider defines the interface for adding authentication to HTTP requests.
// Different implementations can provide various authentication mechanisms.

// CognitoAuthProvider implements AuthProvider interface using AWS Cognito for authentication.
// It manages OAuth2 client credentials flow, handling token generation and caching.
//
// The provider automatically handles token refresh when expired using a mutex for
// thread-safe operations. It caches the token until expiry to minimize API calls.
//
// Example usage:
//
//	provider := NewCognitoAuthProvider(CognitoAuthProviderCfg{
//	    UserPoolID:   "us-east-1_xxxxx",
//	    ClientID:     "client-id",
//	    ClientSecret: "client-secret",
//	    Region:       "us-east-1",
//	})
//
//	req, _ := http.NewRequest("GET", "https://api.example.com", nil)
//	err := provider.AddAuth(req)
//	if err != nil {
//	    // Handle error
//	}
//
// The provider will automatically add the Bearer token to request headers.
package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/notifique/shared/auth"
)

type AuthProvider interface {
	AddAuth(req *http.Request) error
}

type noAuthFunc func(req *http.Request) error

func (f noAuthFunc) AddAuth(req *http.Request) error {
	return f(req)
}

// NoAuth is an AuthProvider implementation that should be used when the
// notification service has authentication disabled. It adds a mock user
// ID to requests.
var NoAuth noAuthFunc = func(req *http.Request) error {
	req.Header.Set(string(auth.UserHeader), "mock-user-id")
	return nil
}

type CognitoAuthProviderCfg struct {
	TokenUrl     string
	ClientID     string
	ClientSecret string
}

type CognitoAuthConfigurator interface {
	GetCognitoAuthCfg() (CognitoAuthProviderCfg, error)
}

type CognitoAuthProvider struct {
	tokenUrl     string
	clientID     string
	clientSecret string
	token        string
	tokenExpiry  time.Time
	mutex        sync.RWMutex
}

func NewCognitoAuthProvider(c CognitoAuthConfigurator) (*CognitoAuthProvider, error) {
	cfg, err := c.GetCognitoAuthCfg()

	if err != nil {
		return nil, fmt.Errorf("failed to get Cognito auth config: %w", err)
	}

	p := &CognitoAuthProvider{
		tokenUrl:     cfg.TokenUrl,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}

	return p, nil
}

func (p *CognitoAuthProvider) AddAuth(req *http.Request) error {
	token, err := p.getToken()

	if err != nil {
		return fmt.Errorf("failed to get valid token: %w", err)
	}

	req.Header.Set("Authorization", token)
	return nil
}

func (p *CognitoAuthProvider) getToken() (string, error) {

	p.mutex.RLock()

	if p.token != "" && time.Now().Before(p.tokenExpiry) {
		token := p.token
		p.mutex.RUnlock()
		return token, nil
	}

	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double check after acquiring write lock
	if p.token != "" && time.Now().Before(p.tokenExpiry) {
		return p.token, nil
	}

	token, expiry, err := p.GenerateToken()
	if err != nil {
		return "", err
	}

	p.token = token
	p.tokenExpiry = expiry
	return token, nil
}

func (p *CognitoAuthProvider) GenerateToken() (string, time.Time, error) {

	req, err := http.NewRequest(http.MethodPost, p.tokenUrl, nil)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create token request: %w", err)
	}

	scopes := []string{
		string(auth.UserNotificationPublisher),
		string(auth.NotificationsPublisher),
	}

	scope := strings.Join(scopes, " ")

	query := req.URL.Query()
	query.Set("grant_type", "client_credentials")
	query.Set("scope", scope)
	req.URL.RawQuery = query.Encode()

	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to execute token request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode token response: %w", err)
	}

	expiry := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return result.AccessToken, expiry, nil
}

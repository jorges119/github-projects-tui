package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	deviceCodeURL = "https://github.com/login/device/code"
	tokenURL      = "https://github.com/login/oauth/access_token"
	validateURL   = "https://api.github.com/user"
	scope         = "repo,project,read:org"
)

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

type UserInfo struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// RequestDeviceCode starts the device authorization flow.
func RequestDeviceCode(clientID string) (*DeviceCode, error) {
	resp, err := http.PostForm(deviceCodeURL, url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	})
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}
	defer resp.Body.Close()

	var dc DeviceCode
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decoding device code response: %w", err)
	}
	return &dc, nil
}

// PollForToken polls GitHub until the user authorizes the device or the code expires.
func PollForToken(ctx context.Context, clientID string, dc *DeviceCode) (string, error) {
	interval := time.Duration(dc.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}

		resp, err := http.PostForm(tokenURL, url.Values{
			"client_id":   {clientID},
			"device_code": {dc.DeviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
		if err != nil {
			continue
		}

		var tr tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		switch tr.Error {
		case "":
			if tr.AccessToken != "" {
				return tr.AccessToken, nil
			}
		case "authorization_pending", "slow_down":
			if tr.Error == "slow_down" {
				interval += 5 * time.Second
			}
		case "expired_token":
			return "", fmt.Errorf("device code expired, please try again")
		case "access_denied":
			return "", fmt.Errorf("authorization denied by user")
		default:
			return "", fmt.Errorf("unexpected error: %s", tr.Error)
		}
	}
	return "", fmt.Errorf("device code expired")
}

// ValidateToken checks that the token is valid and returns user info.
func ValidateToken(token string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", validateURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("validating token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("token is invalid or expired")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var u UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}
	return &u, nil
}

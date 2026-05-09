package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// FacebookConfig holds Facebook OAuth credentials.
type FacebookConfig struct {
	AppID     string
	AppSecret string
}

// FacebookVerifier exchanges an OAuth code for user info via Facebook Graph API.
type FacebookVerifier struct {
	cfg     FacebookConfig
	httpGet func(url string) (*http.Response, error)
}

func NewFacebookVerifier(cfg FacebookConfig) *FacebookVerifier {
	return &FacebookVerifier{cfg: cfg, httpGet: http.Get}
}

// Exchange takes the OAuth code and redirect_uri from the callback and returns verified user claims.
func (v *FacebookVerifier) Exchange(code, redirectURI string) (*GoogleClaims, error) {
	// Exchange code for access token
	tokenURL := fmt.Sprintf(
		"https://graph.facebook.com/v21.0/oauth/access_token?client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		url.QueryEscape(v.cfg.AppID),
		url.QueryEscape(v.cfg.AppSecret),
		url.QueryEscape(redirectURI),
		url.QueryEscape(code),
	)

	resp, err := v.httpGet(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook token exchange HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	// Fetch user info
	meURL := fmt.Sprintf(
		"https://graph.facebook.com/v21.0/me?fields=id,name,email,picture.type(large)&access_token=%s",
		url.QueryEscape(tokenResp.AccessToken),
	)

	meResp, err := v.httpGet(meURL)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}
	defer meResp.Body.Close()

	if meResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook /me HTTP %d", meResp.StatusCode)
	}

	var user struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.NewDecoder(meResp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	if user.Email == "" {
		return nil, fmt.Errorf("facebook account has no email")
	}

	return &GoogleClaims{
		Subject: user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Picture: user.Picture.Data.URL,
	}, nil
}

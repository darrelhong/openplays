package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GoogleOAuthConfig holds Google OAuth 2.0 credentials for the authorization code flow.
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// GoogleOAuthVerifier exchanges an authorization code for user info via Google's OAuth 2.0 flow.
type GoogleOAuthVerifier struct {
	cfg    GoogleOAuthConfig
	httpDo func(req *http.Request) (*http.Response, error)
}

func NewGoogleOAuthVerifier(cfg GoogleOAuthConfig) *GoogleOAuthVerifier {
	return &GoogleOAuthVerifier{cfg: cfg, httpDo: http.DefaultClient.Do}
}

// Exchange takes the OAuth authorization code and redirect_uri, exchanges for tokens,
// then fetches user info from Google's userinfo endpoint.
func (v *GoogleOAuthVerifier) Exchange(code, redirectURI string) (*GoogleClaims, error) {
	// Exchange code for tokens
	data := url.Values{
		"code":          {code},
		"client_id":     {v.cfg.ClientID},
		"client_secret": {v.cfg.ClientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.httpDo(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token exchange HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	// Fetch user info using the access token
	userReq, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := v.httpDo(userReq)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo HTTP %d", userResp.StatusCode)
	}

	var user struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}

	if !user.VerifiedEmail {
		return nil, fmt.Errorf("google email not verified")
	}

	return &GoogleClaims{
		Subject: user.ID,
		Email:   user.Email,
		Name:    user.Name,
		Picture: user.Picture,
	}, nil
}

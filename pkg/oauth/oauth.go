package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Profile struct {
	ProviderUserID string
	Email          string
	Name           string
	AvatarURL      string
}

type Provider struct {
	Key          string
	Configured   bool
	RedirectURL  string
	ClientID     string
	ClientSecret string
	oauth2       *oauth2.Config
}

type Registry struct {
	providers map[string]*Provider
	callback  string
}

func NewRegistry(frontendURL string, googleID, googleSecret, discordID, discordSecret, vkID, vkSecret string) *Registry {
	callback := strings.TrimRight(frontendURL, "/") + "/auth/social/callback"
	r := &Registry{
		providers: make(map[string]*Provider),
		callback:  callback,
	}
	if googleID != "" && googleSecret != "" {
		cfg := &oauth2.Config{
			ClientID:     googleID,
			ClientSecret: googleSecret,
			RedirectURL:  callback,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
		r.providers["google"] = &Provider{
			Key: "google", Configured: true, RedirectURL: callback,
			ClientID: googleID, ClientSecret: googleSecret, oauth2: cfg,
		}
	}
	if discordID != "" && discordSecret != "" {
		cfg := &oauth2.Config{
			ClientID:     discordID,
			ClientSecret: discordSecret,
			RedirectURL:  callback,
			Scopes:       []string{"identify", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
		}
		r.providers["discord"] = &Provider{
			Key: "discord", Configured: true, RedirectURL: callback,
			ClientID: discordID, ClientSecret: discordSecret, oauth2: cfg,
		}
	}
	if vkID != "" && vkSecret != "" {
		r.providers["vk"] = &Provider{
			Key: "vk", Configured: true, RedirectURL: callback,
			ClientID: vkID, ClientSecret: vkSecret,
		}
	}
	return r
}

func (r *Registry) CallbackURL() string { return r.callback }

func (r *Registry) Get(provider string) (*Provider, bool) {
	p, ok := r.providers[strings.ToLower(strings.TrimSpace(provider))]
	return p, ok
}

func (p *Provider) AuthCodeURL(state string) (string, error) {
	switch p.Key {
	case "google", "discord":
		if p.oauth2 == nil {
			return "", fmt.Errorf("oauth not configured")
		}
		return p.oauth2.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
	case "vk":
		q := url.Values{}
		q.Set("client_id", p.ClientID)
		q.Set("redirect_uri", p.RedirectURL)
		q.Set("scope", "email")
		q.Set("response_type", "code")
		q.Set("v", "5.131")
		q.Set("state", state)
		return "https://oauth.vk.com/authorize?" + q.Encode(), nil
	default:
		return "", fmt.Errorf("unsupported provider")
	}
}

func (r *Registry) Exchange(ctx context.Context, provider, code string) (Profile, error) {
	p, ok := r.Get(provider)
	if !ok || !p.Configured {
		return Profile{}, fmt.Errorf("provider not configured")
	}
	switch p.Key {
	case "google":
		return r.exchangeGoogle(ctx, p, code)
	case "discord":
		return r.exchangeDiscord(ctx, p, code)
	case "vk":
		return r.exchangeVK(ctx, p, code)
	default:
		return Profile{}, fmt.Errorf("unsupported provider")
	}
}

func (r *Registry) exchangeGoogle(ctx context.Context, p *Provider, code string) (Profile, error) {
	tok, err := p.oauth2.Exchange(ctx, code)
	if err != nil {
		return Profile{}, err
	}
	client := p.oauth2.Client(ctx, tok)
	res, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return Profile{}, err
	}
	defer res.Body.Close()
	var u struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(res.Body).Decode(&u); err != nil {
		return Profile{}, err
	}
	return Profile{
		ProviderUserID: u.ID,
		Email:          strings.TrimSpace(u.Email),
		Name:           strings.TrimSpace(u.Name),
		AvatarURL:      strings.TrimSpace(u.Picture),
	}, nil
}

func (r *Registry) exchangeDiscord(ctx context.Context, p *Provider, code string) (Profile, error) {
	tok, err := p.oauth2.Exchange(ctx, code)
	if err != nil {
		return Profile{}, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Profile{}, err
	}
	defer res.Body.Close()
	var u struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
	}
	if err := json.NewDecoder(res.Body).Decode(&u); err != nil {
		return Profile{}, err
	}
	name := strings.TrimSpace(u.GlobalName)
	if name == "" {
		name = strings.TrimSpace(u.Username)
	}
	avatar := ""
	if u.Avatar != "" {
		avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", u.ID, u.Avatar)
	}
	return Profile{
		ProviderUserID: u.ID,
		Email:          strings.TrimSpace(u.Email),
		Name:           name,
		AvatarURL:      avatar,
	}, nil
}

func (r *Registry) exchangeVK(ctx context.Context, p *Provider, code string) (Profile, error) {
	q := url.Values{}
	q.Set("client_id", p.ClientID)
	q.Set("client_secret", p.ClientSecret)
	q.Set("redirect_uri", p.RedirectURL)
	q.Set("code", code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://oauth.vk.com/access_token?"+q.Encode(), nil)
	if err != nil {
		return Profile{}, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return Profile{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var tok struct {
		AccessToken string `json:"access_token"`
		UserID      int64  `json:"user_id"`
		Email       string `json:"email"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return Profile{}, err
	}
	if tok.Error != "" {
		return Profile{}, fmt.Errorf("vk: %s", tok.ErrorDesc)
	}
	name := ""
	avatar := ""
	if tok.AccessToken != "" && tok.UserID > 0 {
		uq := url.Values{}
		uq.Set("user_ids", fmt.Sprintf("%d", tok.UserID))
		uq.Set("fields", "photo_200")
		uq.Set("access_token", tok.AccessToken)
		uq.Set("v", "5.131")
		ureq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.vk.com/method/users.get?"+uq.Encode(), nil)
		ures, err := http.DefaultClient.Do(ureq)
		if err == nil {
			defer ures.Body.Close()
			var vk struct {
				Response []struct {
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
					Photo200  string `json:"photo_200"`
				} `json:"response"`
			}
			if json.NewDecoder(ures.Body).Decode(&vk) == nil && len(vk.Response) > 0 {
				name = strings.TrimSpace(vk.Response[0].FirstName + " " + vk.Response[0].LastName)
				avatar = vk.Response[0].Photo200
			}
		}
	}
	return Profile{
		ProviderUserID: fmt.Sprintf("%d", tok.UserID),
		Email:          strings.TrimSpace(tok.Email),
		Name:           name,
		AvatarURL:      avatar,
	}, nil
}

func (r *Registry) ConfiguredKeys() []string {
	keys := make([]string, 0, len(r.providers))
	for key, p := range r.providers {
		if p != nil && p.Configured {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

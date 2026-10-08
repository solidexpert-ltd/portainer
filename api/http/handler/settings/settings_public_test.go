package settings

import (
	"net/url"
	"strings"
	"testing"

	portainer "github.com/portainer/portainer/api"
)

const (
	dummyOAuthClientID          = "1a2b3c4d"
	dummyOAuthScopes            = "openid user:email"
	dummyOAuthAuthenticationURI = "example.com/auth"
	dummyOAuthRedirectURI       = "https://develop.1crm.io/"
	dummyOAuthLogoutURI         = "example.com/logout"
)

func newTestSettings() *portainer.Settings {
	return &portainer.Settings{
		AuthenticationMethod: portainer.AuthenticationOAuth,
		OAuthSettings: portainer.OAuthSettings{
			AuthorizationURI: dummyOAuthAuthenticationURI,
			ClientID:         dummyOAuthClientID,
			Scopes:           dummyOAuthScopes,
			RedirectURI:      dummyOAuthRedirectURI,
			LogoutURI:        dummyOAuthLogoutURI,
		},
	}
}

func TestGeneratePublicSettingsWithSSO(t *testing.T) {
	t.Parallel()
	mockAppSettings := newTestSettings()
	mockAppSettings.OAuthSettings.SSO = true

	publicSettings := generatePublicSettings(mockAppSettings)
	if publicSettings.AuthenticationMethod != portainer.AuthenticationOAuth {
		t.Errorf("wrong AuthenticationMethod, want: %d, got: %d", portainer.AuthenticationOAuth, publicSettings.AuthenticationMethod)
	}

	expectedOAuthLoginURI := buildOAuthLoginURI(&mockAppSettings.OAuthSettings)
	if publicSettings.OAuthLoginURI != expectedOAuthLoginURI {
		t.Errorf("wrong OAuthLoginURI when SSO is switched on, want: %s, got: %s", expectedOAuthLoginURI, publicSettings.OAuthLoginURI)
	}

	if publicSettings.OAuthLogoutURI != dummyOAuthLogoutURI {
		t.Errorf("wrong OAuthLogoutURI, want: %s, got: %s", dummyOAuthLogoutURI, publicSettings.OAuthLogoutURI)
	}
}

func TestGeneratePublicSettingsWithoutSSO(t *testing.T) {
	t.Parallel()
	mockAppSettings := newTestSettings()
	mockAppSettings.OAuthSettings.SSO = false

	publicSettings := generatePublicSettings(mockAppSettings)
	if publicSettings.AuthenticationMethod != portainer.AuthenticationOAuth {
		t.Errorf("wrong AuthenticationMethod, want: %d, got: %d", portainer.AuthenticationOAuth, publicSettings.AuthenticationMethod)
	}

	expectedOAuthLoginURI := buildOAuthLoginURI(&mockAppSettings.OAuthSettings)
	if publicSettings.OAuthLoginURI != expectedOAuthLoginURI {
		t.Errorf("wrong OAuthLoginURI when SSO is switched off, want: %s, got: %s", expectedOAuthLoginURI, publicSettings.OAuthLoginURI)
	}
	if !strings.Contains(publicSettings.OAuthLoginURI, "prompt=login") {
		t.Errorf("SSO=false must include prompt=login; got %s", publicSettings.OAuthLoginURI)
	}

	if publicSettings.OAuthLogoutURI != dummyOAuthLogoutURI {
		t.Errorf("wrong OAuthLogoutURI, want: %s, got: %s", dummyOAuthLogoutURI, publicSettings.OAuthLogoutURI)
	}
}

func TestBuildOAuthLoginURIEncodesQueryValues(t *testing.T) {
	t.Parallel()

	oauth := &portainer.OAuthSettings{
		AuthorizationURI: "https://1crm.io/api/user/connect/authorize",
		ClientID:         "portainer-developer-console",
		RedirectURI:      "https://develop.1crm.io/",
		Scopes:           "openid user:email offline_access",
		SSO:              true,
	}

	got := buildOAuthLoginURI(oauth)
	if strings.Contains(got, "code_challenge") {
		t.Fatalf("login URI must not include PKCE challenge (browser-side); got %s", got)
	}
	if !strings.Contains(got, "redirect_uri="+url.QueryEscape(oauth.RedirectURI)) {
		t.Fatalf("redirect_uri should be query-escaped; got %s", got)
	}
	if !strings.Contains(got, "scope="+url.QueryEscape(oauth.Scopes)) {
		t.Fatalf("scope should be query-escaped; got %s", got)
	}
	if strings.Contains(got, "prompt=login") {
		t.Fatalf("SSO=true must omit prompt=login; got %s", got)
	}
}

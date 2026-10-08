package oauth

import (
	"testing"

	portainer "github.com/portainer/portainer/api"
)

func Test_getUsername(t *testing.T) {
	t.Parallel()
	t.Run("fails for non-matching user identifier", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"name": "john"}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err == nil {
			t.Errorf("getUsername should fail if user identifier doesn't exist as key in oauth userinfo object")
		}
	})

	t.Run("fails if username is empty string", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": ""}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err == nil {
			t.Errorf("getUsername should fail if username from oauth userinfo object is empty string")
		}
	})

	t.Run("fails if username is 0 int", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": 0}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err == nil {
			t.Errorf("getUsername should fail if username from oauth userinfo object is 0 val int")
		}
	})

	t.Run("fails if username is negative int", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": -1}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err == nil {
			t.Errorf("getUsername should fail if username from oauth userinfo object is -1 (negative) int")
		}
	})

	t.Run("succeeds if username is matched and is not empty", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": "john"}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err != nil {
			t.Errorf("getUsername should succeed if username from oauth userinfo object matched and non-empty")
		}
	})

	// looks like a bug!?
	t.Run("fails if username is matched and is positive int", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": 1}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err == nil {
			t.Errorf("getUsername should fail if username from oauth userinfo object matched is positive int")
		}
	})

	t.Run("succeeds if username is matched and is non-zero (or negative) float", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "username"}
		datamap := map[string]any{"username": 1.1}

		if _, err := GetUsername(datamap, oauthSettings.UserIdentifier); err != nil {
			t.Errorf("getUsername should succeed if username from oauth userinfo object matched and non-zero (or negative)")
		}
	})

	t.Run("falls back to preferred_username when email empty", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "email"}
		datamap := map[string]any{
			"email":               "",
			"preferred_username":  "+375291112233",
			"sub":                 "user-guid",
		}

		got, err := GetUsername(datamap, oauthSettings.UserIdentifier)
		if err != nil {
			t.Fatalf("expected fallback username, got err: %v", err)
		}
		if got != "+375291112233" {
			t.Fatalf("expected preferred_username fallback, got %q", got)
		}
	})

	t.Run("falls back to phone_number then sub", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "email"}
		datamap := map[string]any{
			"phone_number": "+375291112233",
			"sub":          "user-guid",
		}

		got, err := GetUsername(datamap, oauthSettings.UserIdentifier)
		if err != nil {
			t.Fatalf("expected phone_number fallback, got err: %v", err)
		}
		if got != "+375291112233" {
			t.Fatalf("expected phone_number, got %q", got)
		}
	})

	t.Run("falls back to sub when only sub present", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "email"}
		datamap := map[string]any{"sub": "user-guid"}

		got, err := GetUsername(datamap, oauthSettings.UserIdentifier)
		if err != nil {
			t.Fatalf("expected sub fallback, got err: %v", err)
		}
		if got != "user-guid" {
			t.Fatalf("expected sub, got %q", got)
		}
	})

	t.Run("prefers configured identifier over later fallbacks", func(t *testing.T) {
		oauthSettings := &portainer.OAuthSettings{UserIdentifier: "preferred_username"}
		datamap := map[string]any{
			"email":              "ops@1crm.io",
			"preferred_username": "+375291112233",
		}

		got, err := GetUsername(datamap, oauthSettings.UserIdentifier)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != "+375291112233" {
			t.Fatalf("expected preferred_username, got %q", got)
		}
	})
}

func Test_splitOAuthScopes(t *testing.T) {
	t.Parallel()

	got := splitOAuthScopes("openid user:email user:firstName")
	if len(got) != 3 || got[0] != "openid" || got[1] != "user:email" || got[2] != "user:firstName" {
		t.Fatalf("space-separated scopes: got %#v", got)
	}

	got = splitOAuthScopes("openid,user:email, offline_access")
	if len(got) != 3 || got[0] != "openid" || got[1] != "user:email" || got[2] != "offline_access" {
		t.Fatalf("comma-separated scopes: got %#v", got)
	}

	got = splitOAuthScopes("openid, user:email user:lastName")
	if len(got) != 3 {
		t.Fatalf("mixed separators: got %#v", got)
	}
}

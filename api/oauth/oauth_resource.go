package oauth

import (
	"errors"
	"strconv"
)

// usernameClaimFallbacks are tried (in order) when the configured UserIdentifier
// is missing or empty. preferred_username is always set by 1CRM user-service
// OidcController; email is often empty for phone-first accounts.
var usernameClaimFallbacks = []string{
	"email",
	"preferred_username",
	"phone_number",
	"sub",
}

func GetUsername(datamap map[string]any, userIdentifier string) (string, error) {
	if username, ok := claimAsUsername(datamap, userIdentifier); ok {
		return username, nil
	}

	for _, claim := range usernameClaimFallbacks {
		if claim == userIdentifier {
			continue
		}
		if username, ok := claimAsUsername(datamap, claim); ok {
			return username, nil
		}
	}

	return "", errors.New("failed to extract username from oauth resource")
}

func claimAsUsername(datamap map[string]any, claim string) (string, bool) {
	if claim == "" {
		return "", false
	}

	username, ok := datamap[claim].(string)
	if ok && username != "" {
		return username, true
	}

	if !ok {
		asFloat, ok := datamap[claim].(float64)
		if ok && asFloat != 0 {
			return strconv.Itoa(int(asFloat)), true
		}
	}

	return "", false
}

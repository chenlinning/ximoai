package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const desktopAuthorizationPath = "/desktop/authorize"

// IssueAuthorizationCodeForRedirect turns a validated browser continuation into
// the one-time code callback used by the desktop shell. Normal web redirects
// are left untouched so existing OAuth flows keep their current behavior.
func (s *DesktopSessionService) IssueAuthorizationCodeForRedirect(ctx context.Context, userID int64, redirectTo string) (string, bool, error) {
	continuation, handled, err := parseDesktopAuthorizationRedirect(redirectTo)
	if !handled || err != nil {
		return "", handled, err
	}

	grant, err := s.CreateAuthorizationCode(ctx, userID, continuation.Request)
	if err != nil {
		return "", true, err
	}
	callbackURL, err := buildDesktopAuthorizationCallbackURL(continuation.Request.RedirectURI, grant.Code, continuation.State)
	if err != nil {
		return "", true, err
	}
	return callbackURL, true, nil
}

type desktopAuthorizationContinuation struct {
	Request DesktopAuthorizationRequest
	State   string
}

func parseDesktopAuthorizationRedirect(raw string) (*desktopAuthorizationContinuation, bool, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.IsAbs() || u.Host != "" || u.Path != desktopAuthorizationPath {
		return nil, false, nil
	}

	query := u.Query()
	desktopKeys := []string{"code_challenge", "code_challenge_method", "device_jwk", "redirect_uri", "state"}
	hasDesktopRequest := false
	for _, key := range desktopKeys {
		if _, ok := query[key]; ok {
			hasDesktopRequest = true
			break
		}
	}
	if !hasDesktopRequest {
		return nil, false, nil
	}

	codeChallenge, err := desktopRedirectQueryValue(query, "code_challenge")
	if err != nil {
		return nil, true, err
	}
	method, err := desktopRedirectQueryValue(query, "code_challenge_method")
	if err != nil {
		return nil, true, err
	}
	encodedJWK, err := desktopRedirectQueryValue(query, "device_jwk")
	if err != nil {
		return nil, true, err
	}
	redirectURI, err := desktopRedirectQueryValue(query, "redirect_uri")
	if err != nil {
		return nil, true, err
	}
	state := ""
	if values, ok := query["state"]; ok {
		if len(values) != 1 {
			return nil, true, infraerrors.BadRequest("DESKTOP_REQUEST_INVALID", "desktop state is invalid")
		}
		state = values[0]
		if len(state) > 2048 {
			return nil, true, infraerrors.BadRequest("DESKTOP_REQUEST_INVALID", "desktop state is invalid")
		}
	}

	deviceJWK, err := decodeDesktopAuthorizationJWK(encodedJWK)
	if err != nil {
		return nil, true, err
	}
	return &desktopAuthorizationContinuation{
		Request: DesktopAuthorizationRequest{
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: method,
			DeviceJWK:           deviceJWK,
			RedirectURI:         redirectURI,
		},
		State: state,
	}, true, nil
}

func desktopRedirectQueryValue(query url.Values, key string) (string, error) {
	values, ok := query[key]
	if !ok || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", infraerrors.BadRequest("DESKTOP_REQUEST_INVALID", "desktop request is invalid")
	}
	return values[0], nil
}

func decodeDesktopAuthorizationJWK(encoded string) (DesktopPublicJWK, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) == 0 || len(raw) > 2048 {
		return DesktopPublicJWK{}, infraerrors.BadRequest("DESKTOP_DEVICE_KEY_INVALID", "device_jwk is invalid")
	}
	var jwk DesktopPublicJWK
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jwk); err != nil {
		return DesktopPublicJWK{}, infraerrors.BadRequest("DESKTOP_DEVICE_KEY_INVALID", "device_jwk is invalid")
	}
	if _, _, err := parseDesktopPublicJWK(jwk); err != nil {
		return DesktopPublicJWK{}, infraerrors.BadRequest("DESKTOP_DEVICE_KEY_INVALID", "device_jwk is invalid").WithCause(err)
	}
	return jwk, nil
}

func buildDesktopAuthorizationCallbackURL(rawRedirectURI, code, state string) (string, error) {
	redirectURI, err := normalizeDesktopRedirectURI(rawRedirectURI)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", infraerrors.BadRequest("DESKTOP_REDIRECT_URI_INVALID", "redirect_uri is invalid")
	}
	query := u.Query()
	query.Set("code", code)
	if state != "" {
		query.Set("state", state)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

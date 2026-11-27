package platform

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// OAuthAuthorizationLink validates the same-origin WhoDB browser handoff.
func OAuthAuthorizationLink(host string, data json.RawMessage) (string, error) {
	const handoffPath = "/api/secrets/oauth/handoff"
	invalid := errors.New("platform returned an invalid OAuth browser handoff")
	var link string
	if err := json.Unmarshal(data, &link); err != nil {
		return "", invalid
	}
	base, err := url.Parse(host)
	if err != nil || base.Host == "" || (base.Scheme != "https" && (base.Scheme != "http" || !isLoopbackHostname(base.Hostname()))) {
		return "", invalid
	}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != base.Scheme || !strings.EqualFold(u.Host, base.Host) || u.User != nil || u.EscapedPath() != handoffPath || u.Fragment != "" {
		return "", invalid
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q) != 1 || len(q["state"]) != 1 || q.Get("state") == "" || len(q.Get("state")) > 4096 {
		return "", invalid
	}
	state := q.Get("state")
	decoded, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != state {
		return "", invalid
	}
	return u.String(), nil
}

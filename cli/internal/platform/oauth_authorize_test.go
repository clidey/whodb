package platform

import (
	"encoding/json"
	"testing"
)

func TestOAuthAuthorizationLinkRejectsForeignAndAmbiguousURLs(t *testing.T) {
	const host = "https://app.whodb.test"
	const path = "/api/secrets/oauth/handoff"
	for _, test := range []struct {
		link string
		ok   bool
	}{
		{host + path + "?state=Zm9v", true},
		{host + path + "?state=Zm9v&state=YmFy", false},
		{host + path + "?state=Zm9v&return_to=https://evil.test", false},
		{"https://evil.test" + path + "?state=Zm9v", false},
		{"https://user@app.whodb.test" + path + "?state=Zm9v", false},
		{"http://app.whodb.test" + path + "?state=Zm9v", false},
		{host + "/api/secrets/oauth/callback?state=Zm9v", false},
		{host + path + "?state=bad%0a", false},
		{host + path + "?state=Zm9v#fragment", false},
	} {
		data, err := json.Marshal(test.link)
		if err != nil {
			t.Fatal(err)
		}
		got, err := OAuthAuthorizationLink(host, data)
		if (err == nil) != test.ok || test.ok && got != test.link {
			t.Fatalf("OAuthAuthorizationLink(%q) = %q, %v", test.link, got, err)
		}
	}
}

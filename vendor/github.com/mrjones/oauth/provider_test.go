package oauth

import (
	"net/http"
	"testing"
)

func TestProviderIsAuthorizedGood(t *testing.T) {
	p := NewProvider(func(s string) (string, error) { return "consumersecret", nil })
	p.clock = &MockClock{Time: 1446226936}

	fakeRequest, err := http.NewRequest("GET", "https://example.com/some/path?q=query&q=another_query", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set header to good oauth1 header
	fakeRequest.Header.Set("Authorization", "OAuth oauth_nonce=\"799507437267152061446226936\", oauth_timestamp=\"1446226936\", oauth_version=\"1.0\", oauth_signature_method=\"HMAC-SHA1\", oauth_consumer_key=\"consumerkey\", oauth_signature=\"wNwcZEM4wZgCD5zvOA%2FYZ6Kl%2F8E%3D\"")

	authorized, err := p.IsAuthorized(fakeRequest)

	assertEq(t, err, nil)
	assertEq(t, *authorized, "consumerkey")
}

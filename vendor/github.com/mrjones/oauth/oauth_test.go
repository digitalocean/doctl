package oauth

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type Mocks struct {
	httpClient     *MockHttpClient
	clock          *MockClock
	nonceGenerator *MockNonceGenerator
	signer         *MockSigner
}

func newMocks(t *testing.T) *Mocks {
	return &Mocks{
		httpClient:     NewMockHttpClient(t),
		clock:          &MockClock{Time: 1},
		nonceGenerator: &MockNonceGenerator{Nonce: 2},
		signer: &MockSigner{
			ConsumerSecret: "consumersecret",
		},
	}
}

func (m *Mocks) install(c *Consumer) {
	c.HttpClient = m.httpClient
	c.clock = m.clock
	c.nonceGenerator = m.nonceGenerator
	c.signer = m.signer
}

func TestSuccessfulTokenRequest(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/requesttoken",
		map[string]string{
			"oauth_callback":         url.QueryEscape("http://www.mrjon.es/callback"),
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_version":          "1.0",
		},
		"oauth_token=TOKEN&oauth_token_secret=SECRET")

	token, url_, err := c.GetRequestTokenAndUrl("http://www.mrjon.es/callback")
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "TOKEN", token.Token)
	assertEq(t, "SECRET", token.Secret)
	assertEq(t, "consumersecret&", m.signer.UsedKey)
	assertEq(t, "http://www.mrjon.es/authorizetoken?oauth_token=TOKEN", url_)
}

func TestSpecialNetflixParams(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	c.AdditionalAuthorizationUrlParams = map[string]string{
		"application_name":   "UnitTest",
		"oauth_consumer_key": "consumerkey",
	}

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/requesttoken",
		map[string]string{
			"oauth_callback":         url.QueryEscape("http://www.mrjon.es/callback"),
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_version":          "1.0",
		},
		"oauth_token=TOKEN&oauth_token_secret=SECRET")

	token, url_, err := c.GetRequestTokenAndUrl("http://www.mrjon.es/callback")
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "TOKEN", token.Token)
	assertEq(t, "SECRET", token.Secret)
	assertEq(t, "consumersecret&", m.signer.UsedKey)

	parsedUrl, err := url.Parse(url_)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "UnitTest", parsedUrl.Query().Get("application_name"))
	assertEq(t, "consumerkey", parsedUrl.Query().Get("oauth_consumer_key"))
	assertEq(t, "TOKEN", parsedUrl.Query().Get("oauth_token"))

}

func TestSuccessfulTokenAuthorization(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/accesstoken",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "RTOKEN",
			"oauth_verifier":         "VERIFICATION_CODE",
			"oauth_version":          "1.0",
		},
		"oauth_token=ATOKEN&oauth_token_secret=ATOKEN_SECRET&oauth_session_handle=SESSION_HANDLE")

	rtoken := &RequestToken{Token: "RTOKEN", Secret: "RSECRET"}
	atoken, err := c.AuthorizeToken(rtoken, "VERIFICATION_CODE")
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "ATOKEN", atoken.Token)
	assertEq(t, "ATOKEN_SECRET", atoken.Secret)
	assertEq(t, "SESSION_HANDLE", atoken.AdditionalData["oauth_session_handle"])
	assertEq(t, "consumersecret&RSECRET", m.signer.UsedKey)
}

func TestSuccessfulTokenRefresh(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/accesstoken",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "ATOKEN",
			"oauth_session_handle":   "SESSION_HANDLE",
			"oauth_version":          "1.0",
		},
		"oauth_token=ATOKEN_REFRESHED&oauth_token_secret=ATOKEN_SECRET_REFRESHED&oauth_session_handle=SESSION_HANDLE")

	additionalData := map[string]string{
		SESSION_HANDLE_PARAM: "SESSION_HANDLE",
	}
	atoken := &AccessToken{Token: "ATOKEN", Secret: "ATOKEN_SECRET", AdditionalData: additionalData}
	atoken, err := c.RefreshToken(atoken)
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "ATOKEN_REFRESHED", atoken.Token)
	assertEq(t, "ATOKEN_SECRET_REFRESHED", atoken.Secret)
	assertEq(t, "SESSION_HANDLE", atoken.AdditionalData["oauth_session_handle"])
	assertEq(t, "consumersecret&ATOKEN_SECRET", m.signer.UsedKey)
}

func TestSuccessfulAuthorizedGet_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	authedClient, err := c.MakeHttpClient(token)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := authedClient.Get("http://www.mrjon.es/someurl?key=val")

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)
	assertEq(t, "GET&http%3A%2F%2Fwww.mrjon.es%2Fsomeurl&key%3Dval%26oauth_consumer_key%3Dconsumerkey%26oauth_nonce%3D2%26oauth_signature_method%3DHMAC-SHA1%26oauth_timestamp%3D1%26oauth_token%3DTOKEN%26oauth_version%3D1.0", m.signer.SignedString)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedGet_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", map[string]string{"key": "val"}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedGetWithAddlHdrs_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGetWithHeaders(
		"http://www.mrjon.es/someurl?key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Accept": {"json"},
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	authedClient, err := c.MakeHttpClient(token)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "http://www.mrjon.es/someurl?key=val", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "json")

	resp, err := authedClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedGetWithAddlHdrs_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGetWithHeaders(
		"http://www.mrjon.es/someurl?key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Accept": {"json"},
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	c.AdditionalHeaders = map[string][]string{
		"Accept": {"json"},
	}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", map[string]string{"key": "val"}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedPost_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectPost(
		"http://www.mrjon.es/someurl",
		"key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	authedClient, err := c.MakeHttpClient(token)
	if err != nil {
		t.Fatal(err)
	}

	vals := url.Values{}
	vals.Add("key", "val")
	resp, err := authedClient.PostForm("http://www.mrjon.es/someurl", vals)
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, int64(7), m.httpClient.lastRequest.ContentLength)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedPost_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectPost(
		"http://www.mrjon.es/someurl",
		"key=val",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Post(
		"http://www.mrjon.es/someurl", map[string]string{"key": "val"}, token)

	assertEq(t, int64(7), m.httpClient.lastRequest.ContentLength)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedJsonPost_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectPost(
		"http://www.mrjon.es/someurl",
		`{"key":"value"}`,
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	authedClient, err := c.MakeHttpClient(token)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "http://www.mrjon.es/someurl", strings.NewReader(`{"key":"value"}`))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := authedClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedJsonPost_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectPost(
		"http://www.mrjon.es/someurl",
		`{"key":"value"}`,
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.PostJson(
		"http://www.mrjon.es/someurl", `{"key":"value"}`, token)

	assertEq(t, int64(15), m.httpClient.lastRequest.ContentLength)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedXMLPost_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	x := `<?xml version="1.0" encoding="utf-8"?><main><node>test</node></main>`

	m.httpClient.ExpectPost(
		"http://www.mrjon.es/someurl",
		x,
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"application/xml"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.PostXML(
		"http://www.mrjon.es/someurl", x, token)

	assertEq(t, int64(len(x)), m.httpClient.lastRequest.ContentLength)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func TestSuccessfulAuthorizedMultipartPost_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	payload := "A bunch of data"

	/*
		expectedBody :=
			"--UNITTESTBOUNDARY\n" +
			"Content-Disposition: form-data; name=\"multipartname\"; filename=\"/no/matter\n" +
			"Content-Type: application/octet-stream\n" +
			"A bunch of data\n" +
			"\n" +
			"--UNITTESTBOUNDARY--\n";
	*/
	m.httpClient.ExpectPost(
		"http://www.mrjon.es/unittest",
		"", //expectedBody,
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		map[string][]string{
			"Content-Type": []string{"multipart/form-data; boundary=UNITTESTBOUNDARY"},
		},
		"RESPONSE_BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.PostMultipart(
		"http://www.mrjon.es/unittest", "multipartname", ioutil.NopCloser(strings.NewReader(payload)), map[string]string{}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "RESPONSE_BODY:SUCCESS", string(body))
}

func Test404OnTokenRequest(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ReturnStatusCode(404, "Not Found")

	_, _, err := c.GetRequestTokenAndUrl("callback")
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func Test404OnAuthorizationRequest(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ReturnStatusCode(404, "Not Found")

	rtoken := &RequestToken{Token: "RTOKEN", Secret: "RSECRET"}
	_, err := c.AuthorizeToken(rtoken, "VERIFICATION_CODE")
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func Test404OnTokenRefresh(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ReturnStatusCode(404, "Not Found")

	additionalData := map[string]string{
		SESSION_HANDLE_PARAM: "SESSION",
	}
	atoken := &AccessToken{Token: "ATOKEN", Secret: "ASECRET", AdditionalData: additionalData}
	_, err := c.RefreshToken(atoken)
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func Test404OnGet_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ReturnStatusCode(404, "Not Found")

	atoken := &AccessToken{Token: "ATOKEN", Secret: "ASECRET"}
	authedClient, err := c.MakeHttpClient(atoken)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := authedClient.Get("http://www.mrjon.es/someurl")
	if err != nil {
		t.Fatal("The new API doesn't explicitly return an error in this case")
	}

	if resp == nil {
		t.Fatal("Response shouldn't be nil")
	}

	assertEqM(t, 404, resp.StatusCode, "Response status code should equal the status code from HTTP response")
}

func Test404OnGet_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ReturnStatusCode(404, "Not Found")

	atoken := &AccessToken{Token: "ATOKEN", Secret: "ASECRET"}
	resp, err := c.Get("URL", map[string]string{}, atoken)
	if err == nil {
		t.Fatal("Should have raised an error")
	}

	assertEqM(t, 404, resp.StatusCode, "Response status code should equal the status code from HTTP response")
}

func TestMissingRequestTokenSecret(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/requesttoken",
		map[string]string{
			"oauth_callback":         url.QueryEscape("http://www.mrjon.es/callback"),
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_version":          "1.0",
		},
		"oauth_token=TOKEN") // Missing token_secret

	_, _, err := c.GetRequestTokenAndUrl("http://www.mrjon.es/callback")
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func TestMissingRequestToken(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/requesttoken",
		map[string]string{
			"oauth_callback":         url.QueryEscape("http://www.mrjon.es/callback"),
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_version":          "1.0",
		},
		"oauth_token_secret=SECRET") // Missing token

	_, _, err := c.GetRequestTokenAndUrl("http://www.mrjon.es/callback")
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func TestMissingSessionRefreshToken(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	additionalData := make(map[string]string) // missing SESSION_HANDLE_PARAM
	atoken := &AccessToken{Token: "ATOKEN", Secret: "SECRET", AdditionalData: additionalData}
	_, err := c.RefreshToken(atoken)
	if err == nil {
		t.Fatal("Should have raised an error")
	}
}

func TestCharacterEscaping_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?escapableChars=+%21%40%23%24%25%5E%26%2A%28%29%2B&nonEscapableChars=abcABC123-._~",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", map[string]string{
			"nonEscapableChars": "abcABC123-._~",
			"escapableChars":    " !@#$%^&*()+",
		}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)
	if !strings.Contains(m.signer.SignedString, "nonEscapableChars%3DabcABC123-._~") {
		t.Fatalf("Bad string to sign: '%s'", m.signer.SignedString)
	}

	if !strings.Contains(m.signer.SignedString, "escapableChars%3D%2520%2521%2540%2523%2524%2525%255E%2526%252A%2528%2529%252B") {
		t.Fatalf("Bad string to sign: '%s'", m.signer.SignedString)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestCharacterEscaping_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?escapableChars=+%21%40%23%24%25%5E%26%2A%28%29%2B&nonEscapableChars=abcABC123-._~",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", map[string]string{
			"nonEscapableChars": "abcABC123-._~",
			"escapableChars":    " !@#$%^&*()+",
		}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestGetWithNilParams_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl",
		nil,
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", nil, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSemicolonInParameters_OldApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?foo=1%3B2",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	resp, err := c.Get(
		"http://www.mrjon.es/someurl", map[string]string{
			"foo": "1;2",
		}, token)

	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func TestSemicolonInParameters_NewApi(t *testing.T) {
	c := basicConsumer()
	m := newMocks(t)
	m.install(c)

	m.httpClient.ExpectGet(
		"http://www.mrjon.es/someurl?foo=1%3B2",
		map[string]string{
			"oauth_consumer_key":     "consumerkey",
			"oauth_nonce":            "2",
			"oauth_signature":        "MOCK_SIGNATURE",
			"oauth_signature_method": "HMAC-SHA1",
			"oauth_timestamp":        "1",
			"oauth_token":            "TOKEN",
			"oauth_version":          "1.0",
		},
		"BODY:SUCCESS")

	token := &AccessToken{Token: "TOKEN", Secret: "SECRET"}

	authedClient, err := c.MakeHttpClient(token)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := authedClient.Get("http://www.mrjon.es/someurl?foo=1%3B2")
	if err != nil {
		t.Fatal(err)
	}

	assertEq(t, "consumersecret&SECRET", m.signer.UsedKey)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertEq(t, "BODY:SUCCESS", string(body))
}

func basicConsumer() *Consumer {
	return NewConsumer(
		"consumerkey",
		"consumersecret",
		ServiceProvider{
			RequestTokenUrl:   "http://www.mrjon.es/requesttoken",
			AuthorizeTokenUrl: "http://www.mrjon.es/authorizetoken",
			AccessTokenUrl:    "http://www.mrjon.es/accesstoken",
		})
}

func assertEq(t *testing.T, expected interface{}, actual interface{}) {
	assertEqM(t, expected, actual, "")
}

func assertEqM(t *testing.T, expected interface{}, actual interface{}, msg string) {
	if expected != actual {
		t.Fatalf("Assertion error.\n\tExpected: '%s'\n\tActual:   '%s'\n\tMessage:  '%s'",
			expected, actual, msg)
	}
}

type MockHttpClient struct {
	// Validate the request
	expectedUrl         string
	expectedRequestBody string
	expectedMethod      string
	expectedHeaders     map[string][]string
	oAuthChecker        *OAuthChecker
	lastRequest         *http.Request

	// Return the mocked response
	responseBody string
	statusCode   int

	t *testing.T
}

func NewMockHttpClient(t *testing.T) *MockHttpClient {
	return &MockHttpClient{t: t, statusCode: 200}
}

func (mock *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	mock.lastRequest = req

	if mock.expectedMethod != "" {
		assertEqM(mock.t, mock.expectedMethod, req.Method, "Unexpected HTTP method")
	}

	if mock.expectedRequestBody != "" {
		actualBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			mock.t.Fatal(err)
		}
		assertEqM(mock.t, mock.expectedRequestBody, string(actualBody), "Unexpected HTTP body")
	}

	if mock.expectedUrl != "" && req.URL.String() != mock.expectedUrl {
		mock.t.Fatalf("URLs did not match.\nExpected: '%s'\nActual: '%s'",
			mock.expectedUrl, req.URL.String())
	}
	if mock.oAuthChecker != nil {
		if req.Header == nil {
			mock.t.Fatal("Missing 'Authorization' header.")
		}
		mock.oAuthChecker.CheckHeader(req.Header.Get("Authorization"))
	}

	if len(mock.expectedHeaders) > 0 {
		for hk, hvals := range mock.expectedHeaders {

			for _, hval := range hvals {
				found := false

				for k, vals := range req.Header {
					for _, val := range vals {
						if k == hk && val == hval {
							found = true
						}
					}
				}
				if found == false {
					mock.t.Fatalf("Expected header %q to contain %q but it did not. %v", hk, hval, req.Header.Get(hk))
				}
			}
		}
	}
	return &http.Response{
			StatusCode: mock.statusCode,
			Body:       NewMockBody(mock.responseBody),
		},
		nil
}

func (mock *MockHttpClient) ExpectGet(expectedUrl string, expectedOAuthPairs map[string]string, responseBody string) {
	mock.expectedMethod = "GET"
	mock.expectedUrl = expectedUrl
	mock.oAuthChecker = NewOAuthChecker(mock.t, expectedOAuthPairs)

	mock.responseBody = responseBody
}

func (mock *MockHttpClient) ExpectGetWithHeaders(expectedUrl string, expectedOAuthPairs map[string]string, headers map[string][]string, responseBody string) {
	mock.expectedMethod = "GET"
	mock.expectedUrl = expectedUrl
	mock.expectedHeaders = headers
	mock.oAuthChecker = NewOAuthChecker(mock.t, expectedOAuthPairs)
	mock.responseBody = responseBody
}

func (mock *MockHttpClient) ExpectPost(expectedUrl string, expectedRequestBody string, expectedOAuthPairs map[string]string, expectedHeaders map[string][]string, responseBody string) {
	mock.expectedMethod = "POST"
	mock.expectedUrl = expectedUrl
	mock.expectedRequestBody = expectedRequestBody
	mock.expectedHeaders = expectedHeaders
	mock.oAuthChecker = NewOAuthChecker(mock.t, expectedOAuthPairs)

	mock.responseBody = responseBody
}

func (mock *MockHttpClient) ReturnStatusCode(statusCode int, body string) {
	mock.expectedMethod = "GET"
	mock.statusCode = statusCode
	mock.responseBody = body
}

type OAuthChecker struct {
	headerPairs map[string]string
	t           *testing.T
}

func NewOAuthChecker(t *testing.T, headerPairs map[string]string) *OAuthChecker {
	return &OAuthChecker{
		headerPairs: headerPairs,
		t:           t,
	}
}

func (o *OAuthChecker) CheckHeader(header string) {
	assertEqM(o.t, "OAuth ", header[0:6], "OAuth Header did not begin correctly.")
	paramsStr := header[6:]
	params := strings.Split(paramsStr, ",")
	paramMap := make(map[string]string)
	for _, param := range params {
		keyvalue := strings.Split(param, "=")
		// line looks like: key="value", strip off the quotes
		// TODO(mrjones): this is pretty hacky
		value := keyvalue[1]
		if strings.HasSuffix(value, ",") {
			value = value[0 : len(value)-1]
		}
		value = value[1 : len(value)-1]
		paramMap[keyvalue[0]] = value
	}
	for key, value := range o.headerPairs {
		assertEqM(o.t, value, paramMap[key], "For OAuth parameter "+key)
	}
}

type MockBody struct {
	reader io.Reader
}

func NewMockBody(body string) *MockBody {
	return &MockBody{
		reader: strings.NewReader(body),
	}
}

func (*MockBody) Close() error {
	return nil
}

func (mock *MockBody) Read(p []byte) (n int, err error) {
	return mock.reader.Read(p)
}

type MockClock struct {
	Time int64
}

func (m *MockClock) Seconds() int64 {
	return m.Time
}

func (m *MockClock) Nanos() int64 {
	return m.Time * 1e9
}

type MockNonceGenerator struct {
	Nonce int64
}

func (m *MockNonceGenerator) Int63() int64 {
	return m.Nonce
}

type MockSigner struct {
	UsedKey        string
	SignedString   string
	ConsumerSecret string
}

func (m *MockSigner) Sign(message string, tokenSecret string) (string, error) {
	m.UsedKey = m.ConsumerSecret + "&" + tokenSecret
	m.SignedString = message
	return "MOCK_SIGNATURE", nil
}

func (m *MockSigner) Debug(enabled bool) {}

func (m *MockSigner) SignatureMethod() string {
	return SIGNATURE_METHOD_HMAC_SHA1
}

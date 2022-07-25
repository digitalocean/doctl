/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/apache/openwhisk-client-go/wski18n"
)

const (
	AuthRequired                   = true
	NoAuth                         = false
	IncludeNamespaceInUrl          = true
	DoNotIncludeNamespaceInUrl     = false
	AppendOpenWhiskPathPrefix      = true
	DoNotAppendOpenWhiskPathPrefix = false
	EncodeBodyAsJson               = "json"
	EncodeBodyAsFormData           = "formdata"
	ProcessTimeOut                 = true
	DoNotProcessTimeOut            = false
	ExitWithErrorOnTimeout         = true
	ExitWithSuccessOnTimeout       = false
	DEFAULT_HTTP_TIMEOUT           = 30
)

type ClientInterface interface {
	NewRequestUrl(method string, urlRelResource *url.URL, body interface{}, includeNamespaceInUrl bool, appendOpenWhiskPath bool, encodeBodyAs string, useAuthentication bool) (*http.Request, error)
	NewRequest(method, urlStr string, body interface{}, includeNamespaceInUrl bool) (*http.Request, error)
	Do(req *http.Request, v interface{}, ExitWithErrorOnTimeout bool, secretToObfuscate ...ObfuscateSet) (*http.Response, error)
}

type TriggerServiceInterface interface {
	List(options *TriggerListOptions) ([]Trigger, *http.Response, error)
	Insert(trigger *Trigger, overwrite bool) (*Trigger, *http.Response, error)
	Get(triggerName string) (*Trigger, *http.Response, error)
	Delete(triggerName string) (*Trigger, *http.Response, error)
	Fire(triggerName string, payload interface{}) (*Trigger, *http.Response, error)
}

type Client struct {
	client *http.Client
	*Config
	Transport *http.Transport

	Sdks        *SdkService
	Triggers    TriggerServiceInterface
	Actions     *ActionService
	Rules       *RuleService
	Activations *ActivationService
	Packages    *PackageService
	Namespaces  *NamespaceService
	Info        *InfoService
	Apis        *ApiService
}

type Config struct {
	Namespace         string // NOTE :: Default is "_"
	Cert              string
	Key               string
	AuthToken         string
	Host              string
	BaseURL           *url.URL // NOTE :: Default is "openwhisk.ng.bluemix.net"
	Version           string
	Verbose           bool
	Debug             bool // For detailed tracing
	Insecure          bool
	UserAgent         string
	ApigwAccessToken  string
	ApigwTenantId     string
	AdditionalHeaders http.Header
}

type ObfuscateSet struct {
	Regex       string
	Replacement string
}

var DefaultObfuscateArr = []ObfuscateSet{
	{
		Regex:       "\"[Pp]assword\":\\s*\".*\"",
		Replacement: `"password": "******"`,
	},
}

// NewClient creates a new whisk client with the provided http client and whisk configuration.
//
// A new http.Transport will be created when client cert or TLS insecure options are set.
// If one use custom transport and want to keep it intact, please opt out TLS related fields
// in configInput and construct TLS configuration in the custom transport.
func NewClient(httpClient *http.Client, configInput *Config) (*Client, error) {

	var config *Config
	if configInput == nil {
		defaultConfig, err := GetDefaultConfig()
		if err != nil {
			return nil, err
		} else {
			config = defaultConfig
		}
	} else {
		config = configInput
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Second * DEFAULT_HTTP_TIMEOUT,
		}
	}

	var err error
	var errStr = ""
	if len(config.Host) == 0 {
		errStr = wski18n.T("Unable to create request URL, because OpenWhisk API host is missing")
	} else if config.BaseURL == nil {
		config.BaseURL, err = GetUrlBase(config.Host)
		if err != nil {
			Debug(DbgError, "Unable to create request URL, because the api host %s is invalid: %s\n", config.Host, err)
			errStr = wski18n.T("Unable to create request URL, because the api host '{{.host}}' is invalid: {{.err}}",
				map[string]interface{}{"host": config.Host, "err": err})
		}
	}

	if len(errStr) != 0 {
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}

	if len(config.Namespace) == 0 {
		config.Namespace = "_"
	}

	if len(config.Version) == 0 {
		config.Version = "v1"
	}

	if len(config.UserAgent) == 0 {
		config.UserAgent = "OpenWhisk-Go-Client " + runtime.GOOS + " " + runtime.GOARCH
	}

	c := &Client{
		client: httpClient,
		Config: config,
	}

	c.Sdks = &SdkService{client: c}
	c.Triggers = &TriggerService{client: c}
	c.Actions = &ActionService{client: c}
	c.Rules = &RuleService{client: c}
	c.Activations = &ActivationService{client: c}
	c.Packages = &PackageService{client: c}
	c.Namespaces = &NamespaceService{client: c}
	c.Info = &InfoService{client: c}
	c.Apis = &ApiService{client: c}

	werr := c.LoadX509KeyPair()
	if werr != nil {
		return nil, werr
	}

	return c, nil
}

func (c *Client) LoadX509KeyPair() error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.Config.Insecure,
	}

	if c.Config.Cert != "" && c.Config.Key != "" {
		if cert, err := ReadX509KeyPair(c.Config.Cert, c.Config.Key); err == nil {
			tlsConfig.Certificates = []tls.Certificate{cert}
		} else {
			errStr := wski18n.T("Unable to load the X509 key pair due to the following reason: {{.err}}",
				map[string]interface{}{"err": err})
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return werr
		}
	} else if !c.Config.Insecure {
		if c.Config.Cert == "" {
			warningStr := "The Cert file is not configured. Please configure the missing Cert file, if there is a security issue accessing the service.\n"
			Debug(DbgWarn, warningStr)
			if c.Config.Key != "" {
				errStr := wski18n.T("The Cert file is not configured. Please configure the missing Cert file.\n")
				werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
				return werr
			}
		}
		if c.Config.Key == "" {
			warningStr := "The Key file is not configured. Please configure the missing Key file, if there is a security issue accessing the service.\n"
			Debug(DbgWarn, warningStr)
			if c.Config.Cert != "" {
				errStr := wski18n.T("The Key file is not configured. Please configure the missing Key file.\n")
				werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
				return werr
			}
		}
	}

	// Only replace the existing transport when a custom TLS configuration is needed
	if tlsConfig.InsecureSkipVerify || tlsConfig.Certificates != nil {
		if c.client.Transport != nil {
			warningStr := "The provided http.Transport is replaced to match the TLS configuration. Custom transport cannot coexist with nondefault TLS configuration"
			Debug(DbgWarn, warningStr)
		}
		// Use the defaultTransport as the transport basis to maintain proxy support
		c.client.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       tlsConfig,
		}
	}

	return nil
}

var ReadX509KeyPair = func(certFile, keyFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, keyFile)
}

///////////////////////////////
// Request/Utility Functions //
///////////////////////////////

func (c *Client) NewRequest(method, urlStr string, body interface{}, includeNamespaceInUrl bool) (*http.Request, error) {
	if includeNamespaceInUrl {
		if c.Config.Namespace != "" {
			urlStr = fmt.Sprintf("%s/namespaces/%s/%s", c.Config.Version, c.Config.Namespace, urlStr)
		} else {
			urlStr = fmt.Sprintf("%s/namespaces", c.Config.Version)
		}
	} else {
		urlStr = fmt.Sprintf("%s/%s", c.Config.Version, urlStr)
	}

	urlStr = fmt.Sprintf("%s/%s", c.BaseURL.String(), urlStr)
	u, err := url.Parse(urlStr)
	if err != nil {
		Debug(DbgError, "url.Parse(%s) error: %s\n", urlStr, err)
		errStr := wski18n.T("Invalid request URL '{{.url}}': {{.err}}",
			map[string]interface{}{"url": urlStr, "err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(body)

		if err != nil {
			Debug(DbgError, "json.Encode(%#v) error: %s\n", body, err)
			errStr := wski18n.T("Error encoding request body: {{.err}}", map[string]interface{}{"err": err})
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return nil, werr
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		Debug(DbgError, "http.NewRequest(%v, %s, buf) error: %s\n", method, u.String(), err)
		errStr := wski18n.T("Error initializing request: {{.err}}", map[string]interface{}{"err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}
	if req.Body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	err = c.addAuthHeader(req, AuthRequired)
	if err != nil {
		Debug(DbgError, "addAuthHeader() error: %s\n", err)
		errStr := wski18n.T("Unable to add the HTTP authentication header: {{.err}}",
			map[string]interface{}{"err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}

	req.Header.Add("User-Agent", c.Config.UserAgent)

	for key := range c.Config.AdditionalHeaders {
		req.Header.Add(key, c.Config.AdditionalHeaders.Get(key))
	}

	return req, nil
}

func (c *Client) addAuthHeader(req *http.Request, authRequired bool) error {
	// Allow for authorization override via Additional Headers
	authHeaderValue := c.Config.AdditionalHeaders.Get("Authorization")
	if authHeaderValue != "" {
		Debug(DbgInfo, "Using additional header authorization\n")
	} else if c.Config.AuthToken != "" {
		encodedAuthToken := base64.StdEncoding.EncodeToString([]byte(c.Config.AuthToken))
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodedAuthToken))
		Debug(DbgInfo, "Adding basic auth header; using authkey\n")
	} else {
		if authRequired {
			Debug(DbgError, "The required authorization key is not configured - neither set as a property nor set via the --auth CLI argument\n")
			errStr := wski18n.T("Authorization key is not configured (--auth is required)")
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_USAGE, DISPLAY_MSG, DISPLAY_USAGE)
			return werr
		}
	}
	return nil
}

// bodyTruncator limits the size of Req/Resp Body for --verbose ONLY.
// It returns truncated Req/Resp Body, reloaded io.ReadCloser and any errors.
func BodyTruncator(body io.ReadCloser) (string, io.ReadCloser, error) {
	limit := 1000 // 1000 byte limit, anything over is truncated

	data, err := ioutil.ReadAll(body)
	if err != nil {
		Verbose("ioutil.ReadAll(req.Body) error: %s\n", err)
		werr := MakeWskError(err, EXIT_CODE_ERR_NETWORK, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return "", body, werr
	}

	reload := ioutil.NopCloser(bytes.NewBuffer(data))

	if len(data) > limit {
		Verbose("Body exceeds %d bytes and will be truncated\n", limit)
		newData := string(data)[:limit] + "..."
		return string(newData), reload, nil
	}

	return string(data), reload, nil
}

// Do sends an API request and returns the API response.  The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred.  If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
func (c *Client) Do(req *http.Request, v interface{}, ExitWithErrorOnTimeout bool, secretToObfuscate ...ObfuscateSet) (*http.Response, error) {
	var err error
	var data []byte
	secrets := append(DefaultObfuscateArr, secretToObfuscate...)

	req, err = PrintRequestInfo(req, secrets...)
	//Putting this based on previous code
	if err != nil {
		return nil, err
	}

	// Issue the request to the Whisk server endpoint
	resp, err := c.client.Do(req)
	if err != nil {
		Debug(DbgError, "HTTP Do() [req %s] error: %s\n", req.URL.String(), err)
		werr := MakeWskError(err, EXIT_CODE_ERR_NETWORK, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}

	resp, data, err = PrintResponseInfo(resp, secrets...)
	if err != nil {
		return resp, err
	}

	// With the HTTP response status code and the HTTP body contents,
	// the possible response scenarios are:
	//
	// 0. HTTP Success + Body indicating a whisk failure result
	// 1. HTTP Success + Valid body matching request expectations
	// 2. HTTP Success + No body expected
	// 3. HTTP Success + Body does NOT match request expectations
	// 4. HTTP Failure + No body
	// 5. HTTP Failure + Body matching error format expectation
	// 6. HTTP Failure + Body NOT matching error format expectation

	// Handle 4. HTTP Failure + No body
	// If this happens, just return no data and an error
	if !IsHttpRespSuccess(resp) && data == nil {
		Debug(DbgError, "HTTP failure %d + no body\n", resp.StatusCode)
		werr := MakeWskError(errors.New(wski18n.T("Command failed due to an HTTP failure")), resp.StatusCode-256,
			DISPLAY_MSG, NO_DISPLAY_USAGE)
		return resp, werr
	}

	// Handle 5. HTTP Failure + Body matching error format expectation, or body matching a whisk.error() response
	// Handle 6. HTTP Failure + Body NOT matching error format expectation
	if !IsHttpRespSuccess(resp) && data != nil {
		return parseErrorResponse(resp, data, v)
	}

	// Handle 0. HTTP Success + Body indicating a whisk failure result
	//   NOTE: Need to ignore activation records send in response to 'wsk get activation NNN` as
	//         these will report the same original error giving the appearance that the command failed.
	//         Need to ignore `wsk action invoke NNN --result` too, otherwise action whose result is sth likes
	//         '{"response": {"key": "value"}}' will return an error to such command.
	if IsHttpRespSuccess(resp) && // HTTP Status == 200
		data != nil && // HTTP response body exists
		v != nil &&
		!strings.Contains(reflect.TypeOf(v).String(), "Activation") && // Request is not `wsk activation get`
		!(req.URL.Query().Get("result") == "true") && // Request is not `wsk action invoke NNN --result`
		!IsResponseResultSuccess(data) { // HTTP response body has Whisk error result
		Debug(DbgInfo, "Got successful HTTP; but activation response reports an error\n")
		return parseErrorResponse(resp, data, v)
	}

	// Handle 2. HTTP Success + No body expected
	if IsHttpRespSuccess(resp) && v == nil {
		Debug(DbgInfo, "No interface provided; no HTTP response body expected\n")
		return resp, nil
	}

	// Handle 1. HTTP Success + Valid body matching request expectations
	// Handle 3. HTTP Success + Body does NOT match request expectations
	if IsHttpRespSuccess(resp) && v != nil {

		// If a timeout occurs, 202 HTTP status code is returned, and the caller wishes to handle such an event, return
		// an error corresponding with the timeout
		if ExitWithErrorOnTimeout && resp.StatusCode == EXIT_CODE_TIMED_OUT {
			errMsg := wski18n.T("Request accepted, but processing not completed yet.")
			err = MakeWskError(errors.New(errMsg), EXIT_CODE_TIMED_OUT, NO_DISPLAY_MSG, NO_DISPLAY_USAGE,
				NO_MSG_DISPLAYED, NO_DISPLAY_PREFIX, NO_APPLICATION_ERR, TIMED_OUT)
		}

		return parseSuccessResponse(resp, data, v), err
	}

	// We should never get here, but just in case return failure to keep the compiler happy
	werr := MakeWskError(errors.New(wski18n.T("Command failed due to an internal failure")), EXIT_CODE_ERR_GENERAL,
		DISPLAY_MSG, NO_DISPLAY_USAGE)
	return resp, werr
}

func PrintRequestInfo(req *http.Request, secretToObfuscate ...ObfuscateSet) (*http.Request, error) {
	var truncatedBody string
	var err error
	if IsVerbose() {
		fmt.Println("REQUEST:")
		fmt.Printf("[%s]\t%s\n", req.Method, req.URL)

		if len(req.Header) > 0 {
			fmt.Println("Req Headers")
			PrintJSON(req.Header)
		}

		if req.Body != nil {
			fmt.Println("Req Body")
			// Since we're emptying out the reader, which is the req.Body, we have to reset it,
			// but create some copies for our debug messages.
			buffer, _ := ioutil.ReadAll(req.Body)
			obfuscatedRequest := ObfuscateText(string(buffer), secretToObfuscate)
			req.Body = ioutil.NopCloser(bytes.NewBuffer(buffer))

			if !IsDebug() {
				if truncatedBody, req.Body, err = BodyTruncator(ioutil.NopCloser(bytes.NewBuffer(buffer))); err != nil {
					return nil, err
				}
				fmt.Println(ObfuscateText(truncatedBody, secretToObfuscate))
			} else {
				fmt.Println(obfuscatedRequest)
			}
			Debug(DbgInfo, "Req Body (ASCII quoted string):\n%+q\n", obfuscatedRequest)
		}
	}
	return req, nil
}

func PrintResponseInfo(resp *http.Response, secretToObfuscate ...ObfuscateSet) (*http.Response, []byte, error) {
	var truncatedBody string
	// Don't "defer resp.Body.Close()" here because the body is reloaded to allow caller to
	// do custom body parsing, such as handling per-route error responses.
	Verbose("RESPONSE:")
	Verbose("Got response with code %d\n", resp.StatusCode)

	if IsVerbose() && len(resp.Header) > 0 {
		fmt.Println("Resp Headers")
		PrintJSON(resp.Header)
	}

	// Read the response body
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Debug(DbgError, "ioutil.ReadAll(resp.Body) error: %s\n", err)
		werr := MakeWskError(err, EXIT_CODE_ERR_NETWORK, DISPLAY_MSG, NO_DISPLAY_USAGE)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))
		return resp, data, werr
	}

	// Reload the response body to allow caller access to the body; otherwise,
	// the caller will have any empty body to read
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	Verbose("Response body size is %d bytes\n", len(data))

	if !IsDebug() {
		if truncatedBody, resp.Body, err = BodyTruncator(ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
			return nil, data, err
		}
		Verbose("Response body received:\n%s\n", ObfuscateText(truncatedBody, secretToObfuscate))
	} else {
		obfuscatedResponse := ObfuscateText(string(data), secretToObfuscate)
		Verbose("Response body received:\n%s\n", obfuscatedResponse)
		Debug(DbgInfo, "Response body received (ASCII quoted string):\n%+q\n", obfuscatedResponse)
	}
	return resp, data, err
}

func ObfuscateText(text string, replacements []ObfuscateSet) string {
	obfuscated := text
	for _, oSet := range replacements {
		r, _ := regexp.Compile(oSet.Regex)
		obfuscated = r.ReplaceAllString(obfuscated, oSet.Replacement)
	}
	return obfuscated
}

func parseErrorResponse(resp *http.Response, data []byte, v interface{}) (*http.Response, error) {
	Debug(DbgInfo, "HTTP failure %d + body\n", resp.StatusCode)

	// Determine if an application error was received (#5)
	buf := bytes.NewBuffer(data)
	d := json.NewDecoder(buf)
	d.UseNumber()

	errorResponse := &ErrorResponse{Response: resp}
	err := d.Decode(&errorResponse)

	// Determine if error is an application error or an error generated by API
	if err == nil {
		if errorResponse.Code == nil /*&& errorResponse.ErrMsg != nil */ && resp.StatusCode == 502 {
			return parseApplicationError(resp, data, v)
		} else if errorResponse.Code != nil && errorResponse.ErrMsg != nil {
			Debug(DbgInfo, "HTTP failure %d; server error %s\n", resp.StatusCode, errorResponse)
			werr := MakeWskError(errorResponse, resp.StatusCode-256, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return resp, werr
		}
	}

	// Body contents are unknown (#6)
	Debug(DbgError, "HTTP response with unexpected body failed due to contents parsing error: '%v'\n", err)
	errMsg := wski18n.T("The connection failed, or timed out. (HTTP status code {{.code}})",
		map[string]interface{}{"code": resp.StatusCode})
	whiskErr := MakeWskError(errors.New(errMsg), resp.StatusCode-256, DISPLAY_MSG, NO_DISPLAY_USAGE)
	return resp, whiskErr
}

func parseApplicationError(resp *http.Response, data []byte, v interface{}) (*http.Response, error) {
	Debug(DbgInfo, "Parsing application error\n")

	whiskErrorResponse := &WhiskErrorResponse{}
	err := json.Unmarshal(data, whiskErrorResponse)

	// Handle application errors that occur when --result option is false (#5)
	if err == nil && whiskErrorResponse != nil && whiskErrorResponse.Response != nil && whiskErrorResponse.Response.Status != nil {
		Debug(DbgInfo, "Detected response status `%s` that a whisk.error(\"%#v\") was returned\n",
			*whiskErrorResponse.Response.Status, whiskErrorResponse.Response.Result)

		errStr := getApplicationErrorMessage(whiskErrorResponse.Response.Result)
		Debug(DbgInfo, "Application error received: %s\n", errStr)

		errMsg := wski18n.T("The following application error was received: {{.err}}",
			map[string]interface{}{"err": errStr})
		whiskErr := MakeWskError(errors.New(errMsg), resp.StatusCode-256, NO_DISPLAY_MSG, NO_DISPLAY_USAGE,
			NO_MSG_DISPLAYED, DISPLAY_PREFIX, APPLICATION_ERR)
		return parseSuccessResponse(resp, data, v), whiskErr
	}

	appErrResult := &AppErrorResult{}
	err = json.Unmarshal(data, appErrResult)

	// Handle application errors that occur with blocking invocations when --result option is true (#5)
	if err == nil && appErrResult.Error != nil {
		Debug(DbgInfo, "Error code is null, blocking with result invocation error has occurred\n")
		errStr := getApplicationErrorMessage(*appErrResult.Error)
		Debug(DbgInfo, "Application error received: %s\n", errStr)

		whiskErr := MakeWskError(errors.New(errStr), resp.StatusCode-256, NO_DISPLAY_MSG, NO_DISPLAY_USAGE,
			NO_MSG_DISPLAYED, DISPLAY_PREFIX, APPLICATION_ERR)
		return parseSuccessResponse(resp, data, v), whiskErr
	}

	// Body contents are unknown (#6)
	Debug(DbgError, "HTTP response with unexpected body failed due to contents parsing error: '%v'\n", err)
	errMsg := wski18n.T("The connection failed, or timed out. (HTTP status code {{.code}})",
		map[string]interface{}{"code": resp.StatusCode})
	whiskErr := MakeWskError(errors.New(errMsg), resp.StatusCode-256, DISPLAY_MSG, NO_DISPLAY_USAGE)
	return resp, whiskErr
}

func getApplicationErrorMessage(errResp interface{}) string {
	var errStr string

	// Handle error results that looks like:
	//
	//   {
	//     "error": {
	//       "error": "An error string",
	//       "message": "An error message",
	//       "another-message": "Another error message"
	//     }
	//   }
	//   Returns "An error string; An error message; Another error message"
	//
	// OR
	//   {
	//     "error": "An error string"
	//   }
	//   Returns "An error string"
	//
	// OR
	//   {
	//     "error": {
	//       "custom-err": {
	//         "error": "An error string",
	//         "message": "An error message"
	//       }
	//     }
	//   }
	//   Returns "{"error": { "custom-err": { "error": "An error string", "message": "An error message" } } }"

	errMapIntf, errMapIntfOk := errResp.(map[string]interface{})
	if !errMapIntfOk {
		errStr = fmt.Sprintf("%v", errResp)
	} else {
		// Check if the "error" field in the response JSON
		errObjIntf, errObjIntfOk := errMapIntf["error"]
		if !errObjIntfOk {
			errStr = fmt.Sprintf("%v", errMapIntf)
		} else {
			// Check if the "error" field value is a JSON object
			errObj, errObjOk := errObjIntf.(map[string]interface{})
			if !errObjOk {
				// The "error" field value is not JSON; check if it's a string
				errorStr, errorStrOk := errObjIntf.(string)
				if !errorStrOk {
					errStr = fmt.Sprintf("%v", errObjIntf)
				} else {
					errStr = errorStr
				}
			} else {
				Debug(DbgInfo, "Application failure error json: %+v\n", errObj)

				// Concatenate all string field values into a single error string
				// Note: entry order of strings from map are not preserved.
				msgSeparator := ""
				for _, val := range errObj {
					valStr, valStrOk := val.(string)
					if valStrOk {
						errStr = errStr + msgSeparator + valStr
						msgSeparator = "; "
					}
				}

				// If no top level string fields exist, return the entire error object
				// Return a nice JSON string if possible; otherwise let Go try it's best
				if len(errStr) == 0 {
					jsonBytes, err := json.Marshal(errObj)
					if err != nil {
						errStr = fmt.Sprintf("%v", errObj)
					} else {
						errStr = string(jsonBytes)
					}
				}
			}
		}
	}

	return errStr
}
func parseSuccessResponse(resp *http.Response, data []byte, v interface{}) *http.Response {
	Debug(DbgInfo, "Parsing HTTP response into struct type: %s\n", reflect.TypeOf(v))

	dc := json.NewDecoder(strings.NewReader(string(data)))
	dc.UseNumber()
	err := dc.Decode(v)

	// If the decode was successful, return the response without error (#1). Otherwise, the decode did not work, so the
	// server response was unexpected (#3)
	if err == nil {
		Debug(DbgInfo, "Successful parse of HTTP response into struct type: %s\n", reflect.TypeOf(v))
		return resp
	} else {
		Debug(DbgWarn, "Unsuccessful parse of HTTP response into struct type: %s; parse error '%v'\n", reflect.TypeOf(v), err)
		Debug(DbgWarn, "Request was successful, so ignoring the following unexpected response body that could not be parsed: %s\n", data)
		return resp
	}
}

////////////
// Errors //
////////////

// For containing the server response body when an error message is returned
// Here's an example error response body with HTTP status code == 400
// {
//     "error": "namespace contains invalid characters",
//     "code": "1422870"
// }
type ErrorResponse struct {
	Response *http.Response `json:"-"`     // HTTP response that caused this error
	ErrMsg   *interface{}   `json:"error"` // error message string
	Code     *interface{}   `json:"code"`  // validation error code (tid)
}

type AppErrorResult struct {
	Error *interface{} `json:"error"`
}

type WhiskErrorResponse struct {
	Response *WhiskResponse `json:"response"`
}

type WhiskResponse struct {
	Result  map[string]interface{} `json:"result"`
	Success bool                   `json:"success"`
	Status  *interface{}           `json:"status"`
}

type WhiskResult struct {
	//  Error   *WhiskError     `json:"error"`  // whisk.error(<string>) and whisk.reject({msg:<string>}) result in two different kinds of 'error' JSON objects
}

type WhiskError struct {
	Msg *string `json:"msg"`
}

func (r ErrorResponse) Error() string {
	return wski18n.T("{{.msg}} (code {{.code}})",
		map[string]interface{}{"msg": fmt.Sprintf("%v", *r.ErrMsg), "code": r.Code})
}

////////////////////////////
// Basic Client Functions //
////////////////////////////

func IsHttpRespSuccess(r *http.Response) bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

func IsResponseResultSuccess(data []byte) bool {
	errResp := new(WhiskErrorResponse)
	err := json.Unmarshal(data, &errResp)

	if errResp.Response != nil {
		return errResp.Response.Success
	} else if err != nil { //failed to parse WhiskErrorResponse
		Debug(DbgWarn, "IsResponseResultSuccess: failed to parse response result: %v\n", err)
	}

	return true
}

//
// Create a HTTP request object using URL stored in url.URL object
// Arguments:
//   method         - HTTP verb (i.e. "GET", "PUT", etc)
//   urlRelResource - *url.URL structure representing the relative resource URL, including query params
//   body           - optional. Object whose contents will be JSON encoded and placed in HTTP request body
//   includeNamespaceInUrl - when true "/namespaces/NAMESPACE" is included in the final URL; otherwise not included.
//   appendOpenWhiskPath - when true, the OpenWhisk URL format is generated
//   encodeBodyAs   - specifies body encoding (json or form data)
//   useAuthentication - when true, the basic Authorization is included with the configured authkey as the value
func (c *Client) NewRequestUrl(
	method string,
	urlRelResource *url.URL,
	body interface{},
	includeNamespaceInUrl bool,
	appendOpenWhiskPath bool,
	encodeBodyAs string,
	useAuthentication bool) (*http.Request, error) {
	var requestUrl *url.URL
	var err error

	if appendOpenWhiskPath {
		var urlVerNamespaceStr string
		var verPathEncoded = (&url.URL{Path: c.Config.Version}).String()

		if includeNamespaceInUrl {
			if c.Config.Namespace != "" {
				// Encode path parts before inserting them into the URI so that any '?' is correctly encoded
				// as part of the path and not the start of the query params
				verNamespaceEncoded := (&url.URL{Path: c.Config.Namespace}).String()
				urlVerNamespaceStr = fmt.Sprintf("%s/namespaces/%s", verPathEncoded, verNamespaceEncoded)
			} else {
				urlVerNamespaceStr = fmt.Sprintf("%s/namespaces", verPathEncoded)
			}
		} else {
			urlVerNamespaceStr = fmt.Sprintf("%s", verPathEncoded)
		}

		// Assemble the complete URL: base + version + [namespace] + resource_relative_path
		Debug(DbgInfo, "basepath: %s, version/namespace path: %s, resource path: %s\n", c.BaseURL.String(), urlVerNamespaceStr, urlRelResource.String())
		urlStr := fmt.Sprintf("%s/%s/%s", c.BaseURL.String(), urlVerNamespaceStr, urlRelResource.String())
		requestUrl, err = url.Parse(urlStr)
		if err != nil {
			Debug(DbgError, "url.Parse(%s) error: %s\n", urlStr, err)
			errStr := wski18n.T("Invalid request URL '{{.url}}': {{.err}}",
				map[string]interface{}{"url": urlStr, "err": err})
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return nil, werr
		}
	} else {
		Debug(DbgInfo, "basepath: %s, resource path: %s\n", c.BaseURL.String(), urlRelResource.String())
		urlStr := fmt.Sprintf("%s/%s", c.BaseURL.String(), urlRelResource.String())
		requestUrl, err = url.Parse(urlStr)
		if err != nil {
			Debug(DbgError, "url.Parse(%s) error: %s\n", urlStr, err)
			errStr := wski18n.T("Invalid request URL '{{.url}}': {{.err}}",
				map[string]interface{}{"url": urlStr, "err": err})
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return nil, werr
		}
	}

	var buf io.ReadWriter
	if body != nil {
		if encodeBodyAs == EncodeBodyAsJson {
			buf = new(bytes.Buffer)
			encoder := json.NewEncoder(buf)
			encoder.SetEscapeHTML(false)
			err := encoder.Encode(body)

			if err != nil {
				Debug(DbgError, "json.Encode(%#v) error: %s\n", body, err)
				errStr := wski18n.T("Error encoding request body: {{.err}}",
					map[string]interface{}{"err": err})
				werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
				return nil, werr
			}
		} else if encodeBodyAs == EncodeBodyAsFormData {
			if values, ok := body.(url.Values); ok {
				buf = bytes.NewBufferString(values.Encode())
			} else {
				Debug(DbgError, "Invalid form data body: %v\n", body)
				errStr := wski18n.T("Internal error.  Form data encoding failure")
				werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
				return nil, werr
			}
		} else {
			Debug(DbgError, "Invalid body encode type: %s\n", encodeBodyAs)
			errStr := wski18n.T("Internal error.  Invalid encoding type '{{.encodetype}}'",
				map[string]interface{}{"encodetype": encodeBodyAs})
			werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return nil, werr
		}
	}

	req, err := http.NewRequest(method, requestUrl.String(), buf)
	if err != nil {
		Debug(DbgError, "http.NewRequest(%v, %s, buf) error: %s\n", method, requestUrl.String(), err)
		errStr := wski18n.T("Error initializing request: {{.err}}", map[string]interface{}{"err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}
	if req.Body != nil && encodeBodyAs == EncodeBodyAsJson {
		req.Header.Add("Content-Type", "application/json")
	}
	if req.Body != nil && encodeBodyAs == EncodeBodyAsFormData {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	if useAuthentication {
		err = c.addAuthHeader(req, AuthRequired)
		if err != nil {
			Debug(DbgError, "addAuthHeader() error: %s\n", err)
			errStr := wski18n.T("Unable to add the HTTP authentication header: {{.err}}",
				map[string]interface{}{"err": err})
			werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
			return nil, werr
		}
	} else {
		Debug(DbgInfo, "No auth header required\n")
	}

	req.Header.Add("User-Agent", c.Config.UserAgent)

	for key := range c.Config.AdditionalHeaders {
		req.Header.Add(key, c.Config.AdditionalHeaders.Get(key))
	}

	return req, nil
}

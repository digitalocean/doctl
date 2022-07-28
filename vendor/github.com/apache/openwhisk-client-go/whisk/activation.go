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
	"errors"
	"fmt"
	"github.com/apache/openwhisk-client-go/wski18n"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ActivationService struct {
	client *Client
}

type Activation struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Subject      string `json:"subject"`
	ActivationID string `json:"activationId"`
	Cause        string `json:"cause,omitempty"`
	Start        int64  `json:"start"`    // When action started (in milliseconds since January 1, 1970 UTC)
	End          int64  `json:"end"`      // Since a 0 is a valid value from server, don't omit
	Duration     int64  `json:"duration"` // Only available for actions
	StatusCode   int    `json:"statusCode"`
	Response     `json:"response"`
	Logs         []string    `json:"logs"`
	Annotations  KeyValueArr `json:"annotations"`
	Publish      *bool       `json:"publish,omitempty"`
}

type ActivationFilteredRow struct {
	Row       Activation
	HeaderFmt string
	RowFmt    string
}

type Response struct {
	Status     string  `json:"status"`
	StatusCode int     `json:"statusCode"`
	Success    bool    `json:"success"`
	Result     *Result `json:"result,omitempty"`
}

type Result map[string]interface{}

type ActivationListOptions struct {
	Name  string `url:"name,omitempty"`
	Limit int    `url:"limit"`
	Skip  int    `url:"skip"`
	Since int64  `url:"since,omitempty"`
	Upto  int64  `url:"upto,omitempty"`
	Docs  bool   `url:"docs,omitempty"`
}

//MWD - This structure may no longer be needed as the log format is now a string and not JSON
type Log struct {
	Log    string `json:"log,omitempty"`
	Stream string `json:"stream,omitempty"`
	Time   string `json:"time,omitempty"`
}

// Status codes to descriptions
var StatusCodes = []string{"success", "application error", "developer error", "internal error"}

// Compare(sortable) compares activation to sortable for the purpose of sorting.
// REQUIRED: sortable must also be of type Activation.
// ***Method of type Sortable***
// ***Currently, no method of sorting defined***
func (activation Activation) Compare(sortable Sortable) bool {
	return true
}

// Compare(sortable) compares activation to sortable for the purpose of sorting.
// REQUIRED: sortable must also be of type Activation.
// ***Method of type Sortable***
// ***Currently, no method of sorting defined***
func (activation ActivationFilteredRow) Compare(sortable Sortable) bool {
	return true
}

// ToHeaderString() returns the header for a list of activations
func (activation ActivationFilteredRow) ToHeaderString() string {
	return fmt.Sprintf(activation.HeaderFmt, "Datetime", "Activation ID", "Kind", "Start", "Duration", "Status", "Entity")
}

// TruncateStr() returns the string, truncated with ...in the middle if it exceeds the specified length
func TruncateStr(str string, maxlen int) string {
	if len(str) <= maxlen {
		return str
	} else {
		mid := maxlen / 2
		upp := len(str) - mid + 3
		if maxlen%2 != 0 {
			mid++
		}
		return str[0:mid] + "..." + str[upp:]
	}
}

// ToSummaryRowString() returns a compound string of required parameters for printing
//   from CLI command `wsk activation list`.
// ***Method of type Sortable***
func (activation ActivationFilteredRow) ToSummaryRowString() string {
	s := time.Unix(0, activation.Row.Start*1000000)
	e := time.Unix(0, activation.Row.End*1000000)

	var duration = e.Sub(s)
	var kind interface{} = activation.Row.Annotations.GetValue("kind")
	var initTime interface{} = activation.Row.Annotations.GetValue("initTime")
	var status = StatusCodes[0] // assume success
	var start = "warm"          // assume warm
	var fqn = TruncateStr(activation.Row.Namespace, 20) + "/" + TruncateStr(activation.Row.Name, 30) + ":" + TruncateStr(activation.Row.Version, 20)

	if activation.Row.Duration == 0 {
		duration = s.Sub(s)
	}
	if kind == nil {
		kind = "unknown"
	}
	if activation.Row.StatusCode > 0 && activation.Row.StatusCode < len(StatusCodes) {
		status = StatusCodes[activation.Row.StatusCode]
	}
	if initTime != nil {
		start = "cold"
	}

	return fmt.Sprintf(
		activation.RowFmt+strconv.Itoa(len(fqn))+"s\n",
		s.Year(), s.Month(), s.Day(), s.Hour(), s.Minute(), s.Second(),
		activation.Row.ActivationID,
		kind.(string),
		start,
		duration,
		status,
		fqn)
}

func (s *ActivationService) List(options *ActivationListOptions) ([]Activation, *http.Response, error) {
	// TODO :: for some reason /activations only works with "_" as namespace
	s.client.Namespace = "_"
	route := "activations"
	routeUrl, err := addRouteOptions(route, options)
	if err != nil {
		Debug(DbgError, "addRouteOptions(%s, %#v) error: '%s'\n", route, options, err)
		errStr := wski18n.T("Unable to append options '{{.options}}' to URL route '{{.route}}': {{.err}}",
			map[string]interface{}{"options": fmt.Sprintf("%#v", options), "route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	req, err := s.client.NewRequestUrl("GET", routeUrl, nil, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired)
	if err != nil {
		Debug(DbgError, "http.NewRequestUrl(GET, %s, nil, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired) error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	Debug(DbgInfo, "Sending HTTP request - URL '%s'; req %#v\n", req.URL.String(), req)

	var activations []Activation
	resp, err := s.client.Do(req, &activations, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return activations, resp, nil
}

func (s *ActivationService) Get(activationID string) (*Activation, *http.Response, error) {
	// TODO :: for some reason /activations/:id only works with "_" as namespace
	s.client.Namespace = "_"

	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	activationID = (&url.URL{Path: activationID}).String()
	route := fmt.Sprintf("activations/%s", activationID)

	req, err := s.client.NewRequest("GET", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(GET, %s) error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	Debug(DbgInfo, "Sending HTTP request - URL '%s'; req %#v\n", req.URL.String(), req)

	a := new(Activation)
	resp, err := s.client.Do(req, &a, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	a.StatusCode = GetStatusCodeForMessage(a.Status)

	return a, resp, nil
}

func GetStatusCodeForMessage(msg string) int {
	var code int

	switch msg {
	case "application error":
		code = 1
	case "action developer error":
		code = 2
	case "whisk internal error":
		code = 3
	default:
		code = 0
	}

	return code
}

func (s *ActivationService) Logs(activationID string) (*Activation, *http.Response, error) {
	// TODO :: for some reason /activations/:id/logs only works with "_" as namespace
	s.client.Namespace = "_"
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	activationID = (&url.URL{Path: activationID}).String()
	route := fmt.Sprintf("activations/%s/logs", activationID)

	req, err := s.client.NewRequest("GET", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(GET, %s) error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	Debug(DbgInfo, "Sending HTTP request - URL '%s'; req %#v\n", req.URL.String(), req)

	activation := new(Activation)
	resp, err := s.client.Do(req, &activation, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return activation, resp, nil
}

func (s *ActivationService) Result(activationID string) (*Response, *http.Response, error) {
	// TODO :: for some reason /activations only works with "_" as namespace
	s.client.Namespace = "_"
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	activationID = (&url.URL{Path: activationID}).String()
	route := fmt.Sprintf("activations/%s/result", activationID)

	req, err := s.client.NewRequest("GET", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(GET, %s) error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	Debug(DbgInfo, "Sending HTTP request - URL '%s'; req %#v\n", req.URL.String(), req)

	r := new(Response)
	resp, err := s.client.Do(req, &r, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return r, resp, nil
}

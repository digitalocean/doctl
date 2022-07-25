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
	"strings"
)

type RuleService struct {
	client *Client
}

type Rule struct {
	Namespace   string      `json:"namespace,omitempty"`
	Name        string      `json:"name,omitempty"`
	Version     string      `json:"version,omitempty"`
	Annotations KeyValueArr `json:"annotations,omitempty"`
	Status      string      `json:"status"`
	Trigger     interface{} `json:"trigger"`
	Action      interface{} `json:"action"`
	Publish     *bool       `json:"publish,omitempty"`
	Updated     int64       `json:"updated,omitempty"`
}

type RuleListOptions struct {
	Limit int  `url:"limit"`
	Skip  int  `url:"skip"`
	Docs  bool `url:"docs,omitempty"`
}

// Compare(sortable) compares rule to sortable for the purpose of sorting.
// REQUIRED: sortable must also be of type Rule.
// ***Method of type Sortable***
func (rule Rule) Compare(sortable Sortable) bool {
	// Sorts alphabetically by NAMESPACE -> PACKAGE_NAME
	ruleToCompare := sortable.(Rule)
	var ruleString string
	var compareString string

	ruleString = strings.ToLower(fmt.Sprintf("%s%s", rule.Namespace, rule.Name))
	compareString = strings.ToLower(fmt.Sprintf("%s%s", ruleToCompare.Namespace,
		ruleToCompare.Name))

	return ruleString < compareString
}

// ToHeaderString() returns the header for a list of rules
func (rule Rule) ToHeaderString() string {
	return fmt.Sprintf("%s\n", "rules")
}

// ToSummaryRowString() returns a compound string of required parameters for printing
//   from CLI command `wsk rule list`.
// ***Method of type Sortable***
func (rule Rule) ToSummaryRowString() string {
	publishState := wski18n.T("private")

	return fmt.Sprintf("%-70s %-20s %s\n", fmt.Sprintf("/%s/%s", rule.Namespace,
		rule.Name), publishState, rule.Status)
}

func (s *RuleService) List(options *RuleListOptions) ([]Rule, *http.Response, error) {
	route := "rules"
	routeUrl, err := addRouteOptions(route, options)
	if err != nil {
		Debug(DbgError, "addRouteOptions(%s, %#v) error: '%s'\n", route, options, err)
		errStr := wski18n.T("Unable to append options '{{.options}}' to URL route '{{.route}}': {{.err}}",
			map[string]interface{}{"options": fmt.Sprintf("%#v", options), "route": route, "err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	req, err := s.client.NewRequestUrl("GET", routeUrl, nil, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired)
	if err != nil {
		Debug(DbgError, "http.NewRequestUrl(GET, %s, nil, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	var rules []Rule
	resp, err := s.client.Do(req, &rules, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return rules, resp, err
}

func (s *RuleService) Insert(rule *Rule, overwrite bool) (*Rule, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	ruleName := (&url.URL{Path: rule.Name}).String()
	route := fmt.Sprintf("rules/%s?overwrite=%t", ruleName, overwrite)

	routeUrl, err := url.Parse(route)
	if err != nil {
		Debug(DbgError, "url.Parse(%s) error: %s\n", route, err)
		errStr := wski18n.T("Invalid request URL '{{.url}}': {{.err}}",
			map[string]interface{}{"url": route, "err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	req, err := s.client.NewRequestUrl("PUT", routeUrl, rule, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired)
	if err != nil {
		Debug(DbgError, "http.NewRequestUrl(PUT, %s, %+v, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired); error: '%s'\n", routeUrl, rule, err)
		errStr := wski18n.T("Unable to create HTTP request for PUT '{{.route}}': {{.err}}",
			map[string]interface{}{"route": routeUrl, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	r := new(Rule)
	resp, err := s.client.Do(req, &r, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return r, resp, nil
}

func (s *RuleService) Get(ruleName string) (*Rule, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	ruleName = (&url.URL{Path: ruleName}).String()
	route := fmt.Sprintf("rules/%s", ruleName)

	req, err := s.client.NewRequest("GET", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(GET, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	r := new(Rule)
	resp, err := s.client.Do(req, &r, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return r, resp, nil
}

func (s *RuleService) Delete(ruleName string) (*http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	ruleName = (&url.URL{Path: ruleName}).String()
	route := fmt.Sprintf("rules/%s", ruleName)

	req, err := s.client.NewRequest("DELETE", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(DELETE, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for DELETE '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, werr
	}

	resp, err := s.client.Do(req, nil, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return resp, err
	}

	return resp, nil
}

func (s *RuleService) SetState(ruleName string, state string) (*Rule, *http.Response, error) {
	state = strings.ToLower(state)
	if state != "active" && state != "inactive" {
		errStr := wski18n.T("Internal error. Invalid state option '{{.state}}'. Valid options are \"active\" and \"inactive\".",
			map[string]interface{}{"state": state})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, DISPLAY_USAGE)
		return nil, nil, werr
	}

	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	ruleName = (&url.URL{Path: ruleName}).String()
	route := fmt.Sprintf("rules/%s", ruleName)

	ruleState := &Rule{Status: state}

	req, err := s.client.NewRequest("POST", route, ruleState, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(POST, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for POST '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	r := new(Rule)
	resp, err := s.client.Do(req, &r, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return r, resp, nil
}

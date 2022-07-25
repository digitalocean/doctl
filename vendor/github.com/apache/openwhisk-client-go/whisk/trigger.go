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

type TriggerService struct {
	client ClientInterface
}

type Trigger struct {
	Namespace    string                 `json:"namespace,omitempty"`
	Name         string                 `json:"name,omityempty"`
	Version      string                 `json:"version,omitempty"`
	ActivationId string                 `json:"activationId,omitempty"`
	Annotations  KeyValueArr            `json:"annotations,omitempty"`
	Parameters   KeyValueArr            `json:"parameters,omitempty"`
	Limits       *Limits                `json:"limits,omitempty"`
	Publish      *bool                  `json:"publish,omitempty"`
	Rules        map[string]interface{} `json:"rules,omitempty"`
	Updated      int64                  `json:"updated,omitempty"`
}

type TriggerListOptions struct {
	Limit int  `url:"limit"`
	Skip  int  `url:"skip"`
	Docs  bool `url:"docs,omitempty"`
}

// Compare(sortable) compares trigger to sortable for the purpose of sorting.
// REQUIRED: sortable must also be of type Trigger.
// ***Method of type Sortable***
func (trigger Trigger) Compare(sortable Sortable) bool {
	// Sorts alphabetically by NAMESPACE -> TRIGGER_NAME
	triggerToCompare := sortable.(Trigger)
	var triggerString string
	var compareString string

	triggerString = strings.ToLower(fmt.Sprintf("%s%s", trigger.Namespace,
		trigger.Name))
	compareString = strings.ToLower(fmt.Sprintf("%s%s", triggerToCompare.Namespace,
		triggerToCompare.Name))

	return triggerString < compareString
}

// ToHeaderString() returns the header for a list of triggers
func (trigger Trigger) ToHeaderString() string {
	return fmt.Sprintf("%s\n", "triggers")
}

// ToSummaryRowString() returns a compound string of required parameters for printing
//   from CLI command `wsk trigger list`.
// ***Method of type Sortable***
func (trigger Trigger) ToSummaryRowString() string {
	publishState := wski18n.T("private")

	return fmt.Sprintf("%-70s %s\n", fmt.Sprintf("/%s/%s", trigger.Namespace,
		trigger.Name), publishState)
}

func (s *TriggerService) List(options *TriggerListOptions) ([]Trigger, *http.Response, error) {
	route := "triggers"
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

	var triggers []Trigger
	resp, err := s.client.Do(req, &triggers, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return triggers, resp, nil
}

func (s *TriggerService) Insert(trigger *Trigger, overwrite bool) (*Trigger, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	triggerName := (&url.URL{Path: trigger.Name}).String()
	route := fmt.Sprintf("triggers/%s?overwrite=%t", triggerName, overwrite)

	routeUrl, err := url.Parse(route)
	if err != nil {
		Debug(DbgError, "url.Parse(%s) error: %s\n", route, err)
		errStr := wski18n.T("Invalid request URL '{{.url}}': {{.err}}",
			map[string]interface{}{"url": route, "err": err})
		werr := MakeWskError(errors.New(errStr), EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	req, err := s.client.NewRequestUrl("PUT", routeUrl, trigger, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired)
	if err != nil {
		Debug(DbgError, "http.NewRequestUrl(PUT, %s, %+v, IncludeNamespaceInUrl, AppendOpenWhiskPathPrefix, EncodeBodyAsJson, AuthRequired); error: '%s'\n", routeUrl, trigger, err)
		errStr := wski18n.T("Unable to create HTTP request for PUT '{{.route}}': {{.err}}",
			map[string]interface{}{"route": routeUrl, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	t := new(Trigger)
	resp, err := s.client.Do(req, &t, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return t, resp, nil

}

func (s *TriggerService) Get(triggerName string) (*Trigger, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	triggerName = (&url.URL{Path: triggerName}).String()
	route := fmt.Sprintf("triggers/%s", triggerName)

	req, err := s.client.NewRequest("GET", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(GET, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	t := new(Trigger)
	resp, err := s.client.Do(req, &t, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return t, resp, nil

}

func (s *TriggerService) Delete(triggerName string) (*Trigger, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	triggerName = (&url.URL{Path: triggerName}).String()
	route := fmt.Sprintf("triggers/%s", triggerName)

	req, err := s.client.NewRequest("DELETE", route, nil, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, "http.NewRequest(DELETE, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for DELETE '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	t := new(Trigger)
	resp, err := s.client.Do(req, &t, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return t, resp, nil
}

func (s *TriggerService) Fire(triggerName string, payload interface{}) (*Trigger, *http.Response, error) {
	// Encode resource name as a path (with no query params) before inserting it into the URI
	// This way any '?' chars in the name won't be treated as the beginning of the query params
	triggerName = (&url.URL{Path: triggerName}).String()
	route := fmt.Sprintf("triggers/%s", triggerName)

	req, err := s.client.NewRequest("POST", route, payload, IncludeNamespaceInUrl)
	if err != nil {
		Debug(DbgError, " http.NewRequest(POST, %s); error: '%s'\n", route, err)
		errStr := wski18n.T("Unable to create HTTP request for POST '{{.route}}': {{.err}}",
			map[string]interface{}{"route": route, "err": err})
		werr := MakeWskErrorFromWskError(errors.New(errStr), err, EXIT_CODE_ERR_GENERAL, DISPLAY_MSG, NO_DISPLAY_USAGE)
		return nil, nil, werr
	}

	t := new(Trigger)
	resp, err := s.client.Do(req, &t, ExitWithSuccessOnTimeout)
	if err != nil {
		Debug(DbgError, "s.client.Do() error - HTTP req %s; error: '%s'\n", req.URL.String(), err)
		return nil, resp, err
	}

	return t, resp, nil
}

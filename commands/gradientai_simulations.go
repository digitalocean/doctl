/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
)

const (
	scenarioSetStatusPrefix        = "SCENARIO_SET_STATUS_"
	scenarioSetSourceKindPrefix    = "SCENARIO_SET_SOURCE_KIND_"
	scenarioSetSortFieldPrefix     = "SCENARIO_SET_SORT_FIELD_"
	scenarioSortFieldPrefix        = "SCENARIO_SORT_FIELD_"
	scenarioLibrarySortFieldPrefix = "SCENARIO_LIBRARY_SORT_FIELD_"
	simulationRunStatusPrefix      = "SIMULATION_RUN_STATUS_"
	simulationRunSortFieldPrefix   = "SIMULATION_RUN_SORT_FIELD_"
	journeyStatusPrefix            = "SIMULATION_JOURNEY_STATUS_"
	journeyVerdictPrefix           = "SIMULATION_JOURNEY_VERDICT_"
	journeySortFieldPrefix         = "SIMULATION_JOURNEY_SORT_FIELD_"
	sortDirectionPrefix            = "SORT_DIRECTION_"
)

// genAIEnumValue expands a user-supplied shorthand into the enum value the API
// expects, so that both `--statuses ready` and
// `--statuses SCENARIO_SET_STATUS_READY` are accepted.
func genAIEnumValue(prefix, value string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"))
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, prefix) {
		return normalized
	}
	return prefix + normalized
}

// genAIEnums expands each shorthand and converts the result to the enum type
// the list options expect.
func genAIEnums[T ~string](prefix string, values []string) []T {
	var out []T
	for _, value := range values {
		if expanded := genAIEnumValue(prefix, value); expanded != "" {
			out = append(out, T(expanded))
		}
	}
	return out
}

// genAIListFilters reads the search and sort flags shared by every Gradient AI
// simulation list command.
func genAIListFilters(c *CmdConfig, sortFieldPrefix string) (search string, sortBy string, sortDirection string, err error) {
	search, err = c.Doit.GetString(c.NS, doctl.ArgGenAISearch)
	if err != nil {
		return "", "", "", err
	}

	rawSortBy, err := c.Doit.GetString(c.NS, doctl.ArgGenAISortBy)
	if err != nil {
		return "", "", "", err
	}

	rawSortDirection, err := c.Doit.GetString(c.NS, doctl.ArgGenAISortDirection)
	if err != nil {
		return "", "", "", err
	}

	return search, genAIEnumValue(sortFieldPrefix, rawSortBy), genAIEnumValue(sortDirectionPrefix, rawSortDirection), nil
}

// scenarioListOptions builds the filters shared by the two scenario list endpoints.
func scenarioListOptions(c *CmdConfig) (*godo.ScenarioListOptions, error) {
	search, sortBy, sortDirection, err := genAIListFilters(c, scenarioSortFieldPrefix)
	if err != nil {
		return nil, err
	}

	return &godo.ScenarioListOptions{
		Search:        search,
		SortBy:        godo.ScenarioSortField(sortBy),
		SortDirection: godo.GenAISortDirection(sortDirection),
	}, nil
}

// parseScenarios unmarshals a JSON array of scenario objects.
func parseScenarios(raw string) ([]*godo.Scenario, error) {
	if raw == "" {
		return nil, nil
	}

	var scenarios []*godo.Scenario
	if err := json.Unmarshal([]byte(raw), &scenarios); err != nil {
		return nil, fmt.Errorf("unable to parse scenarios: %w", err)
	}
	return scenarios, nil
}

// addGenAIListFlags registers the search and sort flags shared by the list commands.
func addGenAIListFlags(cmd *Command, sortFields string) {
	AddStringFlag(cmd, doctl.ArgGenAISearch, "", "", "Filters the results by a free-text search term.")
	AddStringFlag(cmd, doctl.ArgGenAISortBy, "", "", "Sorts the results by a field. One of: "+sortFields)
	AddStringFlag(cmd, doctl.ArgGenAISortDirection, "", "", "Sorts the results in this direction. One of: `asc`, `desc`")
}

// uploadScenarioSetFile requests a presigned URL for path, uploads the file to
// it, and returns the data source that refers to the stored object.
func uploadScenarioSetFile(c *CmdConfig, path string) (*godo.FileUploadDataSource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, not a scenario file", path)
	}

	fileName := filepath.Base(path)
	size := strconv.FormatInt(info.Size(), 10)

	uploads, err := c.GradientAI().CreateScenarioSetUploadPresignedURLs(&godo.CreateScenarioSetUploadPresignedURLsRequest{
		Files: []*godo.PresignedUrlFile{{
			FileName: fileName,
			FileSize: size,
		}},
	})
	if err != nil {
		return nil, err
	}
	if uploads == nil || len(uploads.Uploads) == 0 {
		return nil, fmt.Errorf("no presigned upload URL was returned for %q", fileName)
	}

	upload := uploads.Uploads[0]
	if upload == nil || upload.PresignedURL == "" || upload.ObjectKey == "" {
		return nil, fmt.Errorf("the presigned upload response for %q was incomplete", fileName)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := putPresignedFile(upload.PresignedURL, file, info.Size()); err != nil {
		return nil, err
	}

	return &godo.FileUploadDataSource{
		OriginalFileName: fileName,
		Size:             size,
		StoredObjectKey:  upload.ObjectKey,
	}, nil
}

// putPresignedFile is a variable so that tests can upload without a live URL.
var putPresignedFile = func(url string, body io.Reader, size int64) error {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return err
	}
	req.ContentLength = size

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("uploading the scenario file failed with status %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}

	return nil
}

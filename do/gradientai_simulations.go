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

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

// Scenario wraps a godo.Scenario.
type Scenario struct {
	*godo.Scenario
}

// Scenarios is a slice of Scenario.
type Scenarios []Scenario

// ScenarioSet wraps a godo.ScenarioSet.
type ScenarioSet struct {
	*godo.ScenarioSet
}

// ScenarioSets is a slice of ScenarioSet.
type ScenarioSets []ScenarioSet

// GenAIDownloadURL is a presigned URL for downloading a Gradient AI file.
type GenAIDownloadURL struct {
	DownloadURL string          `json:"download_url,omitempty"`
	ExpiresAt   *godo.Timestamp `json:"expires_at,omitempty"`
}

// ScenarioSetFileUploads wraps a godo.CreateScenarioSetUploadPresignedURLsResponse.
type ScenarioSetFileUploads struct {
	*godo.CreateScenarioSetUploadPresignedURLsResponse
}

// ScenarioLibraryEntry wraps a godo.ScenarioLibraryEntry.
type ScenarioLibraryEntry struct {
	*godo.ScenarioLibraryEntry
}

// ScenarioLibraryEntries is a slice of ScenarioLibraryEntry.
type ScenarioLibraryEntries []ScenarioLibraryEntry

// SimulationRun wraps a godo.SimulationRun.
type SimulationRun struct {
	*godo.SimulationRun
}

// SimulationRuns is a slice of SimulationRun.
type SimulationRuns []SimulationRun

// SimulationRunDetail wraps a godo.SimulationRunGetResponse, which carries a
// simulation run together with its per-scenario result rollups.
type SimulationRunDetail struct {
	*godo.SimulationRunGetResponse
}

// SimulationJourney wraps a godo.SimulationJourney.
type SimulationJourney struct {
	*godo.SimulationJourney
}

// SimulationJourneys is a slice of SimulationJourney.
type SimulationJourneys []SimulationJourney

// SimulationTrajectory wraps a godo.SimulationTrajectory.
type SimulationTrajectory struct {
	*godo.SimulationTrajectory
}

// paginateGenAIList fetches every page of a Gradient AI list endpoint and wraps
// each item in its do type.
func paginateGenAIList[T any, W any](fetch func(*godo.ListOptions) ([]*T, *godo.Response, error), wrap func(*T) W) ([]W, error) {
	si, err := PaginateResp(func(listOpt *godo.ListOptions) ([]any, *godo.Response, error) {
		items, resp, err := fetch(listOpt)
		if err != nil {
			return nil, nil, err
		}
		out := make([]any, len(items))
		for i := range items {
			out[i] = items[i]
		}
		return out, resp, nil
	})
	if err != nil {
		return nil, err
	}

	list := make([]W, len(si))
	for i := range si {
		list[i] = wrap(si[i].(*T))
	}

	return list, nil
}

// CreateScenarioSetUploadPresignedURLs creates presigned URLs for uploading scenario set files.
func (a *gradientAIService) CreateScenarioSetUploadPresignedURLs(req *godo.CreateScenarioSetUploadPresignedURLsRequest) (*ScenarioSetFileUploads, error) {
	uploads, _, err := a.client.GradientAI.CreateScenarioSetUploadPresignedURLs(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &ScenarioSetFileUploads{CreateScenarioSetUploadPresignedURLsResponse: uploads}, nil
}

// CreateScenarioSet creates a scenario set from inline scenarios or an uploaded file.
func (a *gradientAIService) CreateScenarioSet(req *godo.CreateScenarioSetRequest) (*ScenarioSet, error) {
	set, _, err := a.client.GradientAI.CreateScenarioSet(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &ScenarioSet{ScenarioSet: set}, nil
}

// GenerateScenarioSet dispatches goal-driven scenario generation.
func (a *gradientAIService) GenerateScenarioSet(req *godo.GenerateScenarioSetRequest) (*ScenarioSet, error) {
	set, _, err := a.client.GradientAI.GenerateScenarioSet(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &ScenarioSet{ScenarioSet: set}, nil
}

// ListScenarioSets lists all scenario sets for the team.
func (a *gradientAIService) ListScenarioSets(opt *godo.ScenarioSetListOptions) (ScenarioSets, error) {
	if opt == nil {
		opt = &godo.ScenarioSetListOptions{}
	}

	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.ScenarioSet, *godo.Response, error) {
		filters := *opt
		filters.ListOptions = *listOpt
		res, resp, err := a.client.GradientAI.ListScenarioSets(context.TODO(), &filters)
		if err != nil {
			return nil, nil, err
		}
		return res.ScenarioSets, resp, nil
	}, func(set *godo.ScenarioSet) ScenarioSet {
		return ScenarioSet{ScenarioSet: set}
	})
}

// GetScenarioSet retrieves a scenario set by its UUID.
func (a *gradientAIService) GetScenarioSet(scenarioSetUUID string) (*ScenarioSet, error) {
	set, _, err := a.client.GradientAI.GetScenarioSet(context.TODO(), scenarioSetUUID)
	if err != nil {
		return nil, err
	}
	return &ScenarioSet{ScenarioSet: set}, nil
}

// ListScenarios lists all scenarios within a scenario set.
func (a *gradientAIService) ListScenarios(scenarioSetUUID string, opt *godo.ScenarioListOptions) (Scenarios, error) {
	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.Scenario, *godo.Response, error) {
		res, resp, err := a.client.GradientAI.ListScenarios(context.TODO(), scenarioSetUUID, scenarioFilters(opt, listOpt))
		if err != nil {
			return nil, nil, err
		}
		return res.Scenarios, resp, nil
	}, wrapScenario)
}

// GetScenarioSetDownloadURL returns a presigned download URL for a scenario set's file.
func (a *gradientAIService) GetScenarioSetDownloadURL(scenarioSetUUID string) (*GenAIDownloadURL, error) {
	res, _, err := a.client.GradientAI.GetScenarioSetDownloadURL(context.TODO(), scenarioSetUUID)
	if err != nil {
		return nil, err
	}
	return &GenAIDownloadURL{DownloadURL: res.DownloadURL, ExpiresAt: res.ExpiresAt}, nil
}

// UpdateScenarioSet updates a scenario set by its UUID.
func (a *gradientAIService) UpdateScenarioSet(scenarioSetUUID string, req *godo.UpdateScenarioSetRequest) (*ScenarioSet, error) {
	set, _, err := a.client.GradientAI.UpdateScenarioSet(context.TODO(), scenarioSetUUID, req)
	if err != nil {
		return nil, err
	}
	return &ScenarioSet{ScenarioSet: set}, nil
}

// DeleteScenarioSet deletes a scenario set by its UUID.
func (a *gradientAIService) DeleteScenarioSet(scenarioSetUUID string) error {
	_, _, err := a.client.GradientAI.DeleteScenarioSet(context.TODO(), scenarioSetUUID)
	return err
}

// ListScenarioLibrary lists all platform-curated scenario library entries.
func (a *gradientAIService) ListScenarioLibrary(opt *godo.ScenarioLibraryListOptions) (ScenarioLibraryEntries, error) {
	if opt == nil {
		opt = &godo.ScenarioLibraryListOptions{}
	}

	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.ScenarioLibraryEntry, *godo.Response, error) {
		filters := *opt
		filters.ListOptions = *listOpt
		res, resp, err := a.client.GradientAI.ListScenarioLibrary(context.TODO(), &filters)
		if err != nil {
			return nil, nil, err
		}
		return res.Scenarios, resp, nil
	}, func(entry *godo.ScenarioLibraryEntry) ScenarioLibraryEntry {
		return ScenarioLibraryEntry{ScenarioLibraryEntry: entry}
	})
}

// ListScenarioLibraryScenarios lists all scenarios within a scenario library entry.
func (a *gradientAIService) ListScenarioLibraryScenarios(libraryScenarioUUID string, opt *godo.ScenarioListOptions) (Scenarios, error) {
	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.Scenario, *godo.Response, error) {
		res, resp, err := a.client.GradientAI.ListScenarioLibraryScenarios(context.TODO(), libraryScenarioUUID, scenarioFilters(opt, listOpt))
		if err != nil {
			return nil, nil, err
		}
		return res.Scenarios, resp, nil
	}, wrapScenario)
}

// CreateScenarioSetFromLibrary materializes a library entry into a team-owned scenario set.
func (a *gradientAIService) CreateScenarioSetFromLibrary(libraryScenarioUUID string, req *godo.CreateScenarioSetFromLibraryRequest) (*ScenarioSet, error) {
	set, _, err := a.client.GradientAI.CreateScenarioSetFromLibrary(context.TODO(), libraryScenarioUUID, req)
	if err != nil {
		return nil, err
	}
	return &ScenarioSet{ScenarioSet: set}, nil
}

// CreateSimulationRun creates a simulation run.
func (a *gradientAIService) CreateSimulationRun(req *godo.CreateSimulationRunRequest) (*SimulationRun, error) {
	run, _, err := a.client.GradientAI.CreateSimulationRun(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return &SimulationRun{SimulationRun: run}, nil
}

// ListSimulationRuns lists all simulation runs for the team.
func (a *gradientAIService) ListSimulationRuns(opt *godo.SimulationRunListOptions) (SimulationRuns, error) {
	if opt == nil {
		opt = &godo.SimulationRunListOptions{}
	}

	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.SimulationRun, *godo.Response, error) {
		filters := *opt
		filters.ListOptions = *listOpt
		res, resp, err := a.client.GradientAI.ListSimulationRuns(context.TODO(), &filters)
		if err != nil {
			return nil, nil, err
		}
		return res.SimulationRuns, resp, nil
	}, func(run *godo.SimulationRun) SimulationRun {
		return SimulationRun{SimulationRun: run}
	})
}

// GetSimulationRun retrieves a simulation run by its UUID, including per-scenario results.
func (a *gradientAIService) GetSimulationRun(runUUID string) (*SimulationRunDetail, error) {
	res, _, err := a.client.GradientAI.GetSimulationRun(context.TODO(), runUUID)
	if err != nil {
		return nil, err
	}
	return &SimulationRunDetail{SimulationRunGetResponse: res}, nil
}

// UpdateSimulationRun updates a simulation run by its UUID.
func (a *gradientAIService) UpdateSimulationRun(runUUID string, req *godo.UpdateSimulationRunRequest) (*SimulationRun, error) {
	run, _, err := a.client.GradientAI.UpdateSimulationRun(context.TODO(), runUUID, req)
	if err != nil {
		return nil, err
	}
	return &SimulationRun{SimulationRun: run}, nil
}

// CancelSimulationRun cancels an in-progress simulation run.
func (a *gradientAIService) CancelSimulationRun(runUUID string) (*SimulationRun, error) {
	run, _, err := a.client.GradientAI.CancelSimulationRun(context.TODO(), runUUID)
	if err != nil {
		return nil, err
	}
	return &SimulationRun{SimulationRun: run}, nil
}

// DeleteSimulationRun deletes a simulation run by its UUID.
func (a *gradientAIService) DeleteSimulationRun(runUUID string) error {
	_, _, err := a.client.GradientAI.DeleteSimulationRun(context.TODO(), runUUID)
	return err
}

// ListSimulationJourneys lists all journeys for a simulation run.
func (a *gradientAIService) ListSimulationJourneys(runUUID string, opt *godo.SimulationJourneyListOptions) (SimulationJourneys, error) {
	if opt == nil {
		opt = &godo.SimulationJourneyListOptions{}
	}

	return paginateGenAIList(func(listOpt *godo.ListOptions) ([]*godo.SimulationJourney, *godo.Response, error) {
		filters := *opt
		filters.ListOptions = *listOpt
		res, resp, err := a.client.GradientAI.ListSimulationJourneys(context.TODO(), runUUID, &filters)
		if err != nil {
			return nil, nil, err
		}
		return res.Journeys, resp, nil
	}, func(journey *godo.SimulationJourney) SimulationJourney {
		return SimulationJourney{SimulationJourney: journey}
	})
}

// GetSimulationJourney retrieves a single journey within a simulation run.
func (a *gradientAIService) GetSimulationJourney(runUUID string, journeyUUID string) (*SimulationJourney, error) {
	journey, _, err := a.client.GradientAI.GetSimulationJourney(context.TODO(), runUUID, journeyUUID)
	if err != nil {
		return nil, err
	}
	return &SimulationJourney{SimulationJourney: journey}, nil
}

// GetSimulationJourneyTrajectory retrieves the trajectory of a journey.
func (a *gradientAIService) GetSimulationJourneyTrajectory(runUUID string, journeyUUID string) (*SimulationTrajectory, error) {
	trajectory, _, err := a.client.GradientAI.GetSimulationJourneyTrajectory(context.TODO(), runUUID, journeyUUID)
	if err != nil {
		return nil, err
	}
	return &SimulationTrajectory{SimulationTrajectory: trajectory}, nil
}

// GetSimulationJourneyTrajectoryURL returns a presigned download URL for a journey's trajectory.
func (a *gradientAIService) GetSimulationJourneyTrajectoryURL(runUUID string, journeyUUID string) (*GenAIDownloadURL, error) {
	res, _, err := a.client.GradientAI.GetSimulationJourneyTrajectoryURL(context.TODO(), runUUID, journeyUUID)
	if err != nil {
		return nil, err
	}
	return &GenAIDownloadURL{DownloadURL: res.DownloadURL, ExpiresAt: res.ExpiresAt}, nil
}

func scenarioFilters(opt *godo.ScenarioListOptions, listOpt *godo.ListOptions) *godo.ScenarioListOptions {
	filters := godo.ScenarioListOptions{}
	if opt != nil {
		filters = *opt
	}
	filters.ListOptions = *listOpt
	return &filters
}

func wrapScenario(scenario *godo.Scenario) Scenario {
	return Scenario{Scenario: scenario}
}

package commands

import (
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testLibraryScenarioUUID = "22222222-2222-4222-8222-222222222222"

	testScenarioLibraryEntry = do.ScenarioLibraryEntry{
		ScenarioLibraryEntry: &godo.ScenarioLibraryEntry{
			LibraryScenarioUUID: testLibraryScenarioUUID,
			Name:                "Support Flows",
			Category:            "customer-support",
			Status:              godo.ScenarioLibraryEntryStatusActive,
			ScenarioCount:       5,
		},
	}
)

func TestScenarioLibraryCommand(t *testing.T) {
	cmd := ScenarioLibraryCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "list", "list-scenarios", "create-scenario-set")
}

func TestScenarioLibraryList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListScenarioLibrary(&godo.ScenarioLibraryListOptions{}).
			Return(do.ScenarioLibraryEntries{testScenarioLibraryEntry}, nil)

		err := RunScenarioLibraryList(config)
		assert.NoError(t, err)
	})
}

func TestScenarioLibraryListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgScenarioLibraryCategory, "customer-support")
		config.Doit.Set(config.NS, doctl.ArgGenAISearch, "refund")
		config.Doit.Set(config.NS, doctl.ArgGenAISortBy, "name")
		config.Doit.Set(config.NS, doctl.ArgGenAISortDirection, "asc")

		tm.gradientAI.EXPECT().ListScenarioLibrary(&godo.ScenarioLibraryListOptions{
			Category:      "customer-support",
			Search:        "refund",
			SortBy:        godo.ScenarioLibrarySortFieldName,
			SortDirection: godo.GenAISortDirectionAsc,
		}).Return(do.ScenarioLibraryEntries{testScenarioLibraryEntry}, nil)

		err := RunScenarioLibraryList(config)
		assert.NoError(t, err)
	})
}

func TestScenarioLibraryListScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testLibraryScenarioUUID)

		tm.gradientAI.EXPECT().ListScenarioLibraryScenarios(testLibraryScenarioUUID, &godo.ScenarioListOptions{}).
			Return(do.Scenarios{testScenario}, nil)

		err := RunScenarioLibraryListScenarios(config)
		assert.NoError(t, err)
	})
}

func TestScenarioLibraryCreateScenarioSet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testLibraryScenarioUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Support Flows Copy")

		tm.gradientAI.EXPECT().CreateScenarioSetFromLibrary(testLibraryScenarioUUID, &godo.CreateScenarioSetFromLibraryRequest{
			LibraryScenarioUUID: testLibraryScenarioUUID,
			Name:                "Support Flows Copy",
		}).Return(&testScenarioSet, nil)

		err := RunScenarioLibraryCreateScenarioSet(config)
		assert.NoError(t, err)
	})
}

func TestScenarioLibraryCreateScenarioSetRequiresUUID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunScenarioLibraryCreateScenarioSet(config)
		assert.Error(t, err)
	})
}

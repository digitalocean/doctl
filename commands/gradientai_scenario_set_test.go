package commands

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testScenarioSetUUID = "00000000-0000-4000-8000-000000000000"

	testScenarioSet = do.ScenarioSet{
		ScenarioSet: &godo.ScenarioSet{
			ScenarioSetUUID: testScenarioSetUUID,
			Name:            "Test Scenario Set",
			Status:          godo.ScenarioSetStatusReady,
			SourceKind:      godo.ScenarioSetSourceKindUserUpload,
			ScenarioCount:   2,
		},
	}

	testScenario = do.Scenario{
		Scenario: &godo.Scenario{
			ScenarioUUID:      "11111111-1111-4111-8111-111111111111",
			Name:              "Test Scenario",
			Description:       "A customer asks for a refund",
			UserPersona:       "Frustrated customer",
			MaxTurns:          10,
			ExplorationBudget: 3,
		},
	}
)

func TestScenarioSetCommand(t *testing.T) {
	cmd := ScenarioSetCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "generate", "list", "get", "list-scenarios", "update", "delete", "download-url")
}

func TestScenarioSetCreateFromScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetScenarios, `[{"name":"Test Scenario","description":"A customer asks for a refund","max_turns":10}]`)

		tm.gradientAI.EXPECT().CreateScenarioSet(&godo.CreateScenarioSetRequest{
			Name: "Test Scenario Set",
			Scenarios: []*godo.Scenario{{
				Name:        "Test Scenario",
				Description: "A customer asks for a refund",
				MaxTurns:    10,
			}},
		}).Return(&testScenarioSet, nil)

		err := RunScenarioSetCreate(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetCreateFromFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		contents := []byte(`{"name":"Test Scenario"}`)
		path := filepath.Join(t.TempDir(), "scenarios.jsonl")
		require.NoError(t, os.WriteFile(path, contents, 0600))

		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetFile, path)

		size := strconv.Itoa(len(contents))
		tm.gradientAI.EXPECT().CreateScenarioSetUploadPresignedURLs(&godo.CreateScenarioSetUploadPresignedURLsRequest{
			Files: []*godo.PresignedUrlFile{{
				FileName: "scenarios.jsonl",
				FileSize: size,
			}},
		}).Return(&do.ScenarioSetFileUploads{
			CreateScenarioSetUploadPresignedURLsResponse: &godo.CreateScenarioSetUploadPresignedURLsResponse{
				Uploads: []*godo.FilePresignedUrlResponse{{
					ObjectKey:        "stored-object-key",
					OriginalFileName: "scenarios.jsonl",
					PresignedURL:     "https://example.com/upload",
				}},
			},
		}, nil)

		var uploadedTo string
		var uploaded []byte
		originalPut := putPresignedFile
		putPresignedFile = func(url string, body io.Reader, _ int64) error {
			uploadedTo = url
			var err error
			uploaded, err = io.ReadAll(body)
			return err
		}
		defer func() { putPresignedFile = originalPut }()

		tm.gradientAI.EXPECT().CreateScenarioSet(&godo.CreateScenarioSetRequest{
			Name: "Test Scenario Set",
			FileUploadScenarioSet: &godo.FileUploadDataSource{
				OriginalFileName: "scenarios.jsonl",
				Size:             size,
				StoredObjectKey:  "stored-object-key",
			},
		}).Return(&testScenarioSet, nil)

		err := RunScenarioSetCreate(config)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/upload", uploadedTo)
		assert.Equal(t, contents, uploaded)
	})
}

func TestScenarioSetCreateFromFileIncompleteUpload(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		contents := []byte(`{"name":"Test Scenario"}`)
		path := filepath.Join(t.TempDir(), "scenarios.jsonl")
		require.NoError(t, os.WriteFile(path, contents, 0600))

		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetFile, path)

		tm.gradientAI.EXPECT().CreateScenarioSetUploadPresignedURLs(&godo.CreateScenarioSetUploadPresignedURLsRequest{
			Files: []*godo.PresignedUrlFile{{
				FileName: "scenarios.jsonl",
				FileSize: strconv.Itoa(len(contents)),
			}},
		}).Return(&do.ScenarioSetFileUploads{
			CreateScenarioSetUploadPresignedURLsResponse: &godo.CreateScenarioSetUploadPresignedURLsResponse{
				Uploads: []*godo.FilePresignedUrlResponse{{
					OriginalFileName: "scenarios.jsonl",
					PresignedURL:     "https://example.com/upload",
				}},
			},
		}, nil)

		uploadAttempted := false
		originalPut := putPresignedFile
		putPresignedFile = func(string, io.Reader, int64) error {
			uploadAttempted = true
			return nil
		}
		defer func() { putPresignedFile = originalPut }()

		err := RunScenarioSetCreate(config)
		assert.Error(t, err)
		assert.False(t, uploadAttempted, "the file should not be uploaded when the object key is missing")
	})
}

func TestScenarioSetCreateRequiresFileOrScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")

		err := RunScenarioSetCreate(config)
		assert.Error(t, err)
	})
}

func TestScenarioSetCreateRejectsFileAndScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetFile, "scenarios.jsonl")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetScenarios, `[{"name":"Test Scenario"}]`)

		err := RunScenarioSetCreate(config)
		assert.Error(t, err)
	})
}

func TestScenarioSetGenerate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Test Scenario Set")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetGoalDescription, "Customers asking for refunds")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetNumScenarios, 10)
		config.Doit.Set(config.NS, doctl.ArgScenarioSetGeneratorModelUUID, "generator-model-uuid")

		tm.gradientAI.EXPECT().GenerateScenarioSet(&godo.GenerateScenarioSetRequest{
			Name:               "Test Scenario Set",
			GoalDescription:    "Customers asking for refunds",
			NumScenarios:       10,
			GeneratorModelUUID: "generator-model-uuid",
		}).Return(&testScenarioSet, nil)

		err := RunScenarioSetGenerate(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListScenarioSets(&godo.ScenarioSetListOptions{}).Return(do.ScenarioSets{testScenarioSet}, nil)

		err := RunScenarioSetList(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgGenAIStatuses, []string{"ready", "SCENARIO_SET_STATUS_FAILED"})
		config.Doit.Set(config.NS, doctl.ArgScenarioSetSourceKinds, []string{"user_upload"})
		config.Doit.Set(config.NS, doctl.ArgGenAISearch, "refund")
		config.Doit.Set(config.NS, doctl.ArgGenAISortBy, "created-at")
		config.Doit.Set(config.NS, doctl.ArgGenAISortDirection, "desc")

		tm.gradientAI.EXPECT().ListScenarioSets(&godo.ScenarioSetListOptions{
			Statuses:      []godo.ScenarioSetStatus{godo.ScenarioSetStatusReady, godo.ScenarioSetStatusFailed},
			SourceKinds:   []godo.ScenarioSetSourceKind{godo.ScenarioSetSourceKindUserUpload},
			Search:        "refund",
			SortBy:        godo.ScenarioSetSortFieldCreatedAt,
			SortDirection: godo.GenAISortDirectionDesc,
		}).Return(do.ScenarioSets{testScenarioSet}, nil)

		err := RunScenarioSetList(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)
		tm.gradientAI.EXPECT().GetScenarioSet(testScenarioSetUUID).Return(&testScenarioSet, nil)

		err := RunScenarioSetGet(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetGetRequiresUUID(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		err := RunScenarioSetGet(config)
		assert.Error(t, err)
	})
}

func TestScenarioSetListScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAISortBy, "name")

		tm.gradientAI.EXPECT().ListScenarios(testScenarioSetUUID, &godo.ScenarioListOptions{
			SortBy: godo.ScenarioSortFieldName,
		}).Return(do.Scenarios{testScenario}, nil)

		err := RunScenarioSetListScenarios(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgGenAIName, "Updated Scenario Set")

		tm.gradientAI.EXPECT().UpdateScenarioSet(testScenarioSetUUID, &godo.UpdateScenarioSetRequest{
			ScenarioSetUUID: testScenarioSetUUID,
			Name:            "Updated Scenario Set",
		}).Return(&testScenarioSet, nil)

		err := RunScenarioSetUpdate(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetUpdateRequiresNameOrScenarios(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)

		err := RunScenarioSetUpdate(config)
		assert.Error(t, err)
	})
}

func TestScenarioSetDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().DeleteScenarioSet(testScenarioSetUUID).Return(nil)

		err := RunScenarioSetDelete(config)
		assert.NoError(t, err)
	})
}

func TestScenarioSetDownloadURL(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testScenarioSetUUID)

		tm.gradientAI.EXPECT().GetScenarioSetDownloadURL(testScenarioSetUUID).
			Return(&do.GenAIDownloadURL{DownloadURL: "https://example.com/download"}, nil)

		err := RunScenarioSetDownloadURL(config)
		assert.NoError(t, err)
	})
}

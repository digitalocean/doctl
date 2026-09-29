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
	testEvaluationDatasetUUID = "66666666-6666-4666-8666-666666666666"

	testEvaluationDataset = do.EvaluationDataset{
		EvaluationDatasetInfo: &godo.EvaluationDatasetInfo{
			DatasetUUID:     testEvaluationDatasetUUID,
			DatasetName:     "support-prompts",
			DatasetType:     godo.EvaluationDatasetTypeModel,
			DatasetParadigm: godo.EvaluationDatasetParadigmSingleTurn,
			RowCount:        100,
			HasGroundTruth:  true,
		},
	}

	testEvaluationDatasetCreate = do.EvaluationDatasetCreate{
		CreateEvaluationDatasetResponse: &godo.CreateEvaluationDatasetResponse{
			EvaluationDatasetUUID: testEvaluationDatasetUUID,
		},
	}
)

func TestEvaluationDatasetCommand(t *testing.T) {
	cmd := EvaluationDatasetCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "create", "list", "delete")
}

func TestEvaluationDatasetCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		contents := []byte("input,ground_truth\nhello,world\n")
		path := filepath.Join(t.TempDir(), "prompts.csv")
		require.NoError(t, os.WriteFile(path, contents, 0600))

		config.Doit.Set(config.NS, doctl.ArgGenAIName, "support-prompts")
		config.Doit.Set(config.NS, doctl.ArgScenarioSetFile, path)

		size := strconv.Itoa(len(contents))
		tm.gradientAI.EXPECT().CreateModelEvalDatasetUploadPresignedURLs(&godo.CreateModelEvalDatasetUploadPresignedURLsRequest{
			Files: []*godo.PresignedUrlFile{{
				FileName: "prompts.csv",
				FileSize: size,
			}},
		}).Return(&do.ModelEvalDatasetFileUploads{
			CreateModelEvalDatasetUploadPresignedURLsResponse: &godo.CreateModelEvalDatasetUploadPresignedURLsResponse{
				Uploads: []*godo.FilePresignedUrlResponse{{
					ObjectKey:        "stored-object-key",
					OriginalFileName: "prompts.csv",
					PresignedURL:     "https://example.com/upload",
				}},
			},
		}, nil)

		// putPresignedFile normally HTTP PUTs the dataset to the Spaces
		// presigned URL. Stub it so the test records the upload without
		// hitting the network.
		var uploadedTo string
		var contentType string
		var uploaded []byte
		originalPut := putPresignedFile
		putPresignedFile = func(url string, body io.Reader, _ int64, ct string) error {
			uploadedTo = url
			contentType = ct
			var err error
			uploaded, err = io.ReadAll(body)
			return err
		}
		defer func() { putPresignedFile = originalPut }()

		tm.gradientAI.EXPECT().CreateEvaluationDataset(&godo.CreateEvaluationDatasetRequest{
			Name:            "support-prompts",
			DatasetType:     godo.EvaluationDatasetTypeModel,
			DatasetParadigm: godo.EvaluationDatasetParadigmSingleTurn,
			FileUploadDataset: &godo.FileUploadDataSource{
				OriginalFileName: "prompts.csv",
				Size:             size,
				StoredObjectKey:  "stored-object-key",
			},
		}).Return(&testEvaluationDatasetCreate, nil)

		err := RunEvaluationDatasetCreate(config)
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/upload", uploadedTo)
		assert.Equal(t, "text/csv", contentType)
		assert.Equal(t, contents, uploaded)
	})
}

func TestEvaluationDatasetList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.gradientAI.EXPECT().ListEvaluationDatasets(&godo.EvaluationDatasetListOptions{}).
			Return(do.EvaluationDatasets{testEvaluationDataset}, nil)

		err := RunEvaluationDatasetList(config)
		assert.NoError(t, err)
	})
}

func TestEvaluationDatasetListWithFilters(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Doit.Set(config.NS, doctl.ArgEvaluationDatasetType, "model")
		config.Doit.Set(config.NS, doctl.ArgEvaluationDatasetParadigm, "single_turn")
		config.Doit.Set(config.NS, doctl.ArgEvaluationHasGroundTruth, true)

		hasGroundTruth := true
		tm.gradientAI.EXPECT().ListEvaluationDatasets(&godo.EvaluationDatasetListOptions{
			DatasetType:     godo.EvaluationDatasetTypeModel,
			DatasetParadigm: godo.EvaluationDatasetParadigmSingleTurn,
			HasGroundTruth:  &hasGroundTruth,
		}).Return(do.EvaluationDatasets{testEvaluationDataset}, nil)

		err := RunEvaluationDatasetList(config)
		assert.NoError(t, err)
	})
}

func TestEvaluationDatasetDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		config.Args = append(config.Args, testEvaluationDatasetUUID)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.gradientAI.EXPECT().DeleteEvaluationDataset(testEvaluationDatasetUUID).Return(nil)

		err := RunEvaluationDatasetDelete(config)
		assert.NoError(t, err)
	})
}

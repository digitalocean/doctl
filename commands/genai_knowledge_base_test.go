package commands

import (
	"testing"
	"time"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
)

var (
	testKnowledgeBase = do.KnowledgeBase{
		KnowledgeBase: &godo.KnowledgeBase{
			Uuid:               "d35e5cb7-7957-4643-8e3a-1ab4eb3a494c",
			Name:               "Test Knowledge Base",
			Region:             "nyc3",
			ProjectId:          "test-project-id",
			DatabaseId:         "test-database-id",
			EmbeddingModelUuid: "test-embedding-model-uuid",
			IsPublic:           true,
			Tags:               []string{"tag1", "tag2"},
		},
	}

	testKBDataSource = do.KnowledgeBaseDataSource{
		KnowledgeBaseDataSource: &godo.KnowledgeBaseDataSource{
			Uuid: "data-source-id",
		},
	}

	testIndexingJob = do.IndexingJob{
		LastIndexingJob: &godo.LastIndexingJob{
			CompletedDatasources: 1,
			CreatedAt:            &godo.Timestamp{Time: time.Now()},
			DataSourceUuids:      []string{"data-source-uuid-1", "data-source-uuid-2"},
			FinishedAt:           &godo.Timestamp{Time: time.Now()},
			KnowledgeBaseUuid:    "kb-uuid-123",
			Phase:                "BATCH_JOB_PHASE_SUCCEEDED",
			StartedAt:            &godo.Timestamp{Time: time.Now()},
			Status:               "INDEX_JOB_STATUS_COMPLETED",
			Tokens:               1000,
			TotalDatasources:     2,
			TotalItemsFailed:     "0",
			TotalItemsIndexed:    "100",
			TotalItemsSkipped:    "5",
			UpdatedAt:            &godo.Timestamp{Time: time.Now()},
			Uuid:                 "indexing-job-uuid-123",
		},
	}

	testIndexingJobDataSource = do.IndexingJobDataSource{
		IndexedDataSource: &godo.IndexedDataSource{
			CompletedAt:       &godo.Timestamp{Time: time.Now()},
			DataSourceUuid:    "data-source-uuid-1",
			StartedAt:         &godo.Timestamp{Time: time.Now()},
			Status:            "DATA_SOURCE_STATUS_COMPLETED",
			IndexedItemCount:  "100",
			FailedItemCount:   "0",
			SkippedItemCount:  "5",
			IndexedFileCount:  "50",
			TotalFileCount:    "50",
			TotalBytes:        "1024000",
			TotalBytesIndexed: "1024000",
		},
	}
)

func TestKnowledgeBasesCommand(t *testing.T) {
	cmd := KnowledgeBaseCmd()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "add-datasource", "attach", "cancel-indexing-job", "create", "delete", "delete-datasource", "detach", "get", "get-indexing-job", "list", "list-datasources", "list-indexing-job-data-sources", "list-indexing-jobs", "update")
}

func TestKnowledgeBaseGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)
		tm.genAI.EXPECT().GetKnowledgeBase("00000000-0000-4000-8000-000000000000").Return(&testKnowledgeBase, nil)
		err := RunKnowledgeBaseGet(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListKnowledgeBases().Return(do.KnowledgeBases{testKnowledgeBase}, nil)
		err := RunKnowledgeBasesList(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseCreate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {

		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseName, "Test Knowledge Base")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseRegion, "tor1")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseProjectID, "test-project-id")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseEmbeddingModelUUID, "test-embedding-model-uuid")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseDataSource, `[{"web_crawler_data_source":{"base_url":"https://example.com","crawling_option":"Unknown","embed_media":true}}]`)

		tm.genAI.EXPECT().CreateKnowledgeBase(&godo.KnowledgeBaseCreateRequest{
			Name:               "Test Knowledge Base",
			Region:             "tor1",
			ProjectID:          "test-project-id",
			EmbeddingModelUuid: "test-embedding-model-uuid",
			DataSources: []godo.KnowledgeBaseDataSource{
				{
					WebCrawlerDataSource: &godo.WebCrawlerDataSource{
						BaseUrl:        "https://example.com",
						CrawlingOption: "Unknown",
						EmbedMedia:     true,
					},
				},
			},
		}).Return(&testKnowledgeBase, nil)

		err := RunKnowledgeBaseCreate(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseDelete(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)
		tm.genAI.EXPECT().DeleteKnowledgeBase("00000000-0000-4000-8000-000000000000").Return(nil)
		err := RunKnowledgeBaseDelete(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseUpdate(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)

		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseName, "Updated Knowledge Base")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseProjectID, "updated-project-id")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseEmbeddingModelUUID, "updated-embedding-model-uuid")

		tm.genAI.EXPECT().UpdateKnowledgeBase("00000000-0000-4000-8000-000000000000", &godo.UpdateKnowledgeBaseRequest{
			Name:               "Updated Knowledge Base",
			ProjectID:          "updated-project-id",
			EmbeddingModelUuid: "updated-embedding-model-uuid",
		}).Return(&testKnowledgeBase, nil)

		err := RunKnowledgeBaseUpdate(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseAddDataSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)

		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseBaseURL, "https://example.com")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseCrawlingOption, "Unknown")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseEmbedMedia, true)
		tm.genAI.EXPECT().AddKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", &godo.AddKnowledgeBaseDataSourceRequest{
			KnowledgeBaseUuid: knowledge_base_id,
			WebCrawlerDataSource: &godo.WebCrawlerDataSource{
				BaseUrl:        "https://example.com",
				CrawlingOption: "Unknown",
				EmbedMedia:     true,
			},
		}).Return(&testKBDataSource, nil)

		err := RunKnowledgeBaseAddDataSource(config)
		assert.NoError(t, err)
	})
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)

		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseBucketName, "sample-bucket")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseItemPath, "files/test")
		config.Doit.Set(config.NS, doctl.ArgKnowledgeBaseRegion, "tor1")
		tm.genAI.EXPECT().AddKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", &godo.AddKnowledgeBaseDataSourceRequest{
			KnowledgeBaseUuid: knowledge_base_id,
			SpacesDataSource: &godo.SpacesDataSource{
				BucketName: "sample-bucket",
				ItemPath:   "files/test",
				Region:     "tor1",
			},
		}).Return(&testKBDataSource, nil)

		err := RunKnowledgeBaseAddDataSource(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseDeleteDataSource(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		data_source_id := "data-source-id"
		config.Args = append(config.Args, knowledge_base_id, data_source_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.genAI.EXPECT().DeleteKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", "data-source-id").Return(nil)

		err := RunKnowledgeBaseDeleteDataSource(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseListDataSources(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)

		tm.genAI.EXPECT().ListKnowledgeBaseDataSources("00000000-0000-4000-8000-000000000000").Return(do.KnowledgeBaseDataSources{
			{
				KnowledgeBaseDataSource: &godo.KnowledgeBaseDataSource{
					Uuid: "data-source-id",
				},
			},
		}, nil)

		err := RunKnowledgeBaseListDataSources(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseAttach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agent_id := "00000000-0000-4000-8000-000000000000"
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, agent_id, knowledge_base_id)

		tm.genAI.EXPECT().AttachKnowledgeBaseToAgent(agent_id, knowledge_base_id).Return(&testAgent, nil)

		err := RunAttachKnowledgeBase(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseDetach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agent_id := "00000000-0000-4000-8000-000000000000"
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, agent_id, knowledge_base_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.genAI.EXPECT().DetachKnowledgeBaseToAgent(agent_id, knowledge_base_id).Return(&testAgent, nil)

		err := RunDetachKnowledgeBase(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseListIndexingJobs(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genAI.EXPECT().ListIndexingJobs().Return(do.IndexingJobs{testIndexingJob}, nil)
		err := RunKnowledgeBaseListIndexingJobs(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseGetIndexingJob(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		indexing_job_id := "indexing-job-uuid-123"
		config.Args = append(config.Args, indexing_job_id)
		tm.genAI.EXPECT().GetIndexingJob(indexing_job_id).Return(&testIndexingJob, nil)
		err := RunKnowledgeBaseGetIndexingJob(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseCancelIndexingJob(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		indexing_job_id := "indexing-job-uuid-123"
		config.Args = append(config.Args, indexing_job_id)
		tm.genAI.EXPECT().CancelIndexingJob(indexing_job_id).Return(&testIndexingJob, nil)
		err := RunKnowledgeBaseCancelIndexingJob(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseListIndexingJobDataSources(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		indexing_job_id := "indexing-job-uuid-123"
		config.Args = append(config.Args, indexing_job_id)
		tm.genAI.EXPECT().ListIndexingJobDataSources(indexing_job_id).Return(do.IndexingJobDataSources{testIndexingJobDataSource}, nil)
		err := RunKnowledgeBaseListIndexingJobDataSources(config)
		assert.NoError(t, err)
	})
}

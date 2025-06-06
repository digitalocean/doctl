package commands

import (
	"testing"

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

	testAgent = &do.Agent{
		Agent: &godo.Agent{
			Uuid:      "00000000-0000-4000-8000-000000000000",
			Name:      "Agent1",
			Region:    "tor1",
			ProjectId: "00000000-0000-4000-8000-000000000000",
			Model: &godo.Model{
				Uuid: "00000000-0000-4000-8000-000000000000",
			},
			Instruction: "You are an agent who thinks deeply about the world",
		},
	}

	testKBDataSource = do.KnowledgeBaseDataSource{
		KnowledgeBaseDataSource: &godo.KnowledgeBaseDataSource{
			UUID: "data-source-id",
		},
	}
)

func TestKnowledgeBasesCommand(t *testing.T) {
	cmd := KnowledgeBase()
	assert.NotNil(t, cmd)
	assertCommandNames(t, cmd, "add-datasource", "attach", "create", "delete", "delete-datasource", "detach", "get", "list", "list-datasources", "update")
}

func TestKnowledgeBaseGet(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)
		tm.genai.EXPECT().GetKnowledgeBase("00000000-0000-4000-8000-000000000000").Return(&testKnowledgeBase, nil)
		err := RunKnowledgeBaseGet(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseList(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		tm.genai.EXPECT().ListKnowledgeBases().Return(do.KnowledgeBases{testKnowledgeBase}, nil)
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

		tm.genai.EXPECT().CreateKnowledgeBase(&godo.KnowledgeBaseCreateRequest{
			Name:               "Test Knowledge Base",
			Region:             "tor1",
			ProjectID:          "test-project-id",
			EmbeddingModelUUID: "test-embedding-model-uuid",
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
		tm.genai.EXPECT().DeleteKnowledgebase("00000000-0000-4000-8000-000000000000").Return(nil)
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

		tm.genai.EXPECT().UpdateKnowledgebase("00000000-0000-4000-8000-000000000000", &godo.UpdateKnowledgeBaseRequest{
			Name:               "Updated Knowledge Base",
			ProjectID:          "updated-project-id",
			EmbeddingModelUUID: "updated-embedding-model-uuid",
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
		tm.genai.EXPECT().AddKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", &godo.AddDataSourceRequest{
			KnowledgeBaseUUID: knowledge_base_id,
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
		tm.genai.EXPECT().AddKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", &godo.AddDataSourceRequest{
			KnowledgeBaseUUID: knowledge_base_id,
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

		tm.genai.EXPECT().DeleteKnowledgeBaseDataSource("00000000-0000-4000-8000-000000000000", "data-source-id").Return(nil)

		err := RunKnowledgeBaseDeleteDataSource(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseListDataSources(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		knowledge_base_id := "00000000-0000-4000-8000-000000000000"
		config.Args = append(config.Args, knowledge_base_id)

		tm.genai.EXPECT().ListKnowledgeBaseDataSources("00000000-0000-4000-8000-000000000000").Return(do.KnowledgeBaseDataSources{
			{
				KnowledgeBaseDataSource: &godo.KnowledgeBaseDataSource{
					UUID: "data-source-id",
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
		knowledge_base_id := "00000000-0000-4000-8000-000000000001"
		config.Args = append(config.Args, agent_id, knowledge_base_id)

		tm.genai.EXPECT().AttachKnowledgebase(agent_id, knowledge_base_id).Return(testAgent, nil)

		err := RunAttachKnowledgeBase(config)
		assert.NoError(t, err)
	})
}

func TestKnowledgeBaseDetach(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		agent_id := "00000000-0000-4000-8000-000000000000"
		knowledge_base_id := "00000000-0000-4000-8000-000000000001"
		config.Args = append(config.Args, agent_id, knowledge_base_id)
		config.Doit.Set(config.NS, doctl.ArgForce, true)

		tm.genai.EXPECT().DetachKnowledgebase(agent_id, knowledge_base_id).Return(testAgent, nil)

		err := RunDetachKnowledgeBase(config)
		assert.NoError(t, err)
	})
}

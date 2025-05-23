package commands

import (
	"github.com/digitalocean/doctl"
	"github.com/digitalocean/doctl/commands/displayers"
	"github.com/digitalocean/doctl/do"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

func KnowledgeBase() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "knowledge-base",
			Aliases: []string{"kb"},
			Short:   "Display commands that manage DigitalOcean Agent Knowledge Bases.",
			Long:    "The subcommands of `doctl agent knowledge-base` allow you to access and manage knowledge bases of agents.",
		},
	}

	knowledgebaseDetails := `
		- The Knowledge Base UUID
		- The Knowledge Base Name
		- The Knowledge Base Region
		- The Knowledge Base Project ID
		- The Knowledge Base Embedding Model UUID
		- The Knowledge Base Created At
		- The Knowledge Base Added To Agent At
		- The Knowledge Base Embedding Model UUID
	`

	cmdKnowledgeBaseGet := CmdBuilder(
		cmd,
		RunKnowledgeBaseGet,
		"get <knowledge-base-uuid>",
		"Retrieves a Knowledge Base by its UUID.",
		"Retrieves information about a Knowledge Base, including:"+knowledgebaseDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	cmdKnowledgeBaseGet.Example = `The following example retrieves information about a Knowledge Base with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		` doctl compute agents knowledge-base get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdKnowledgeBaseCreate := CmdBuilder(
		cmd,
		RunKnowledgeBaseCreate,
		"create",
		"Creates a Knowledge Base.",
		"Creates a Knowledge Base with the following parameters:",
		Writer, aliasOpt("g"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	AddStringFlag(cmdKnowledgeBaseCreate, "name", "", "", "The name of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "region", "", "", "The region of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "project-id", "", "", "The project ID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "embedding-model-uuid", "", "", "The embedding model UUID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "database-id", "", "", "The database ID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "base-url", "", "", "The base URL of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "crawling-option", "", "", "The crawling option of the Knowledge Base.")
	AddBoolFlag(cmdKnowledgeBaseCreate, "embed-media", "", false, "The embed media option of the Knowledge Base.")
	AddStringSliceFlag(cmdKnowledgeBaseCreate, "tags", "", []string{}, "The tags of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "vpc_uuid", "", "", "The VPC UUID of the Knowledge Base.")
	cmdKnowledgeBaseCreate.Example = `The following example creates Knowledge Base with the paramters ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		` doctl compute agents knowledge-base get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdKnowledgeBasesList := "List all knowledge bases for agents."
	cmdKnowledgeBaseList := CmdBuilder(
		cmd,
		RunKnowledgeBasesList,
		"list",
		"List all knowledge bases for agent.",
		cmdKnowledgeBasesList,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	cmdKnowledgeBaseList.Example = "The following command lists all knowledge base for agents " +
		"`doctl compute agents knowledge-base list`"

	cmdKnowledgeBasesUpdateDetail := "Update a knowledge base by its uuid."
	cmdKnowledgeBasesUpdate := CmdBuilder(
		cmd,
		RunKnowledgeBaseUpdate,
		"update <knowledge-base-uuid>",
		"List all knowledge bases for agent.",
		cmdKnowledgeBasesUpdateDetail,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	AddStringFlag(cmdKnowledgeBasesUpdate, "name", "", "", "The name of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "project-id", "", "", "The project ID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "embedding-model-uuid", "", "", "The embedding model UUID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "database-id", "", "", "The database ID of the Knowledge Base.")
	AddStringSliceFlag(cmdKnowledgeBasesUpdate, "tags", "", []string{}, "The tags of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "uuid", "", "", "The UUID of the Knowledge Base.")
	cmdKnowledgeBasesUpdate.Example = "The following command usdate the knowledge base by its uuid " +
		"`doctl compute agents knowledge-base list`"

	cmdDataSourcesList := "List all datasource for knowledge base."
	cmdDataSourceList := CmdBuilder(
		cmd,
		RunKnowledgeBaseListDataSources,
		"list-datasources <knowledge-base-uuid>",
		"List all datasource for knowledge base.",
		cmdDataSourcesList,
		Writer, aliasOpt("ls-ds"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdDataSourceList.Example = "The following example retrieves information about a Data Sources with the Knowledge Base ID " + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		" : `doctl compute agents knowledge-base list-datasources f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdAttachKnowledgeBaseDetails := "Attach a knowledge base to an agent."
	cmdAttachKnowledgeBase := CmdBuilder(
		cmd,
		RunAttachKnowledgeBase,
		"attach <agent-uuid> <knowledge-base-uuid>",
		"Attach a knowledge base to an agent.",
		cmdAttachKnowledgeBaseDetails,
		Writer, aliasOpt("ath"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdAttachKnowledgeBase.Example = "The following example attaches the Knowledge Base ID" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + " to a specific agent ID" + "`" + `f81d4fae-0000-11d0-a765-000000000000` + "`" +
		"  `doctl compute agents knowledge-base attach  f81d4fae-0000-11d0-a765-000000000000 f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdDetachKnowledgeBaseDetails := "Detach a knowledge base from an agent."
	cmdDetachKnowledgeBase := CmdBuilder(
		cmd,
		RunAttachKnowledgeBase,
		"detach <agent-uuid> <knowledgebase-uuid>",
		"Detach a knowledge base from an agent.",
		cmdDetachKnowledgeBaseDetails,
		Writer, aliasOpt("dth"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdDetachKnowledgeBase.Example = "The following example detaches the Knowledge Base ID" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + " from specific agent ID" + "`" + `f81d4fae-0000-11d0-a765-000000000000` + "`" +
		"  `doctl compute agents knowledge-base detach  f81d4fae-0000-11d0-a765-000000000000 f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	return cmd
}

func RunKnowledgeBasesList(c *CmdConfig) error {

	knowledgeBases, err := c.GenAI().ListKnowledgeBases()
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBase{KnowledgeBases: knowledgeBases})
}

func RunKnowledgeBaseGet(c *CmdConfig) error {
	knowledgeBase, err := c.GenAI().GetKnowledgeBase(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBase{KnowledgeBases: do.KnowledgeBases{*knowledgeBase}})
}

func RunKnowledgeBaseCreate(c *CmdConfig) error {

	name, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseName)
	if err != nil {
		return err
	}

	region, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseRegion)
	if err != nil {
		return err
	}

	projectID, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseProjectID)
	if err != nil {
		return err
	}

	embeddingModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseEmbeddingModelUUID)
	if err != nil {
		return err
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgKnowledgeBaseTags)
	if err != nil {
		return err
	}

	vpcUUID, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseVPCUUID)
	if err != nil {
		return err
	}

	databaseId, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseDatabaseID)
	if err != nil {
		return err
	}

	baseUrl, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseBaseURL)
	if err != nil {
		return err
	}

	crawlingOption, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseCrawlingOption)
	if err != nil {
		return err
	}

	embedMedia, err := c.Doit.GetBool(c.NS, doctl.ArgKnowledgeBaseEmbedMedia)
	if err != nil {
		return err
	}

	webCrawler := &godo.WebCrawlerDataSource{
		BaseUrl:        baseUrl,
		CrawlingOption: crawlingOption,
		EmbedMedia:     embedMedia,
	}

	dataSources := []godo.KnowledgeBaseDataSource{
		{
			WebCrawlerDataSource: webCrawler,
		},
	}

	req := &godo.KnowledgeBaseCreateRequest{
		DatabaseID:         databaseId,
		DataSources:        dataSources,
		Name:               name,
		EmbeddingModelUUID: embeddingModelUUID,
		Region:             region,
		ProjectID:          projectID,
		Tags:               tags,
		VPCUUIUD:           vpcUUID,
	}

	knowledgeBase, err := c.GenAI().CreateKnowledgeBase(req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBase{KnowledgeBases: do.KnowledgeBases{*knowledgeBase}})
}

func RunKnowledgeBaseUpdate(c *CmdConfig) error {
	name, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseName)
	if err != nil {
		return err
	}

	projectID, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseProjectID)
	if err != nil {
		return err
	}

	tags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgKnowledgeBaseTags)
	if err != nil {
		return err
	}

	databaseId, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseDatabaseID)
	if err != nil {
		return err
	}

	embeddingModelUUID, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseEmbeddingModelUUID)
	if err != nil {
		return err
	}

	uuid, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseUUID)
	if err != nil {
		return err
	}

	req := &godo.UpdateKnowledgeBaseRequest{
		Name:               name,
		Tags:               tags,
		DatabaseID:         databaseId,
		ProjectID:          projectID,
		EmbeddingModelUUID: embeddingModelUUID,
		UUID:               uuid,
	}
	knowledgeBase, err := c.GenAI().UpdateKnowledgebase(c.Args[0], req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBase{KnowledgeBases: do.KnowledgeBases{*knowledgeBase}})
}

func RunKnowledgeBaseDelete(c *CmdConfig) error {
	err := c.GenAI().DeleteKnowledgebase(c.Args[0])
	return err
}

func RunKnowledgeBaseListDataSources(c *CmdConfig) error {
	knowledgeBaseDataSource, err := c.GenAI().ListKnowledgeBaseDataSources(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBaseDataSource{KnowledgeBaseDataSources: knowledgeBaseDataSource})
}

func RunAttachKnowledgeBase(c *CmdConfig) error {
	agent, err := c.GenAI().AttachKnowledgebase(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

func RunDetachKnowledgeBase(c *CmdConfig) error {
	agent, err := c.GenAI().DetachKnowledgebase(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

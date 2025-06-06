package commands

import (
	"encoding/json"
	"fmt"

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
		- The Knowledge Base Database ID
		- The Knowledge Base Last Indexing Job
		- The Knowledge Base Created At
		- The Knowledge Base Updated At
		- The Knowledge Base Added To Agent At
		- The Knowledge Base Embedding Model UUID
		- The Knowledge Base Tags
		- The Knowledge Base Is Public
		- The Knowledge Base User ID
	`

	cmdKnowledgeBaseGet := CmdBuilder(
		cmd,
		RunKnowledgeBaseGet,
		"get <knowledge-base-uuid>",
		"Retrieves a Knowledge Base by its uuid",
		"Retrieves information about a Knowledge Base, including:"+knowledgebaseDetails,
		Writer, aliasOpt("g"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	cmdKnowledgeBaseGet.Example = `The following example retrieves information about a Knowledge Base with the ID ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		` doctl genai knowledge-base get f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdKnowledgeBaseCreate := CmdBuilder(
		cmd,
		RunKnowledgeBaseCreate,
		"create",
		"Creates a knowledge base",
		"Creates a knowledge base and returns the following information \n"+knowledgebaseDetails+" \nFor more information about datasources, see the [datasources reference](https://docs.digitalocean.com/reference/api/digitalocean/#tag/GenAI-Platform-(Public-Preview)/operation/genai_create_knowledge_base)\n",
		Writer, aliasOpt("c"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	AddStringFlag(cmdKnowledgeBaseCreate, "name", "", "", "The name of the Knowledge Base.", requiredOpt())
	AddStringFlag(cmdKnowledgeBaseCreate, "region", "", "", "The region of the Knowledge Base.", requiredOpt())
	AddStringFlag(cmdKnowledgeBaseCreate, "project-id", "", "", "The project ID of the Knowledge Base.", requiredOpt())
	AddStringFlag(cmdKnowledgeBaseCreate, "embedding-model-uuid", "", "", "The embedding model UUID of the Knowledge Base.", requiredOpt())
	AddStringFlag(cmdKnowledgeBaseCreate, "data-sources", "", "", "JSON array of data source objects.", requiredOpt())
	AddStringFlag(cmdKnowledgeBaseCreate, "database-id", "", "", "The database ID of the Knowledge Base.")
	AddStringSliceFlag(cmdKnowledgeBaseCreate, "tags", "", []string{}, "The tags of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBaseCreate, "vpc_uuid", "", "", "The VPC UUID of the Knowledge Base.")
	cmdKnowledgeBaseCreate.Example = `The following example creates Knowledge Base with the parameters ` + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		` doctl genai knowledge-base create f81d4fae-7dec-11d0-a765-00a0c91e6bf6`

	cmdKnowledgeBasesList := "List all knowledge bases for agents where each knowledge base contains the following information:\n" + knowledgebaseDetails
	cmdKnowledgeBaseList := CmdBuilder(
		cmd,
		RunKnowledgeBasesList,
		"list",
		"List all knowledge bases for agents",
		cmdKnowledgeBasesList,
		Writer, aliasOpt("ls"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	cmdKnowledgeBaseList.Example = "The following command lists all knowledge base for agents " +
		"`doctl genai knowledge-base list`"

	cmdKnowledgeBasesUpdateDetail := "Update a knowledge base by its uuid and returns the updated information of the knowledge base with following details\n" + knowledgebaseDetails
	cmdKnowledgeBasesUpdate := CmdBuilder(
		cmd,
		RunKnowledgeBaseUpdate,
		"update <knowledge-base-uuid>",
		"Update a knowledge base",
		cmdKnowledgeBasesUpdateDetail,
		Writer, aliasOpt("u"),
		displayerType(&displayers.KnowledgeBase{}),
	)
	AddStringFlag(cmdKnowledgeBasesUpdate, "name", "", "", "The name of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "project-id", "", "", "The project ID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "embedding-model-uuid", "", "", "The embedding model UUID of the Knowledge Base.")
	AddStringFlag(cmdKnowledgeBasesUpdate, "database-id", "", "", "The database ID of the Knowledge Base.")
	AddStringSliceFlag(cmdKnowledgeBasesUpdate, "tags", "", []string{}, "The tags of the Knowledge Base. Example: --tags tag1,tag2,tag3")
	AddStringFlag(cmdKnowledgeBasesUpdate, "uuid", "", "", "The UUID of the Knowledge Base.")
	cmdKnowledgeBasesUpdate.Example = "The following command usdate the knowledge base by its uuid " +
		"`doctl genai knowledge-base list`"

	cmdKnowledgeBasesDeleteDetails := "Deletes the knowledge bases by its valid uuid."
	cmdKnowledgeBaseDelete := CmdBuilder(
		cmd,
		RunKnowledgeBaseDelete,
		"delete <knowledge-base-uuid>",
		"Deletes a knowledge base",
		cmdKnowledgeBasesDeleteDetails,
		Writer, aliasOpt("del", "rm"),
	)
	AddBoolFlag(cmdKnowledgeBaseDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the knowledge base without a confirmation prompt")
	cmdKnowledgeBaseDelete.Example = "The following command deletes the knowledge base by its uuid " + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` +
		"\n`doctl genai knowledge-base delete f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdDataSourcesList := "List all datasource for a valid knowledge base uuid."
	cmdDataSourceList := CmdBuilder(
		cmd,
		RunKnowledgeBaseListDataSources,
		"list-datasources <knowledge-base-uuid>",
		"List all datasource for knowledge base",
		cmdDataSourcesList,
		Writer, aliasOpt("ls-ds"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdDataSourceList.Example = "The following example retrieves information about a Data Sources with the Knowledge Base ID " + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		" : `doctl genai knowledge-base list-datasources f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdDataSourcesAddDetail := "Add a datasource for knowledge base by its uuid. Add only one Spaces or Webcrawler as a datasource. For more info about datasources, see the [datasources reference](https://docs.digitalocean.com/reference/api/digitalocean/#tag/GenAI-Platform-(Public-Preview)/operation/genai_create_knowledge_base_data_source)"
	cmdDataSourceAdd := CmdBuilder(
		cmd,
		RunKnowledgeBaseAddDataSource,
		"add-datasource <knowledge-base-uuid>",
		"Add one datasource for knowledge base",
		cmdDataSourcesAddDetail,
		Writer, aliasOpt("add-ds"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdDataSourceAdd.Example = "The following example adds a Webcrawler Data Sources with the Knowledge Base ID " + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + "`" +
		" : `doctl genai knowledge-base add-datasource f81d4fae-7dec-11d0-a765-00a0c91e6bf6 ----base-url https://www.example.com/data_source --crawling-option DOMAIN --embed-media false` \n Similarly for spaces Data Sources, you can use the following command: " +
		" \n `doctl genai knowledge-base add-datasource f81d4fae-7dec-11d0-a765-00a0c91e6bf6 --bucket-name my-bucket --item-path /path/to/item --region tor1`"
	AddStringFlag(cmdDataSourceAdd, "bucket-name", "", "", "The bucket name of data source from Spaces")
	AddStringFlag(cmdDataSourceAdd, "item-path", "", "", "Item path of data source from Spaces.")
	AddStringFlag(cmdDataSourceAdd, "region", "", "", "The region of the data source.")
	AddStringFlag(cmdDataSourceAdd, "base-url", "", "", "The base URL of the web crawler data source.")
	AddStringFlag(cmdDataSourceAdd, "crawling-option", "", "", "The crawling option of the web crawler data source.")
	AddBoolFlag(cmdDataSourceAdd, "embed-media", "", false, "The embed media option of the web crawler data source.")

	cmdDataSourcesDeleteDetail := "Delete a datasource for knowledge base using its id."
	cmdDataSourceDelete := CmdBuilder(
		cmd,
		RunKnowledgeBaseDeleteDataSource,
		"delete-datasource <knowledge-base-uuid> <data-source-id>",
		"Delete a datasource for knowledge base",
		cmdDataSourcesDeleteDetail,
		Writer, aliasOpt("d-ds"),
	)
	AddBoolFlag(cmdDataSourceDelete, doctl.ArgForce, doctl.ArgShortForce, false, "Deletes the datasource for knowledge base without a confirmation prompt")
	cmdDataSourceDelete.Example = "The following example deletes data source having uuid like " + `00000000-0000-0000-0000-000000000000` + " from a Knowledge Base having uuid " + "`" + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + " \nUsing the following command `" +
		" : `doctl genai knowledge-base delete-datasource f81d4fae-7dec-11d0-a765-00a0c91e6bf6 00000000-0000-0000-0000-000000000000`"

	cmdAttachKnowledgeBaseDetails := "Attach a knowledge base to an agent using knowledge base uuid and agent uuid. It returns the information of corresponding agent."
	cmdAttachKnowledgeBase := CmdBuilder(
		cmd,
		RunAttachKnowledgeBase,
		"attach <agent-uuid> <knowledge-base-uuid>",
		"Attach a knowledge base to an agent",
		cmdAttachKnowledgeBaseDetails,
		Writer, aliasOpt("ath"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	cmdAttachKnowledgeBase.Example = "The following example attaches the Knowledge Base having uuid - " + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + " to a specific agent having uuid - " + "`" + `f81d4fae-0000-11d0-a765-000000000000` + " \nUsing the following command : " +
		"doctl genai knowledge-base attach f81d4fae-0000-11d0-a765-000000000000 f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

	cmdDetachKnowledgeBaseDetails := "Detaches a knowledge base from an agent using knowledge base uuid and agent uuid."
	cmdDetachKnowledgeBase := CmdBuilder(
		cmd,
		RunDetachKnowledgeBase,
		"detach <agent-uuid> <knowledgebase-uuid>",
		"Detaches a knowledge base from an agent",
		cmdDetachKnowledgeBaseDetails,
		Writer, aliasOpt("dth"),
		displayerType(&displayers.KnowledgeBaseDataSource{}),
	)
	AddBoolFlag(cmdDetachKnowledgeBase, doctl.ArgForce, doctl.ArgShortForce, false, "Detaches the knowledge base without a confirmation prompt")
	cmdDetachKnowledgeBase.Example = "The following example detaches the Knowledge Base having uuid " + `f81d4fae-7dec-11d0-a765-00a0c91e6bf6` + " from specific agent with uuid as " + "`" + `f81d4fae-0000-11d0-a765-000000000000` + "`" +
		"\n`doctl genai knowledge-base detach f81d4fae-0000-11d0-a765-000000000000 f81d4fae-7dec-11d0-a765-00a0c91e6bf6`"

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
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
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

	dataSourceArray, err := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseDataSource)
	if err != nil {
		return err
	}

	var dataSources []godo.KnowledgeBaseDataSource
	if err := json.Unmarshal([]byte(dataSourceArray), &dataSources); err != nil {
		return fmt.Errorf("failed to parse data sources: %w", err)
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
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
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
	err := ensureOneArg(c)
	if err != nil {
		return err
	}
	knowledgeBaseId := c.Args[0]
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Knowledge Base", 1) == nil {
		err := c.GenAI().DeleteKnowledgebase(knowledgeBaseId)
		if err != nil {
			return err
		}
		notice("Knowledge Base deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}
	return nil
}

func RunKnowledgeBaseListDataSources(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	knowledgeBaseDataSource, err := c.GenAI().ListKnowledgeBaseDataSources(c.Args[0])
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBaseDataSource{KnowledgeBaseDataSources: knowledgeBaseDataSource})
}

func RunKnowledgeBaseAddDataSource(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	region, _ := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseRegion)
	bucketName, _ := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseBucketName)
	itemPath, _ := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseItemPath)
	baseUrl, _ := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseBaseURL)
	crawlingOption, _ := c.Doit.GetString(c.NS, doctl.ArgKnowledgeBaseCrawlingOption)
	baseEmbedMedia, _ := c.Doit.GetBool(c.NS, doctl.ArgKnowledgeBaseEmbedMedia)

	req := &godo.AddDataSourceRequest{
		KnowledgeBaseUUID: c.Args[0],
	}
	if bucketName != "" || itemPath != "" || region != "" {
		spacesDataSource := &godo.SpacesDataSource{
			BucketName: bucketName,
			ItemPath:   itemPath,
			Region:     region,
		}
		req.SpacesDataSource = spacesDataSource
	} else if baseUrl != "" {
		webCrawlerDataSource := &godo.WebCrawlerDataSource{
			BaseUrl:        baseUrl,
			CrawlingOption: crawlingOption,
			EmbedMedia:     baseEmbedMedia,
		}
		req.WebCrawlerDataSource = webCrawlerDataSource
	}

	knowledgeBaseDataSource, err := c.GenAI().AddKnowledgeBaseDataSource(c.Args[0], req)
	if err != nil {
		return err
	}
	return c.Display(&displayers.KnowledgeBaseDataSource{KnowledgeBaseDataSources: do.KnowledgeBaseDataSources{*knowledgeBaseDataSource}})
}

func RunKnowledgeBaseDeleteDataSource(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("DataSource of Knowledge Base", 1) == nil {
		err := c.GenAI().DeleteKnowledgeBaseDataSource(c.Args[0], c.Args[1])
		if err != nil {
			return err
		}
		notice("DataSource of Knowledge Base deleted successfully")
	} else {
		return fmt.Errorf("operation aborted")
	}

	return err
}

func RunAttachKnowledgeBase(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	agent, err := c.GenAI().AttachKnowledgebase(c.Args[0], c.Args[1])
	if err != nil {
		return err
	}
	return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
}

func RunDetachKnowledgeBase(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return doctl.NewMissingArgsErr(c.NS)
	}
	force, err := c.Doit.GetBool(c.NS, doctl.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirmDelete("Detach Knowledge Base from an Agent?", 1) == nil {
		agent, err := c.GenAI().DetachKnowledgebase(c.Args[0], c.Args[1])
		if err != nil {
			return err
		}
		notice("Knowledge Base detached successfully")
		return c.Display(&displayers.Agent{Agents: do.Agents{*agent}})
	} else {
		return fmt.Errorf("operation aborted")
	}
}

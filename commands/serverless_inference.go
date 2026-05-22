package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
)

// Inference creates the serverless inference command group.
func Inference() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "serverless-inference",
			Aliases: []string{"inference", "si"},
			Short:   "Call DigitalOcean serverless inference APIs",
			Long: `The subcommands of doctl inference call the serverless inference API at https://inference.do-ai.run.

Authenticate using --access-token. The value may be a model access key or a DigitalOcean personal access token with full access; all scopes must be granted for the serverless inference API to work.`,
			GroupID: serverlessInferenceGroup,
		},
	}

	cmd.AddCommand(serverlessInferenceChatCompletionsCmd())
	cmd.AddCommand(serverlessInferenceEmbeddingsCmd())
	cmd.AddCommand(serverlessInferenceImagesCmd())
	cmd.AddCommand(serverlessInferenceMessagesCmd())
	cmd.AddCommand(serverlessInferenceModelsCmd())
	cmd.AddCommand(serverlessInferenceResponsesCmd())
	cmd.AddCommand(serverlessInferenceAsyncCmd())

	return cmd
}

// ServerlessInference is an alias for Inference.
func ServerlessInference() *Command {
	return Inference()
}

// --- shared helpers ---

func serverlessInferenceCreateTestConfig(parent *Command, createName string, config *CmdConfig) {
	for _, child := range parent.ChildCommands() {
		if child.Name() != createName {
			continue
		}
		config.Command = child.Command
		config.NS = cmdNS(child)
		return
	}
}

func readServerlessInferenceJSON(path string, v any) error {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	if err := json.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

func serverlessInferenceRequestPath(c *CmdConfig) (string, error) {
	return c.Doit.GetString(c.NS, doctl.ArgInferenceRequest)
}

func serverlessInferenceModel(c *CmdConfig) (string, error) {
	return c.Doit.GetString(c.NS, doctl.ArgInferenceModel)
}

func serverlessInferenceStream(c *CmdConfig) (bool, error) {
	return c.Doit.GetBool(c.NS, doctl.ArgInferenceStream)
}

func writeServerlessInferenceJSON(w io.Writer, v any) error {
	if strings.EqualFold(Output, "json") {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func writeServerlessInferenceText(w io.Writer, text string) error {
	_, err := fmt.Fprintln(w, text)
	return err
}

func runServerlessInferenceStream(c *CmdConfig, stream interface {
	Next() bool
	Err() error
	Close() error
}, writeEvent func() error) error {
	defer stream.Close()

	ctx, stop := signal.NotifyContext(c.Command.Context(), os.Interrupt)
	defer stop()

	wroteText := false
	for stream.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := writeEvent(); err != nil {
			return err
		}
		wroteText = true
	}
	if err := stream.Err(); err != nil {
		return err
	}
	if wroteText && !strings.EqualFold(Output, "json") {
		_, err := fmt.Fprintln(c.Out)
		return err
	}
	return nil
}

func serverlessInferenceFlagChanged(c *CmdConfig, name string) bool {
	return c.Command != nil && c.Command.Flags().Changed(name)
}

// --- chat-completions ---

func serverlessInferenceChatCompletionsCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "chat-completions",
			Aliases: []string{"chat", "chatcompletion"},
			Short:   "Display commands for creating chat completions",
			Long:    "The subcommands of `doctl inference chat-completions` send chat-style prompts to a model and return responses.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceChatCompletionCreate, "create", "Create a chat completion",
		`Creates a chat completion using the specified model. Use --model and --message for quick prompts, or --request for a full JSON body. Use --stream to receive tokens as they arrive via server-sent events.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "Model ID (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceMessage, "", "", "User message (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceSystemMessage, "", "", "Optional system message")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")
	AddBoolFlag(create, doctl.ArgInferenceStream, "", false, "Stream using server-sent events")
	AddFloatFlag(create, doctl.ArgInferenceTemperature, "", 0, "Sampling temperature")
	AddIntFlag(create, doctl.ArgInferenceMaxTokens, "", 0, "Maximum tokens to generate")

	create.Example = `doctl inference chat-completions create --model llama3-8b-instruct --message "Hello"
doctl inference chat-completions create --model llama3-8b-instruct --message "Hello" --stream
doctl inference chat-completions create --request ./chat-request.json`

	return cmd
}

// RunServerlessInferenceChatCompletionCreate runs chat-completions create.
func RunServerlessInferenceChatCompletionCreate(c *CmdConfig) error {
	params, err := serverlessInferenceChatCompletionParams(c)
	if err != nil {
		return err
	}
	stream, err := serverlessInferenceStream(c)
	if err != nil {
		return err
	}
	if stream {
		return runServerlessInferenceChatCompletionStream(c, params)
	}
	return runServerlessInferenceChatCompletion(c, params)
}

// RunInferenceChatCompletionCreate is an alias for tests and backward compatibility.
func RunInferenceChatCompletionCreate(c *CmdConfig) error {
	return RunServerlessInferenceChatCompletionCreate(c)
}

func serverlessInferenceChatCompletionParams(c *CmdConfig) (*godo.ChatCompletionNewParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.ChatCompletionNewParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	message, err := c.Doit.GetString(c.NS, doctl.ArgInferenceMessage)
	if err != nil {
		return nil, err
	}
	if message == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceMessage, doctl.ArgInferenceRequest)
	}

	params := &godo.ChatCompletionNewParams{
		Model:    model,
		Messages: []godo.ChatCompletionMessage{godo.UserMessage(message)},
	}

	systemMessage, err := c.Doit.GetString(c.NS, doctl.ArgInferenceSystemMessage)
	if err != nil {
		return nil, err
	}
	if systemMessage != "" {
		params.Messages = append([]godo.ChatCompletionMessage{godo.SystemMessage(systemMessage)}, params.Messages...)
	}

	if serverlessInferenceFlagChanged(c, doctl.ArgInferenceTemperature) {
		temp, err := c.Doit.GetFloat64(c.NS, doctl.ArgInferenceTemperature)
		if err != nil {
			return nil, err
		}
		params.Temperature = godo.PtrTo(temp)
	}

	if serverlessInferenceFlagChanged(c, doctl.ArgInferenceMaxTokens) {
		maxTokens, err := c.Doit.GetInt(c.NS, doctl.ArgInferenceMaxTokens)
		if err != nil {
			return nil, err
		}
		params.MaxTokens = godo.PtrTo(maxTokens)
	}

	return params, nil
}

func inferenceChatCompletionParams(c *CmdConfig) (*godo.ChatCompletionNewParams, error) {
	return serverlessInferenceChatCompletionParams(c)
}

func runServerlessInferenceChatCompletion(c *CmdConfig, params *godo.ChatCompletionNewParams) error {
	completion, err := c.Inference().CreateChatCompletion(params)
	if err != nil {
		return err
	}
	if strings.EqualFold(Output, "json") {
		return writeServerlessInferenceJSON(c.Out, completion)
	}
	text := serverlessInferenceChatCompletionText(completion)
	if text == "" {
		return writeServerlessInferenceJSON(c.Out, completion)
	}
	return writeServerlessInferenceText(c.Out, text)
}

func runServerlessInferenceChatCompletionStream(c *CmdConfig, params *godo.ChatCompletionNewParams) error {
	stream, err := c.Inference().CreateChatCompletionStreaming(params)
	if err != nil {
		return err
	}
	jsonOut := strings.EqualFold(Output, "json")
	return runServerlessInferenceStream(c, stream, func() error {
		chunk := stream.Current()
		if jsonOut {
			return writeServerlessInferenceJSON(c.Out, chunk)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				if _, err := fmt.Fprint(c.Out, choice.Delta.Content); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func serverlessInferenceChatCompletionText(completion *godo.ChatCompletion) string {
	if completion == nil {
		return ""
	}
	var b strings.Builder
	for _, choice := range completion.Choices {
		if choice.Message.Content != nil {
			b.WriteString(*choice.Message.Content)
		}
	}
	return b.String()
}

// --- embeddings ---

func serverlessInferenceEmbeddingsCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "embeddings",
			Short: "Display commands for creating embedding vectors",
			Long:  "The subcommands of `doctl inference embeddings` convert text into dense vector representations for semantic search, RAG, and similarity matching.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceEmbeddingsCreate, "create", "Create embeddings",
		`Creates embedding vectors for the provided input text. Use --model and --input for quick requests, or --request for a full JSON body.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "Model ID (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceInput, "", "", "Input text (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")

	create.Example = `doctl inference embeddings create --model qwen3-embedding-0.6b --input "hello"
doctl inference embeddings create --request ./embedding-request.json`

	return cmd
}

func RunServerlessInferenceEmbeddingsCreate(c *CmdConfig) error {
	params, err := serverlessInferenceEmbeddingParams(c)
	if err != nil {
		return err
	}
	resp, err := c.Inference().CreateEmbedding(params)
	if err != nil {
		return err
	}
	return writeServerlessInferenceJSON(c.Out, resp)
}

func serverlessInferenceEmbeddingParams(c *CmdConfig) (*godo.EmbeddingNewParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.EmbeddingNewParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	input, err := c.Doit.GetString(c.NS, doctl.ArgInferenceInput)
	if err != nil {
		return nil, err
	}
	if input == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceInput, doctl.ArgInferenceRequest)
	}

	return &godo.EmbeddingNewParams{Model: model, Input: input}, nil
}

// --- images ---

func serverlessInferenceImagesCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "images",
			Short: "Display commands for generating images",
			Long:  "The subcommands of `doctl inference images` generate images from text prompts.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceImagesCreate, "create", "Generate an image",
		`Generates an image from a text prompt. Use --model and --prompt for quick requests, or --request for a full JSON body. Use --output to write the generated image to a file.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "Model ID (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferencePrompt, "", "", "Image prompt (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")
	AddBoolFlag(create, doctl.ArgInferenceStream, "", false, "Stream partial images using SSE")
	AddStringFlag(create, doctl.ArgInferenceOutput, "o", "", "Write the first image (base64 decoded) to this path")
	AddIntFlag(create, doctl.ArgInferenceN, "", 1, "Number of images to generate")

	create.Example = `doctl inference images create --model openai-gpt-image-1 --prompt "a green cube" --output cube.png
doctl inference images create --model openai-gpt-image-1 --prompt "a green cube" --stream`

	return cmd
}

func RunServerlessInferenceImagesCreate(c *CmdConfig) error {
	params, err := serverlessInferenceImageParams(c)
	if err != nil {
		return err
	}
	stream, err := serverlessInferenceStream(c)
	if err != nil {
		return err
	}
	if stream {
		return runServerlessInferenceImageStream(c, params)
	}
	return runServerlessInferenceImage(c, params)
}

func serverlessInferenceImageParams(c *CmdConfig) (*godo.ImageGenerateParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.ImageGenerateParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	prompt, err := c.Doit.GetString(c.NS, doctl.ArgInferencePrompt)
	if err != nil {
		return nil, err
	}
	if prompt == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferencePrompt, doctl.ArgInferenceRequest)
	}

	n, err := c.Doit.GetInt(c.NS, doctl.ArgInferenceN)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		n = 1
	}

	return &godo.ImageGenerateParams{Model: model, Prompt: prompt, N: n}, nil
}

func runServerlessInferenceImage(c *CmdConfig, params *godo.ImageGenerateParams) error {
	resp, err := c.Inference().GenerateImage(params)
	if err != nil {
		return err
	}

	outputPath, err := c.Doit.GetString(c.NS, doctl.ArgInferenceOutput)
	if err != nil {
		return err
	}
	if outputPath != "" {
		if len(resp.Data) == 0 || resp.Data[0].B64JSON == "" {
			return fmt.Errorf("no image data in response")
		}
		data, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
		if err != nil {
			return fmt.Errorf("decode image: %w", err)
		}
		if err := os.WriteFile(outputPath, data, 0o644); err != nil {
			return err
		}
		if !strings.EqualFold(Output, "json") {
			_, err = fmt.Fprintf(c.Out, "wrote %s\n", outputPath)
			return err
		}
	}

	return writeServerlessInferenceJSON(c.Out, resp)
}

func runServerlessInferenceImageStream(c *CmdConfig, params *godo.ImageGenerateParams) error {
	stream, err := c.Inference().GenerateImageStreaming(params)
	if err != nil {
		return err
	}
	jsonOut := strings.EqualFold(Output, "json")
	return runServerlessInferenceStream(c, stream, func() error {
		ev := stream.Current()
		if jsonOut {
			return writeServerlessInferenceJSON(c.Out, ev)
		}
		if ev.B64JSON != "" {
			_, err := fmt.Fprintf(c.Out, "[%s] partial image index %d\n", ev.Type, ev.PartialImageIndex)
			return err
		}
		_, err := fmt.Fprintf(c.Out, "[%s]\n", ev.Type)
		return err
	})
}

// --- messages ---

func serverlessInferenceMessagesCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "messages",
			Short: "Display commands for creating Anthropic-style messages",
			Long:  "The subcommands of `doctl inference messages` send prompts to Anthropic-compatible models and return responses.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceMessagesCreate, "create", "Create a message",
		`Creates a message using the Anthropic-compatible messages API. Use --model and --message for quick prompts, or --request for a full JSON body.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "Model ID (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceMessage, "", "", "User message (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")
	AddBoolFlag(create, doctl.ArgInferenceStream, "", false, "Stream using server-sent events")
	AddIntFlag(create, doctl.ArgInferenceMaxTokens, "", 1024, "Maximum tokens to generate")

	create.Example = `doctl inference messages create --model claude-opus-4-6 --message "Hello" --max-tokens 256
doctl inference messages create --request ./message-request.json --stream`

	return cmd
}

func RunServerlessInferenceMessagesCreate(c *CmdConfig) error {
	params, err := serverlessInferenceMessageParams(c)
	if err != nil {
		return err
	}
	stream, err := serverlessInferenceStream(c)
	if err != nil {
		return err
	}
	if stream {
		return runServerlessInferenceMessageStream(c, params)
	}
	return runServerlessInferenceMessage(c, params)
}

func serverlessInferenceMessageParams(c *CmdConfig) (*godo.MessageNewParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.MessageNewParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	message, err := c.Doit.GetString(c.NS, doctl.ArgInferenceMessage)
	if err != nil {
		return nil, err
	}
	if message == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceMessage, doctl.ArgInferenceRequest)
	}

	maxTokens, err := c.Doit.GetInt(c.NS, doctl.ArgInferenceMaxTokens)
	if err != nil {
		return nil, err
	}

	content, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	return &godo.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []godo.MessageParam{{
			Role:    "user",
			Content: content,
		}},
	}, nil
}

func runServerlessInferenceMessage(c *CmdConfig, params *godo.MessageNewParams) error {
	msg, err := c.Inference().CreateMessage(params)
	if err != nil {
		return err
	}
	if strings.EqualFold(Output, "json") {
		return writeServerlessInferenceJSON(c.Out, msg)
	}
	text := serverlessInferenceMessageText(msg)
	if text == "" {
		return writeServerlessInferenceJSON(c.Out, msg)
	}
	return writeServerlessInferenceText(c.Out, text)
}

func runServerlessInferenceMessageStream(c *CmdConfig, params *godo.MessageNewParams) error {
	stream, err := c.Inference().CreateMessageStreaming(params)
	if err != nil {
		return err
	}
	jsonOut := strings.EqualFold(Output, "json")
	return runServerlessInferenceStream(c, stream, func() error {
		ev := stream.Current()
		if jsonOut {
			return writeServerlessInferenceJSON(c.Out, ev)
		}
		if ev.Delta.Text != "" {
			if _, err := fmt.Fprint(c.Out, ev.Delta.Text); err != nil {
				return err
			}
		}
		return nil
	})
}

func serverlessInferenceMessageText(msg *godo.Message) string {
	if msg == nil {
		return ""
	}
	var b strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

// --- models ---

func serverlessInferenceModelsCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "models",
			Short: "Display commands for listing available inference models",
			Long:  "The subcommands of `doctl inference models` list the models available to your inference API key.",
		},
	}

	list := CmdBuilder(cmd, RunServerlessInferenceModelsList, "list", "List inference models",
		`Lists all models available to your inference API key, including their IDs and owners.`, Writer, aliasOpt("ls"))
	list.Example = `doctl inference models list`

	return cmd
}

func RunServerlessInferenceModelsList(c *CmdConfig) error {
	list, err := c.Inference().ListModels()
	if err != nil {
		return err
	}
	if strings.EqualFold(Output, "json") {
		return writeServerlessInferenceJSON(c.Out, list)
	}
	for _, m := range list.Data {
		if _, err := fmt.Fprintf(c.Out, "%s\t%s\n", m.ID, m.OwnedBy); err != nil {
			return err
		}
	}
	return nil
}

// --- responses ---

func serverlessInferenceResponsesCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "responses",
			Short: "Display commands for creating model responses",
			Long:  "The subcommands of `doctl inference responses` send prompts to a model using the Responses API and return structured output.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceResponsesCreate, "create", "Create a response",
		`Creates a response using the Responses API. Use --model and --input for quick requests, or --request for a full JSON body. Use --stream to receive output as it is generated.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "Model ID (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceInput, "", "", "Input text (required unless --request is set)")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")
	AddBoolFlag(create, doctl.ArgInferenceStream, "", false, "Stream using server-sent events")
	AddStringFlag(create, doctl.ArgInferenceInstructions, "", "", "Optional instructions")

	create.Example = `doctl inference responses create --model openai-gpt-oss-20b --input "Hello"
doctl inference responses create --model openai-gpt-oss-20b --input "Hello" --stream`

	return cmd
}

func RunServerlessInferenceResponsesCreate(c *CmdConfig) error {
	params, err := serverlessInferenceResponseParams(c)
	if err != nil {
		return err
	}
	stream, err := serverlessInferenceStream(c)
	if err != nil {
		return err
	}
	if stream {
		return runServerlessInferenceResponseStream(c, params)
	}
	return runServerlessInferenceResponse(c, params)
}

func serverlessInferenceResponseParams(c *CmdConfig) (*godo.ResponseNewParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.ResponseNewParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	input, err := c.Doit.GetString(c.NS, doctl.ArgInferenceInput)
	if err != nil {
		return nil, err
	}
	if input == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceInput, doctl.ArgInferenceRequest)
	}

	params := &godo.ResponseNewParams{Model: model, Input: input}

	instructions, err := c.Doit.GetString(c.NS, doctl.ArgInferenceInstructions)
	if err != nil {
		return nil, err
	}
	if instructions != "" {
		params.Instructions = godo.PtrTo(instructions)
	}

	return params, nil
}

func runServerlessInferenceResponse(c *CmdConfig, params *godo.ResponseNewParams) error {
	resp, err := c.Inference().CreateResponse(params)
	if err != nil {
		return err
	}
	if strings.EqualFold(Output, "json") {
		return writeServerlessInferenceJSON(c.Out, resp)
	}
	text := resp.OutputText()
	if text == "" {
		return writeServerlessInferenceJSON(c.Out, resp)
	}
	return writeServerlessInferenceText(c.Out, text)
}

func runServerlessInferenceResponseStream(c *CmdConfig, params *godo.ResponseNewParams) error {
	stream, err := c.Inference().CreateResponseStreaming(params)
	if err != nil {
		return err
	}
	jsonOut := strings.EqualFold(Output, "json")
	return runServerlessInferenceStream(c, stream, func() error {
		ev := stream.Current()
		if jsonOut {
			return writeServerlessInferenceJSON(c.Out, ev)
		}
		if ev.Delta != "" {
			if _, err := fmt.Fprint(c.Out, ev.Delta); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- async-invoke ---

func serverlessInferenceAsyncCmd() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "async-invoke",
			Aliases: []string{"async"},
			Short:   "Display commands for managing async model invocations",
			Long:    "The subcommands of `doctl inference async-invoke` submit asynchronous jobs to fal models and retrieve their results.",
		},
	}

	create := CmdBuilder(cmd, RunServerlessInferenceAsyncCreate, "create", "Start an async invocation",
		`Starts an asynchronous job for a fal model and returns a request ID. Use --model and --prompt for image or audio generation, --text for text-to-speech, or --request for a full JSON body. Poll the result using the get subcommand.`, Writer)
	AddStringFlag(create, doctl.ArgInferenceModel, "m", "", "fal model ID (maps to model_id in the API)")
	AddStringFlag(create, doctl.ArgInferencePrompt, "", "", "Prompt for image or audio generation (input.prompt)")
	AddStringFlag(create, doctl.ArgInferenceText, "", "", "Text for text-to-speech generation (input.text)")
	AddIntFlag(create, doctl.ArgInferenceSecondsTotal, "", 0, "Audio duration in seconds (input.seconds_total)")
	AddStringFlag(create, doctl.ArgInferenceRequest, "", "", "Path to JSON request body. Use \"-\" for stdin.")
	AddStringSliceFlag(create, doctl.ArgTag, "", nil, "Tag in key=value form (repeatable)")
	create.Example = `doctl inference async-invoke create --model fal-ai/flux/schnell --prompt "A futuristic city at sunset"
doctl inference async-invoke create --model fal-ai/elevenlabs/tts/multilingual-v2 --text "Hello world"
doctl inference async-invoke create --request ./async-request.json`

	get := CmdBuilder(cmd, RunServerlessInferenceAsyncGet, "get <request-id>", "Get an async invocation",
		`Retrieves the status and output of an async invocation. Output is populated once the job reaches a terminal state (COMPLETED or FAILED).`, Writer)
	get.Example = `doctl inference async-invoke get req_abc123`

	return cmd
}

func RunServerlessInferenceAsyncCreate(c *CmdConfig) error {
	params, err := serverlessInferenceAsyncInvocationParams(c)
	if err != nil {
		return err
	}

	inv, err := c.Inference().CreateAsyncInvocation(params)
	if err != nil {
		return err
	}
	return writeServerlessInferenceJSON(c.Out, inv)
}

func serverlessInferenceAsyncInvocationParams(c *CmdConfig) (*godo.AsyncInvocationNewParams, error) {
	requestPath, err := serverlessInferenceRequestPath(c)
	if err != nil {
		return nil, err
	}
	if requestPath != "" {
		params := new(godo.AsyncInvocationNewParams)
		if err := readServerlessInferenceJSON(requestPath, params); err != nil {
			return nil, err
		}
		return params, nil
	}

	model, err := serverlessInferenceModel(c)
	if err != nil {
		return nil, err
	}
	if model == "" {
		return nil, fmt.Errorf("--%s is required when --%s is not set", doctl.ArgInferenceModel, doctl.ArgInferenceRequest)
	}

	prompt, err := c.Doit.GetString(c.NS, doctl.ArgInferencePrompt)
	if err != nil {
		return nil, err
	}
	text, err := c.Doit.GetString(c.NS, doctl.ArgInferenceText)
	if err != nil {
		return nil, err
	}
	if prompt == "" && text == "" {
		return nil, fmt.Errorf("at least one of --%s or --%s is required when --%s is not set",
			doctl.ArgInferencePrompt, doctl.ArgInferenceText, doctl.ArgInferenceRequest)
	}

	input := make(map[string]any)
	if prompt != "" {
		input["prompt"] = prompt
	}
	if text != "" {
		input["text"] = text
	}
	if serverlessInferenceFlagChanged(c, doctl.ArgInferenceSecondsTotal) {
		seconds, err := c.Doit.GetInt(c.NS, doctl.ArgInferenceSecondsTotal)
		if err != nil {
			return nil, err
		}
		if seconds > 0 {
			input["seconds_total"] = seconds
		}
	}

	tags, err := serverlessInferenceAsyncTags(c)
	if err != nil {
		return nil, err
	}

	params := &godo.AsyncInvocationNewParams{
		ModelID: model,
		Input:   input,
		Tags:    tags,
	}
	return params, nil
}

func serverlessInferenceAsyncTags(c *CmdConfig) ([]godo.InferenceTag, error) {
	tagFlags, err := c.Doit.GetStringSlice(c.NS, doctl.ArgTag)
	if err != nil {
		return nil, err
	}
	if len(tagFlags) == 0 {
		return nil, nil
	}

	tags := make([]godo.InferenceTag, 0, len(tagFlags))
	for _, tag := range tagFlags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("--%s must be key=value", doctl.ArgTag)
		}
		tags = append(tags, godo.InferenceTag{Key: parts[0], Value: parts[1]})
	}
	return tags, nil
}

func RunServerlessInferenceAsyncGet(c *CmdConfig) error {
	if len(c.Args) < 1 {
		return doctl.NewMissingArgsErr(c.NS)
	}

	inv, err := c.Inference().GetAsyncInvocation(c.Args[0])
	if err != nil {
		return err
	}
	return writeServerlessInferenceJSON(c.Out, inv)
}

package commands

import (
	"io"
	"os"
	"testing"

	"github.com/digitalocean/doctl"
	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunServerlessInferenceChatCompletionCreate_NonStreaming(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceChatCompletionsCmd(), "create", config)
		config.Out = io.Discard
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "openai-gpt-oss-20b")
		config.Doit.Set(config.NS, doctl.ArgInferenceMessage, "hi")

		content := "hello back"
		expected := &godo.ChatCompletion{
			Choices: []godo.ChatCompletionChoice{{
				Message: godo.ChatCompletionMessage{
					Role:    "assistant",
					Content: &content,
				},
			}},
		}

		tm.inference.EXPECT().
			CreateChatCompletion(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, params *godo.ChatCompletionNewParams) (*godo.ChatCompletion, error) {
				assert.Equal(t, "openai-gpt-oss-20b", params.Model)
				require.Len(t, params.Messages, 1)
				require.NotNil(t, params.Messages[0].Content)
				assert.Equal(t, "hi", *params.Messages[0].Content)
				return expected, nil
			})

		err := RunServerlessInferenceChatCompletionCreate(config)
		require.NoError(t, err)
	})
}

func TestRunServerlessInferenceChatCompletionCreate_Streaming(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceChatCompletionsCmd(), "create", config)
		config.Out = io.Discard
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "openai-gpt-oss-20b")
		config.Doit.Set(config.NS, doctl.ArgInferenceMessage, "hi")
		config.Doit.Set(config.NS, doctl.ArgInferenceStream, true)

		tm.inference.EXPECT().
			CreateChatCompletionStreaming(gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError)

		err := RunServerlessInferenceChatCompletionCreate(config)
		assert.Error(t, err)
	})
}

func TestServerlessInferenceChatCompletionParams_FromRequestFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceChatCompletionsCmd(), "create", config)
		body := `{"model":"m","messages":[{"role":"user","content":"from file"}]}`
		path := t.TempDir() + "/request.json"
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		config.Doit.Set(config.NS, doctl.ArgInferenceRequest, path)

		params, err := serverlessInferenceChatCompletionParams(config)
		require.NoError(t, err)
		assert.Equal(t, "m", params.Model)
		require.Len(t, params.Messages, 1)
		require.NotNil(t, params.Messages[0].Content)
		assert.Equal(t, "from file", *params.Messages[0].Content)
	})
}

func TestServerlessInferenceChatCompletionText(t *testing.T) {
	content := "answer"
	c := &godo.ChatCompletion{
		Choices: []godo.ChatCompletionChoice{{
			Message: godo.ChatCompletionMessage{Content: &content},
		}},
	}
	assert.Equal(t, "answer", serverlessInferenceChatCompletionText(c))
}

func TestRunServerlessInferenceAsyncCreate_FromFlags(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, tm *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceAsyncCmd(), "create", config)
		config.Out = io.Discard
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "fal-ai/flux/schnell")
		config.Doit.Set(config.NS, doctl.ArgInferencePrompt, "sunset city")
		config.Doit.Set(config.NS, doctl.ArgTag, []string{"type=test"})

		expected := &godo.AsyncInvocation{RequestID: "req_1", Status: "QUEUED"}
		tm.inference.EXPECT().
			CreateAsyncInvocation(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ any, params *godo.AsyncInvocationNewParams) (*godo.AsyncInvocation, error) {
				assert.Equal(t, "fal-ai/flux/schnell", params.ModelID)
				assert.Equal(t, "sunset city", params.Input["prompt"])
				require.Len(t, params.Tags, 1)
				assert.Equal(t, "type", params.Tags[0].Key)
				assert.Equal(t, "test", params.Tags[0].Value)
				return expected, nil
			})

		err := RunServerlessInferenceAsyncCreate(config)
		require.NoError(t, err)
	})
}

func TestServerlessInferenceAsyncInvocationParams_FromRequestFile(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceAsyncCmd(), "create", config)
		body := `{"model_id":"fal-ai/fast-sdxl","input":{"prompt":"from file"}}`
		path := t.TempDir() + "/async-request.json"
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		config.Doit.Set(config.NS, doctl.ArgInferenceRequest, path)

		params, err := serverlessInferenceAsyncInvocationParams(config)
		require.NoError(t, err)
		assert.Equal(t, "fal-ai/fast-sdxl", params.ModelID)
		assert.Equal(t, "from file", params.Input["prompt"])
	})
}

func TestServerlessInferenceAsyncInvocationParams_RequiresInput(t *testing.T) {
	withTestClient(t, func(config *CmdConfig, _ *tcMocks) {
		serverlessInferenceCreateTestConfig(serverlessInferenceAsyncCmd(), "create", config)
		config.Doit.Set(config.NS, doctl.ArgInferenceModel, "fal-ai/flux/schnell")

		_, err := serverlessInferenceAsyncInvocationParams(config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), doctl.ArgInferencePrompt)
	})
}

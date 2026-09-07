package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The Serverless Inference catalog mixes agent-capable models with embedding,
// reranking, and media-generation ones, and its list endpoint reports no type,
// so the family is read off the id. This pins that heuristic against a real
// snapshot of the catalog: a false negative hides a model people want, and a
// false positive offers one whose first turn cannot work.
func TestIsGenerateChatModel(t *testing.T) {
	notChat := []string{
		"all-mini-lm-l6-v2",
		"bge-m3",
		"bge-reranker-v2-m3",
		"e5-large-v2",
		"gte-large-en-v1.5",
		"multi-qa-mpnet-base-dot-v1",
		"qwen3-embedding-0.6b",
		"openai-gpt-image-1",
		"openai-gpt-image-1.5",
		"openai-gpt-image-2",
		"stable-diffusion-3.5-large",
		"qwen3-tts-voicedesign",
		"wan2-2-t2v-a14b",
	}
	for _, id := range notChat {
		assert.False(t, isGenerateChatModel(id), "%s should be filtered out", id)
	}

	chat := []string{
		"anthropic-claude-4.5-sonnet",
		"anthropic-claude-opus-5",
		"arcee-trinity-large-thinking",
		"deepseek-v4-pro",
		"gemma-4-31B-it",
		"glm-5.3-flash",
		"kimi-k3",
		"llama-4-maverick",
		"mistral-3-14B",
		"nemotron-3-nano-omni",
		"nemotron-nano-12b-v2-vl",
		"nvidia-nemotron-3-super-120b",
		"openai-gpt-5.3-codex",
		"openai-gpt-oss-120b",
		"openai-o3",
		"qwen3.8-max",
		// The smart routers are chat endpoints, and one of them is aimed at
		// exactly this use case.
		"router:software-engineering",
	}
	for _, id := range chat {
		assert.True(t, isGenerateChatModel(id), "%s should be offered", id)
	}
}

// Every harness's preferred DigitalOcean model must survive the filter, or the
// picker would never be able to promote it to the Enter default.
func TestGenerateDOModelHintsAreChatModels(t *testing.T) {
	for _, h := range generateHarnessCatalog {
		if h.Inference == nil || h.Inference.DOModelHint == "" {
			continue
		}
		assert.True(t, isGenerateChatModel(h.Inference.DOModelHint), "%s: %s", h.ID, h.Inference.DOModelHint)
	}
}

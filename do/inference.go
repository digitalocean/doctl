/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package do

import (
	"context"

	"github.com/digitalocean/godo"
)

//go:generate go run go.uber.org/mock/mockgen -source inference.go -package=mocks -destination mocks/InferenceService.go InferenceService

// InferenceService calls the serverless inference API at https://inference.do-ai.run/.
type InferenceService interface {
	CreateChatCompletion(ctx context.Context, params *godo.ChatCompletionNewParams) (*godo.ChatCompletion, error)
	CreateChatCompletionStreaming(ctx context.Context, params *godo.ChatCompletionNewParams) (*godo.ChatCompletionStream, error)
	CreateEmbedding(ctx context.Context, params *godo.EmbeddingNewParams) (*godo.CreateEmbeddingResponse, error)
	GenerateImage(ctx context.Context, params *godo.ImageGenerateParams) (*godo.ImagesResponse, error)
	GenerateImageStreaming(ctx context.Context, params *godo.ImageGenerateParams) (*godo.ImageGenerationStream, error)
	CreateMessage(ctx context.Context, params *godo.MessageNewParams) (*godo.Message, error)
	CreateMessageStreaming(ctx context.Context, params *godo.MessageNewParams) (*godo.MessageStream, error)
	ListModels(ctx context.Context) (*godo.ModelList, error)
	CreateResponse(ctx context.Context, params *godo.ResponseNewParams) (*godo.ResponsesResponse, error)
	CreateResponseStreaming(ctx context.Context, params *godo.ResponseNewParams) (*godo.ResponseStream, error)
	CreateAsyncInvocation(ctx context.Context, params *godo.AsyncInvocationNewParams) (*godo.AsyncInvocation, error)
	GetAsyncInvocation(ctx context.Context, requestID string) (*godo.AsyncInvocation, error)
}

type inferenceService struct {
	client *godo.Client
}

var _ InferenceService = &inferenceService{}

// NewInferenceService builds an InferenceService instance.
func NewInferenceService(client *godo.Client) InferenceService {
	return &inferenceService{client: client}
}

func (s *inferenceService) CreateChatCompletion(ctx context.Context, params *godo.ChatCompletionNewParams) (*godo.ChatCompletion, error) {
	completion, _, err := s.client.Chat.Completions.New(ctx, params)
	return completion, err
}

func (s *inferenceService) CreateChatCompletionStreaming(ctx context.Context, params *godo.ChatCompletionNewParams) (*godo.ChatCompletionStream, error) {
	stream, _, err := s.client.Chat.Completions.NewStreaming(ctx, params)
	return stream, err
}

func (s *inferenceService) CreateEmbedding(ctx context.Context, params *godo.EmbeddingNewParams) (*godo.CreateEmbeddingResponse, error) {
	resp, _, err := s.client.Embeddings.New(ctx, params)
	return resp, err
}

func (s *inferenceService) GenerateImage(ctx context.Context, params *godo.ImageGenerateParams) (*godo.ImagesResponse, error) {
	resp, _, err := s.client.ImageGenerations.Generate(ctx, params)
	return resp, err
}

func (s *inferenceService) GenerateImageStreaming(ctx context.Context, params *godo.ImageGenerateParams) (*godo.ImageGenerationStream, error) {
	stream, _, err := s.client.ImageGenerations.GenerateStreaming(ctx, params)
	return stream, err
}

func (s *inferenceService) CreateMessage(ctx context.Context, params *godo.MessageNewParams) (*godo.Message, error) {
	msg, _, err := s.client.Messages.New(ctx, params)
	return msg, err
}

func (s *inferenceService) CreateMessageStreaming(ctx context.Context, params *godo.MessageNewParams) (*godo.MessageStream, error) {
	stream, _, err := s.client.Messages.NewStreaming(ctx, params)
	return stream, err
}

func (s *inferenceService) ListModels(ctx context.Context) (*godo.ModelList, error) {
	list, _, err := s.client.Models.List(ctx)
	return list, err
}

func (s *inferenceService) CreateResponse(ctx context.Context, params *godo.ResponseNewParams) (*godo.ResponsesResponse, error) {
	resp, _, err := s.client.Responses.New(ctx, params)
	return resp, err
}

func (s *inferenceService) CreateResponseStreaming(ctx context.Context, params *godo.ResponseNewParams) (*godo.ResponseStream, error) {
	stream, _, err := s.client.Responses.NewStreaming(ctx, params)
	return stream, err
}

func (s *inferenceService) CreateAsyncInvocation(ctx context.Context, params *godo.AsyncInvocationNewParams) (*godo.AsyncInvocation, error) {
	inv, _, err := s.client.AsyncInvocations.New(ctx, params)
	return inv, err
}

func (s *inferenceService) GetAsyncInvocation(ctx context.Context, requestID string) (*godo.AsyncInvocation, error) {
	inv, _, err := s.client.AsyncInvocations.Get(ctx, requestID)
	return inv, err
}

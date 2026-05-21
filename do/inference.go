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
	CreateChatCompletion(params *godo.ChatCompletionNewParams) (*godo.ChatCompletion, error)
	CreateChatCompletionStreaming(params *godo.ChatCompletionNewParams) (*godo.ChatCompletionStream, error)
	CreateEmbedding(params *godo.EmbeddingNewParams) (*godo.CreateEmbeddingResponse, error)
	GenerateImage(params *godo.ImageGenerateParams) (*godo.ImagesResponse, error)
	GenerateImageStreaming(params *godo.ImageGenerateParams) (*godo.ImageGenerationStream, error)
	CreateMessage(params *godo.MessageNewParams) (*godo.Message, error)
	CreateMessageStreaming(params *godo.MessageNewParams) (*godo.MessageStream, error)
	ListModels() (*godo.ModelList, error)
	CreateResponse(params *godo.ResponseNewParams) (*godo.ResponsesResponse, error)
	CreateResponseStreaming(params *godo.ResponseNewParams) (*godo.ResponseStream, error)
	CreateAsyncInvocation(params *godo.AsyncInvocationNewParams) (*godo.AsyncInvocation, error)
	GetAsyncInvocation(requestID string) (*godo.AsyncInvocation, error)
}

type inferenceService struct {
	client *godo.Client
}

var _ InferenceService = &inferenceService{}

// NewInferenceService builds an InferenceService instance.
func NewInferenceService(client *godo.Client) InferenceService {
	return &inferenceService{client: client}
}

func (s *inferenceService) CreateChatCompletion(params *godo.ChatCompletionNewParams) (*godo.ChatCompletion, error) {
	completion, _, err := s.client.Chat.Completions.New(context.TODO(), params)
	return completion, err
}

func (s *inferenceService) CreateChatCompletionStreaming(params *godo.ChatCompletionNewParams) (*godo.ChatCompletionStream, error) {
	stream, _, err := s.client.Chat.Completions.NewStreaming(context.TODO(), params)
	return stream, err
}

func (s *inferenceService) CreateEmbedding(params *godo.EmbeddingNewParams) (*godo.CreateEmbeddingResponse, error) {
	resp, _, err := s.client.Embeddings.New(context.TODO(), params)
	return resp, err
}

func (s *inferenceService) GenerateImage(params *godo.ImageGenerateParams) (*godo.ImagesResponse, error) {
	resp, _, err := s.client.ImageGenerations.Generate(context.TODO(), params)
	return resp, err
}

func (s *inferenceService) GenerateImageStreaming(params *godo.ImageGenerateParams) (*godo.ImageGenerationStream, error) {
	stream, _, err := s.client.ImageGenerations.GenerateStreaming(context.TODO(), params)
	return stream, err
}

func (s *inferenceService) CreateMessage(params *godo.MessageNewParams) (*godo.Message, error) {
	msg, _, err := s.client.Messages.New(context.TODO(), params)
	return msg, err
}

func (s *inferenceService) CreateMessageStreaming(params *godo.MessageNewParams) (*godo.MessageStream, error) {
	stream, _, err := s.client.Messages.NewStreaming(context.TODO(), params)
	return stream, err
}

func (s *inferenceService) ListModels() (*godo.ModelList, error) {
	list, _, err := s.client.Models.List(context.TODO())
	return list, err
}

func (s *inferenceService) CreateResponse(params *godo.ResponseNewParams) (*godo.ResponsesResponse, error) {
	resp, _, err := s.client.Responses.New(context.TODO(), params)
	return resp, err
}

func (s *inferenceService) CreateResponseStreaming(params *godo.ResponseNewParams) (*godo.ResponseStream, error) {
	stream, _, err := s.client.Responses.NewStreaming(context.TODO(), params)
	return stream, err
}

func (s *inferenceService) CreateAsyncInvocation(params *godo.AsyncInvocationNewParams) (*godo.AsyncInvocation, error) {
	inv, _, err := s.client.AsyncInvocations.New(context.TODO(), params)
	return inv, err
}

func (s *inferenceService) GetAsyncInvocation(requestID string) (*godo.AsyncInvocation, error) {
	inv, _, err := s.client.AsyncInvocations.Get(context.TODO(), requestID)
	return inv, err
}

package ollama

import (
	"context"
	"errors"
	"strings"
	"testing"

	ollamaPKG "github.com/luoxiaojun1992/ai-agent/pkg/ollama"
)

type mockOllamaClient struct {
	resp   *ollamaPKG.EmbedResponse
	err    error
	called bool
}

func (m *mockOllamaClient) EmbeddingPrompt(embedReq *ollamaPKG.EmbedRequest) (*ollamaPKG.EmbedResponse, error) {
	_ = embedReq
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func (m *mockOllamaClient) Talk(chatReq *ollamaPKG.ChatRequest, callback func(response string) error) error {
	_, _ = chatReq, callback
	return nil
}

func TestEmbedding_Do_Success(t *testing.T) {
	cli := &mockOllamaClient{resp: &ollamaPKG.EmbedResponse{Embeddings: [][]float32{{0.1}}}}
	s := &Embedding{OllamaCli: cli}

	called := false
	err := s.Do(context.Background(), map[string]any{
		"model":   "m",
		"content": "hello",
	}, func(output any) (any, error) {
		called = true
		if _, ok := output.(*ollamaPKG.EmbedResponse); !ok {
			t.Fatalf("unexpected output type: %T", output)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called || !cli.called {
		t.Fatalf("expected callback and client call")
	}
}

func TestEmbedding_Do_EmptyResponse(t *testing.T) {
	s := &Embedding{OllamaCli: &mockOllamaClient{resp: nil}}
	err := s.Do(context.Background(), map[string]any{
		"model":   "m",
		"content": "hello",
	}, func(output any) (any, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected empty response error")
	}
}

func TestEmbedding_Do_ClientError(t *testing.T) {
	expected := errors.New("embedding failed")
	s := &Embedding{OllamaCli: &mockOllamaClient{err: expected}}
	err := s.Do(context.Background(), map[string]any{
		"model":   "m",
		"content": "hello",
	}, func(output any) (any, error) { return nil, nil })
	if !errors.Is(err, expected) {
		t.Fatalf("expected client error, got: %v", err)
	}
}

func TestEmbedding_Descriptions(t *testing.T) {
	e := &Embedding{}
	desc, err := e.GetDescription()
	if err != nil || desc == "" || e.ShortDescription() == "" {
		t.Fatalf("descriptions should not be empty")
	}
	if !strings.Contains(desc, "Parameters") {
		t.Fatalf("expected details in description")
	}
}

func TestEmbedding_Do_InvalidParams(t *testing.T) {
	e := &Embedding{OllamaCli: &mockOllamaClient{}}
	if err := e.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
}

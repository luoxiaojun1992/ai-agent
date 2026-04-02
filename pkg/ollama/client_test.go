package ollama

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_EmbeddingPrompt_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"model":"m","embeddings":[[0.1,0.2]]}`))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	resp, err := cli.EmbeddingPrompt(&EmbedRequest{Model: "m", Input: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Embeddings) != 1 || len(resp.Embeddings[0]) != 2 {
		t.Fatalf("unexpected embedding response: %+v", resp)
	}
}

func TestClient_DefaultAPIType_UsesOllamaEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"ok\"},\"done\":true}\n"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	err := cli.Talk(&ChatRequest{Model: "m"}, func(response string) error {
		_ = response
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_EmbeddingPrompt_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	if _, err := cli.EmbeddingPrompt(&EmbedRequest{Model: "m", Input: "hello"}); err == nil {
		t.Fatalf("expected non-200 error")
	}
}

func TestClient_OpenAICompatible_EmbeddingPrompt_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req["model"] != "m" || req["input"] != "hello" {
			t.Fatalf("unexpected request body: %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"model":"m","data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL, APIType: "openai", APIKey: "secret-key"})
	resp, err := cli.EmbeddingPrompt(&EmbedRequest{Model: "m", Input: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Embeddings) != 1 || len(resp.Embeddings[0]) != 2 {
		t.Fatalf("unexpected embedding response: %+v", resp)
	}
}

func TestClient_Talk_StreamSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"hello \"},\"done\":false}\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"world\"},\"done\":true}\n"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	var sb strings.Builder
	err := cli.Talk(&ChatRequest{Model: "m"}, func(response string) error {
		sb.WriteString(response)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.String() != "hello world" {
		t.Fatalf("unexpected streamed content: %s", sb.String())
	}
}

func TestClient_OpenAICompatible_Talk_StreamSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		if req["stream"] != true {
			t.Fatalf("expected stream=true, got: %v", req["stream"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"world\"},\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL, APIType: "openai", APIKey: "secret-key"})
	var sb strings.Builder
	err := cli.Talk(&ChatRequest{
		Model: "m",
		Options: &ChatRequestOptions{
			Temperature: 0.2,
		},
	}, func(response string) error {
		sb.WriteString(response)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.String() != "hello world" {
		t.Fatalf("unexpected streamed content: %s", sb.String())
	}
}

func TestClient_Talk_CallbackError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"chunk\"},\"done\":true}\n"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	expected := errors.New("stop")
	err := cli.Talk(&ChatRequest{Model: "m"}, func(response string) error {
		_ = response
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestClient_Talk_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	if err := cli.Talk(&ChatRequest{Model: "m"}, func(response string) error { return nil }); err == nil {
		t.Fatalf("expected non-200 talk error")
	}
}

func TestClient_Talk_IgnoreInvalidJsonLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json\n"))
		_, _ = w.Write([]byte("{\"message\":{\"content\":\"ok\"},\"done\":true}\n"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	var sb strings.Builder
	err := cli.Talk(&ChatRequest{Model: "m"}, func(response string) error {
		sb.WriteString(response)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.String() != "ok" {
		t.Fatalf("expected only valid json line to be processed, got: %s", sb.String())
	}
}

func TestClient_EmbeddingPrompt_InvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid-json"))
	}))
	defer server.Close()

	cli := NewClient(&Config{Host: server.URL})
	if _, err := cli.EmbeddingPrompt(&EmbedRequest{Model: "m", Input: "x"}); err == nil {
		t.Fatalf("expected unmarshal error")
	}
}

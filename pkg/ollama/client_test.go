package ollama

import (
	"errors"
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

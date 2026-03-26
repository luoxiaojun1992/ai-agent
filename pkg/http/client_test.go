package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestClient_SendRequest_WithBaseURLQueryHeadersAndJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/echo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("q"); got != "v" {
			t.Fatalf("unexpected query value: %s", got)
		}
		if got := r.Header.Get("X-Test"); got != "ok" {
			t.Fatalf("unexpected header value: %s", got)
		}
		if r.Header.Get(HeaderContentType) != ContentTypeJson {
			t.Fatalf("expected content-type %s", ContentTypeJson)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cli := NewHTTPClient(5*time.Second, true, 5)
	cli.SetBaseURL(server.URL)
	cli.AddDefaultHeader("X-Test", "ok")

	res, err := cli.SendRequest(http.MethodPost, "/echo", map[string]any{"a": 1}, url.Values{"q": []string{"v"}}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body, &payload); err != nil {
		t.Fatalf("unexpected json body: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true")
	}
}

func TestClient_SendRequest_NoRedirectWhenDisabled(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer redirectServer.Close()

	cli := NewHTTPClient(5*time.Second, false, 0)
	res, err := cli.Get(redirectServer.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 when redirects disabled, got %d", res.StatusCode)
	}
}

func TestClient_SendRequest_InvalidURL(t *testing.T) {
	cli := NewHTTPClient(5*time.Second, true, 5)
	if _, err := cli.Get("://bad-url", nil, nil); err == nil {
		t.Fatalf("expected invalid url error")
	}
}

func TestClient_SendRequest_RejectsUnsafeURL(t *testing.T) {
	cli := NewHTTPClient(5*time.Second, true, 5)

	if _, err := cli.Get("ftp://example.com", nil, nil); err == nil {
		t.Fatalf("expected unsupported scheme error")
	}

	if _, err := cli.Get("http://user:pass@example.com", nil, nil); err == nil {
		t.Fatalf("expected user info rejection error")
	}
}

func TestClient_WrapperMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.Method))
	}))
	defer server.Close()

	cli := NewHTTPClient(5*time.Second, true, 2)
	if res, err := cli.Post(server.URL, "x", nil, nil); err != nil || string(res.Body) != http.MethodPost {
		t.Fatalf("post failed: err=%v body=%s", err, string(res.Body))
	}
	if res, err := cli.Patch(server.URL, "x", nil, nil); err != nil || string(res.Body) != http.MethodPatch {
		t.Fatalf("patch failed: err=%v body=%s", err, string(res.Body))
	}
	if res, err := cli.Delete(server.URL, "x", nil, nil); err != nil || string(res.Body) != http.MethodDelete {
		t.Fatalf("delete failed: err=%v body=%s", err, string(res.Body))
	}
}

func TestClient_SendRequest_BytesBodyAndBadURLForQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !bytes.Equal(b, []byte("abc")) {
			t.Fatalf("unexpected body: %s", string(b))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cli := NewHTTPClient(5*time.Second, true, 5)
	if _, err := cli.SendRequest(http.MethodPost, server.URL, []byte("abc"), nil, nil); err != nil {
		t.Fatalf("expected bytes body to work: %v", err)
	}

	if _, err := cli.SendRequest(http.MethodGet, "://bad-url", nil, url.Values{"a": []string{"1"}}, nil); err == nil {
		t.Fatalf("expected url parse error when query params are provided")
	}
}

func TestClient_SendRequest_InvalidJSONBodyMarshal(t *testing.T) {
	cli := NewHTTPClient(5*time.Second, true, 5)
	badBody := map[string]any{"x": make(chan int)}
	if _, err := cli.SendRequest(http.MethodPost, "http://localhost", badBody, nil, nil); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestClient_NewHTTPClient_MaxRedirectsExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	defer server.Close()

	cli := NewHTTPClient(5*time.Second, true, 0)
	_, err := cli.Get(server.URL, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "too many redirects") {
		t.Fatalf("expected too many redirects error, got: %v", err)
	}
}

func TestClient_SendRequest_StringBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if string(b) != "hello" {
			t.Fatalf("expected string body 'hello', got: %s", string(b))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cli := NewHTTPClient(5*time.Second, true, 5)
	if _, err := cli.SendRequest(http.MethodPost, server.URL, "hello", nil, nil); err != nil {
		t.Fatalf("unexpected error with string body: %v", err)
	}
}

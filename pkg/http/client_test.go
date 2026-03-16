package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

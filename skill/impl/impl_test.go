package impl

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	ai_agent "github.com/luoxiaojun1992/ai-agent"
	httpPKG "github.com/luoxiaojun1992/ai-agent/pkg/http"
	mcpPkg "github.com/luoxiaojun1992/ai-agent/pkg/mcp"
)

type mockHTTPClient struct {
	response *httpPKG.Response
	err      error

	method string
	path   string
}

func (m *mockHTTPClient) SetBaseURL(baseURL string)          { _ = baseURL }
func (m *mockHTTPClient) AddDefaultHeader(key, value string) { _, _ = key, value }
func (m *mockHTTPClient) Get(path string, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	return m.SendRequest("GET", path, nil, queryParams, headers)
}
func (m *mockHTTPClient) Post(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	return m.SendRequest("POST", path, body, queryParams, headers)
}
func (m *mockHTTPClient) Patch(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	return m.SendRequest("PATCH", path, body, queryParams, headers)
}
func (m *mockHTTPClient) Delete(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	return m.SendRequest("DELETE", path, body, queryParams, headers)
}
func (m *mockHTTPClient) SendRequest(method, path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _ = body, queryParams, headers
	m.method = method
	m.path = path
	if m.err != nil {
		return nil, m.err
	}
	if m.response == nil {
		return &httpPKG.Response{StatusCode: 200}, nil
	}
	return m.response, nil
}

func TestHTTP_Do_Success(t *testing.T) {
	cli := &mockHTTPClient{response: &httpPKG.Response{StatusCode: 200}}
	s := &Http{Client: cli, AllowedURLList: []string{"https://api.example.com"}}

	called := false
	err := s.Do(context.Background(), map[string]any{
		"method":       "POST",
		"path":         "https://api.example.com",
		"body":         "{}",
		"query_params": url.Values{"a": []string{"1"}},
		"http_header":  http.Header{"X": []string{"Y"}},
	}, func(output any) (any, error) {
		called = true
		if _, ok := output.(*httpPKG.Response); !ok {
			t.Fatalf("unexpected callback output type: %T", output)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected callback called")
	}
	if cli.method != "POST" || cli.path != "https://api.example.com" {
		t.Fatalf("unexpected request captured: %s %s", cli.method, cli.path)
	}
}

func TestHTTP_Do_PathNotAllowed(t *testing.T) {
	s := &Http{Client: &mockHTTPClient{}, AllowedURLList: []string{"https://ok"}}
	err := s.Do(context.Background(), map[string]any{
		"method":       "GET",
		"path":         "https://blocked",
		"body":         "",
		"query_params": url.Values{},
		"http_header":  http.Header{},
	}, nil)
	if err == nil {
		t.Fatalf("expected path not allowed error")
	}
}

func TestMCP_Do_ValidationErrors(t *testing.T) {
	m := &MCP{}
	if err := m.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid param error")
	}
	if err := m.Do(context.Background(), map[string]any{"name": 1, "arguments": map[string]interface{}{}}, nil); err == nil {
		t.Fatalf("expected invalid name error")
	}
	if err := m.Do(context.Background(), map[string]any{"name": "n", "arguments": 1}, nil); err == nil {
		t.Fatalf("expected invalid arguments error")
	}
}

func TestTeam_Do_Errors(t *testing.T) {
	if err := (&Team{}).Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
	if err := (&Team{}).Do(context.Background(), map[string]any{}, nil); err == nil {
		t.Fatalf("expected missing member error")
	}
	teamWithMember := &Team{Members: map[string]*ai_agent.AgentDouble{"m": nil}}
	if err := teamWithMember.Do(context.Background(), map[string]any{"member": "m"}, nil); err == nil {
		t.Fatalf("expected missing message error")
	}
	teamWithMap := &Team{Members: map[string]*ai_agent.AgentDouble{}}
	err := teamWithMap.Do(context.Background(), map[string]any{"member": "missing", "message": "hi"}, nil)
	if err == nil {
		t.Fatalf("expected member not found error")
	}
}

func TestDescriptions_Basic(t *testing.T) {
	h := &Http{}
	hDesc, err := h.GetDescription()
	if err != nil || hDesc == "" || h.ShortDescription() == "" {
		t.Fatalf("http descriptions should not be empty")
	}

	team := &Team{Members: map[string]*ai_agent.AgentDouble{}}
	desc, err := team.GetDescription()
	if err != nil {
		t.Fatalf("unexpected team description error: %v", err)
	}
	if desc == "" || !strings.Contains(desc, "Team Members") {
		t.Fatalf("unexpected team description: %s", desc)
	}
}

func TestMCP_GetDescription_ErrorOnListFailure(t *testing.T) {
	client, err := mcpPkg.NewClient(&mcpPkg.Config{Host: "http://127.0.0.1:1", ClientType: mcpPkg.ClientTypeSSE})
	if err != nil {
		t.Fatalf("unexpected new client error: %v", err)
	}
	m := &MCP{MCPClient: client}
	if _, err := m.GetDescription(); err == nil {
		t.Fatalf("expected get description error when list tools fails")
	}
	if m.ShortDescription() == "" {
		t.Fatalf("short description should not be empty")
	}
}

package impl

import (
	"context"
	"errors"
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

func TestHTTP_Do_AllParamErrors(t *testing.T) {
	cli := &mockHTTPClient{}
	s := &Http{Client: cli}

	if err := s.Do(context.Background(), "bad", nil); err == nil {
		t.Fatalf("expected invalid params error")
	}
	base := func(extra map[string]any) map[string]any {
		m := map[string]any{
			"method":       "GET",
			"path":         "x",
			"body":         "",
			"query_params": url.Values{},
			"http_header":  http.Header{},
		}
		for k, v := range extra {
			delete(m, k)
			if v != nil {
				m[k] = v
			}
		}
		return m
	}
	if err := s.Do(context.Background(), base(map[string]any{"method": nil}), nil); err == nil {
		t.Fatalf("expected missing method error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"method": 1}), nil); err == nil {
		t.Fatalf("expected method type error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"path": nil}), nil); err == nil {
		t.Fatalf("expected missing path error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"path": 1}), nil); err == nil {
		t.Fatalf("expected path type error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"body": nil}), nil); err == nil {
		t.Fatalf("expected missing body error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"body": 1}), nil); err == nil {
		t.Fatalf("expected body type error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"query_params": nil}), nil); err == nil {
		t.Fatalf("expected missing query_params error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"query_params": "bad"}), nil); err == nil {
		t.Fatalf("expected query_params type error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"http_header": nil}), nil); err == nil {
		t.Fatalf("expected missing http_header error")
	}
	if err := s.Do(context.Background(), base(map[string]any{"http_header": "bad"}), nil); err == nil {
		t.Fatalf("expected http_header type error")
	}
}

func TestHTTP_Do_ClientError(t *testing.T) {
	s := &Http{Client: &mockHTTPClient{err: errors.New("network error")}}
	err := s.Do(context.Background(), map[string]any{
		"method": "GET", "path": "x", "body": "", "query_params": url.Values{}, "http_header": http.Header{},
	}, func(any) (any, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expected client error")
	}
}

func TestHTTP_Do_CallbackError(t *testing.T) {
	expected := errors.New("callback error")
	s := &Http{Client: &mockHTTPClient{response: &httpPKG.Response{StatusCode: 200}}}
	err := s.Do(context.Background(), map[string]any{
		"method": "GET", "path": "x", "body": "", "query_params": url.Values{}, "http_header": http.Header{},
	}, func(any) (any, error) { return nil, expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestMCP_Do_MissingNameAndArguments(t *testing.T) {
	m := &MCP{}
	if err := m.Do(context.Background(), map[string]any{"arguments": map[string]interface{}{}}, nil); err == nil {
		t.Fatalf("expected missing name error")
	}
	if err := m.Do(context.Background(), map[string]any{"name": "n"}, nil); err == nil {
		t.Fatalf("expected missing arguments error")
	}
}

func TestTeam_Do_MemberNotStringAndMessageNotString(t *testing.T) {
	team := &Team{Members: map[string]*ai_agent.AgentDouble{}}
	if err := team.Do(context.Background(), map[string]any{"member": 1}, nil); err == nil {
		t.Fatalf("expected member type error")
	}
	teamWithMember := &Team{Members: map[string]*ai_agent.AgentDouble{"m": nil}}
	if err := teamWithMember.Do(context.Background(), map[string]any{"member": "m", "message": 123}, nil); err == nil {
		t.Fatalf("expected message type error")
	}
}

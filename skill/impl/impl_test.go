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
	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
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

type mockMCPClient struct {
	listToolsResp []string
	listToolsErr  error

	callToolResp []string
	callToolErr  error

	calledName string
	calledArgs map[string]interface{}
}

func (m *mockMCPClient) Initialize(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *mockMCPClient) Close() error { return nil }

func (m *mockMCPClient) ListTools(ctx context.Context) ([]string, error) {
	_ = ctx
	if m.listToolsErr != nil {
		return nil, m.listToolsErr
	}
	return m.listToolsResp, nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, name string, arguments map[string]interface{}) ([]string, error) {
	_ = ctx
	m.calledName = name
	m.calledArgs = arguments
	if m.callToolErr != nil {
		return nil, m.callToolErr
	}
	return m.callToolResp, nil
}

func TestMCP_GetDescription_SuccessAndError(t *testing.T) {
	m1 := &MCP{MCPClient: &mockMCPClient{listToolsResp: []string{"tool-a", "tool-b"}}}
	desc, err := m1.GetDescription()
	if err != nil {
		t.Fatalf("unexpected get description error: %v", err)
	}
	if !strings.Contains(desc, "tool-a") || !strings.Contains(desc, "tool-b") {
		t.Fatalf("expected tool list in description, got: %s", desc)
	}
	if m1.ShortDescription() == "" {
		t.Fatalf("short description should not be empty")
	}

	m2 := &MCP{MCPClient: &mockMCPClient{listToolsErr: errors.New("list tools failed")}}
	if _, err := m2.GetDescription(); err == nil {
		t.Fatalf("expected get description error")
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

func TestMCP_Do_SuccessAndCallbackError(t *testing.T) {
	mockCli := &mockMCPClient{callToolResp: []string{"ok"}}
	m := &MCP{MCPClient: mockCli}

	called := false
	err := m.Do(context.Background(), map[string]any{
		"name":      "tool-a",
		"arguments": map[string]interface{}{"k": "v"},
	}, func(output any) (any, error) {
		called = true
		if _, ok := output.([]string); !ok {
			t.Fatalf("expected []string callback output, got %T", output)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected do error: %v", err)
	}
	if !called {
		t.Fatalf("expected callback called")
	}
	if mockCli.calledName != "tool-a" {
		t.Fatalf("expected call tool name captured")
	}

	expected := errors.New("callback failed")
	err = m.Do(context.Background(), map[string]any{
		"name":      "tool-a",
		"arguments": map[string]interface{}{},
	}, func(output any) (any, error) {
		_ = output
		return nil, expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestMCP_Do_CallToolError(t *testing.T) {
	m := &MCP{MCPClient: &mockMCPClient{callToolErr: errors.New("call failed")}}
	err := m.Do(context.Background(), map[string]any{
		"name":      "tool-a",
		"arguments": map[string]interface{}{},
	}, func(output any) (any, error) {
		_ = output
		return nil, nil
	})
	if err == nil {
		t.Fatalf("expected call tool error")
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

type mockTeamOllamaClient struct{}

func (m *mockTeamOllamaClient) EmbeddingPrompt(embedReq *ollama.EmbedRequest) (*ollama.EmbedResponse, error) {
	_ = embedReq
	return &ollama.EmbedResponse{Embeddings: [][]float32{}}, nil
}

func (m *mockTeamOllamaClient) Talk(chatReq *ollama.ChatRequest, callback func(response string) error) error {
	_ = chatReq
	if callback != nil {
		return callback("member-response")
	}
	return nil
}

type mockTeamMilvusClient struct{}

func (m *mockTeamMilvusClient) InsertVector(ctx context.Context, collectionName, content string, vector []float32) error {
	_, _, _, _ = ctx, collectionName, content, vector
	return nil
}

func (m *mockTeamMilvusClient) SearchVector(ctx context.Context, collectionName string, vector []float32) ([]string, error) {
	_, _, _ = ctx, collectionName, vector
	return nil, nil
}

func (m *mockTeamMilvusClient) Close() error { return nil }

func TestTeam_Do_Success(t *testing.T) {
	config := &ai_agent.Config{
		ChatModel:        "m",
		SupervisorModel:  "m",
		EmbeddingModel:   "e",
		MilvusCollection: "c",
		AgentMode:        ai_agent.AgentModeChat,
	}
	agent, err := ai_agent.NewAgent(context.Background(), func(option *ai_agent.AgentOption) {
		option.SetConfig(config)
		option.SetOllamaCli(&mockTeamOllamaClient{})
		option.SetMilvusCli(&mockTeamMilvusClient{})
		option.SetHttpCli(&mockHTTPClient{})
		option.SetCharacter("agent")
		option.SetRole("assistant")
	})
	if err != nil {
		t.Fatalf("unexpected NewAgent error: %v", err)
	}

	member, err := ai_agent.NewAgentDouble(context.Background(), func(option *ai_agent.AgentDoubleOption) {
		option.SetConfig(config)
		option.SetAgent(agent)
		option.SetCharacter("member")
		option.SetRole("assistant")
	})
	if err != nil {
		t.Fatalf("unexpected NewAgentDouble error: %v", err)
	}

	team := &Team{Members: map[string]*ai_agent.AgentDouble{"m": member}}
	called := false
	err = team.Do(context.Background(), map[string]any{"member": "m", "message": "hello"}, func(output any) (any, error) {
		called = true
		if _, ok := output.(string); !ok {
			t.Fatalf("expected string callback output, got %T", output)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected Team.Do error: %v", err)
	}
	if !called {
		t.Fatalf("expected callback to be called")
	}

	desc, err := team.GetDescription()
	if err != nil {
		t.Fatalf("unexpected Team.GetDescription error: %v", err)
	}
	if !strings.Contains(desc, "assistant") {
		t.Fatalf("expected team description to include member description")
	}
}

var _ milvus.IClient = (*mockTeamMilvusClient)(nil)

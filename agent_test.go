package ai_agent

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	httpPKG "github.com/luoxiaojun1992/ai-agent/pkg/http"
	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
	"github.com/luoxiaojun1992/ai-agent/skill"
)

type mockSkill struct {
	called bool
	err    error
}

func (m *mockSkill) GetDescription() string { return "mock-skill" }

func (m *mockSkill) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	_, _ = ctx, cmdCtx
	m.called = true
	if m.err != nil {
		return m.err
	}
	if callback != nil {
		_, _ = callback("mock-output")
	}
	return nil
}

type mockOllamaClient struct {
	talkChunks []string
	talkErr    error

	embedResp *ollama.EmbedResponse
	embedErr  error
}

func (m *mockOllamaClient) EmbeddingPrompt(embedReq *ollama.EmbedRequest) (*ollama.EmbedResponse, error) {
	_ = embedReq
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	if m.embedResp != nil {
		return m.embedResp, nil
	}
	return &ollama.EmbedResponse{}, nil
}

func (m *mockOllamaClient) Talk(chatReq *ollama.ChatRequest, callback func(response string) error) error {
	_ = chatReq
	if m.talkErr != nil {
		return m.talkErr
	}
	for _, chunk := range m.talkChunks {
		if err := callback(chunk); err != nil {
			return err
		}
	}
	return nil
}

type mockMilvusClient struct {
	insertCalled bool
	searchCalled bool

	insertErr error
	searchErr error

	searchResult []string
}

func (m *mockMilvusClient) InsertVector(ctx context.Context, collectionName, content string, vector []float32) error {
	_, _, _, _ = ctx, collectionName, content, vector
	m.insertCalled = true
	return m.insertErr
}

func (m *mockMilvusClient) SearchVector(ctx context.Context, collectionName string, vector []float32) ([]string, error) {
	_, _, _ = ctx, collectionName, vector
	m.searchCalled = true
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResult, nil
}

func (m *mockMilvusClient) Close() error { return nil }

type mockHTTPClient struct {
	resp *httpPKG.Response
	err  error
}

func (m *mockHTTPClient) SetBaseURL(baseURL string) { _ = baseURL }
func (m *mockHTTPClient) AddDefaultHeader(key, value string) {
	_, _ = key, value
}
func (m *mockHTTPClient) Get(path string, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _ = path, queryParams, headers
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	return &httpPKG.Response{StatusCode: 200}, nil
}
func (m *mockHTTPClient) Post(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _, _ = path, body, queryParams, headers
	return nil, nil
}
func (m *mockHTTPClient) Patch(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _, _ = path, body, queryParams, headers
	return nil, nil
}
func (m *mockHTTPClient) Delete(path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _, _ = path, body, queryParams, headers
	return nil, nil
}
func (m *mockHTTPClient) SendRequest(method, path string, body any, queryParams url.Values, headers http.Header) (*httpPKG.Response, error) {
	_, _, _, _, _ = method, path, body, queryParams, headers
	return nil, nil
}

func testConfig() *Config {
	return &Config{
		ChatModel:              "qwen3:4b",
		EmbeddingModel:         "nomic-embed-text",
		SupervisorModel:        "qwen3:4b",
		HttpTimeout:            2 * time.Second,
		ChatModelContextLimit:  64,
		ContextReserveTokens:   8,
		NearDuplicateThreshold: 0.85,
		MilvusCollection:       "memory",
		AgentMode:              AgentModeChat,
		AgentLoopDuration:      time.Millisecond,
	}
}

func newAgentDoubleWithMocks(t *testing.T) (*AgentDouble, *mockOllamaClient, *mockMilvusClient, *mockHTTPClient) {
	t.Helper()
	ollamaCli := &mockOllamaClient{}
	milvusCli := &mockMilvusClient{}
	httpCli := &mockHTTPClient{}

	agent, err := NewAgent(context.Background(), func(option *AgentOption) {
		option.SetConfig(testConfig())
		option.SetOllamaCli(ollamaCli)
		option.SetMilvusCli(milvusCli)
		option.SetHttpCli(httpCli)
		option.SetCharacter("agent")
		option.SetRole("assistant")
	})
	if err != nil {
		t.Fatalf("new agent failed: %v", err)
	}

	ad, err := NewAgentDouble(context.Background(), func(option *AgentDoubleOption) {
		option.SetConfig(testConfig())
		option.SetAgent(agent)
		option.SetCharacter("double")
		option.SetRole("lead")
	})
	if err != nil {
		t.Fatalf("new agent double failed: %v", err)
	}

	return ad, ollamaCli, milvusCli, httpCli
}

func TestNewAgent_RequiresConfig(t *testing.T) {
	if _, err := NewAgent(context.Background()); err == nil {
		t.Fatalf("expected invalid agent config error")
	}
}

func TestNewAgentDouble_RequiresConfig(t *testing.T) {
	if _, err := NewAgentDouble(context.Background()); err == nil {
		t.Fatalf("expected invalid agent double config error")
	}
}

func TestAgent_CommandSkillNotFound(t *testing.T) {
	a := &Agent{skillSet: map[string]skill.Skill{}}
	if err := a.Command(context.Background(), "missing", nil, nil); err == nil {
		t.Fatalf("expected missing skill error")
	}
}

func TestAgentDouble_MemorySnapshotLoadAndForget(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.AddUserMemory("u1", nil).AddAssistantMemory("a1", nil).AddToolMemory("t1", nil)

	snapshot := ad.MemorySnapshot()
	if len(snapshot.Contexts) != 3 {
		t.Fatalf("expected snapshot len 3, got %d", len(snapshot.Contexts))
	}

	ad.Forget(2)
	if len(ad.memory.Contexts) != 1 {
		t.Fatalf("expected len 1 after forget, got %d", len(ad.memory.Contexts))
	}

	ad.LoadMemory(snapshot)
	if len(ad.memory.Contexts) != 3 {
		t.Fatalf("expected len 3 after load, got %d", len(ad.memory.Contexts))
	}

	ad.Forget(-1)
	if len(ad.memory.Contexts) != 0 {
		t.Fatalf("expected all memory removed")
	}
}

func TestAgentDouble_CompressContextByTokenBudget(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.config.ChatModelContextLimit = 20
	ad.config.ContextReserveTokens = 2
	ad.config.NearDuplicateThreshold = 0.70

	ad.AddSystemMemory("fixed system instruction", nil)
	ad.AddAssistantMemory("the weather in beijing is sunny today", nil)
	ad.AddAssistantMemory("the weather in beijing is sunny today", nil)
	ad.AddUserMemory("latest query", nil)

	ad.compressContextByTokenBudget()

	if len(ad.memory.Contexts) == 0 {
		t.Fatalf("expected contexts after compression")
	}
	hasSystem := false
	hasLatestUser := false
	for _, c := range ad.memory.Contexts {
		if c.Role == "system" {
			hasSystem = true
		}
		if c.Role == "user" && c.Content == "latest query" {
			hasLatestUser = true
		}
	}
	if !hasSystem || !hasLatestUser {
		t.Fatalf("expected protected memory entries retained")
	}
}

func TestAgentDouble_RememberAndRecall(t *testing.T) {
	ad, ollamaCli, milvusCli, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{{0.1, 0.2}}}
	milvusCli.searchResult = []string{"ctx1"}

	if err := ad.Remember(context.Background(), "hello"); err != nil {
		t.Fatalf("remember failed: %v", err)
	}
	if !milvusCli.insertCalled {
		t.Fatalf("expected insert called")
	}

	ctxs, err := ad.Recall(context.Background(), "q")
	if err != nil {
		t.Fatalf("recall failed: %v", err)
	}
	if !milvusCli.searchCalled || len(ctxs) != 1 || ctxs[0] != "ctx1" {
		t.Fatalf("unexpected recall result: %#v", ctxs)
	}
}

func TestAgentDouble_Read(t *testing.T) {
	ad, _, _, httpCli := newAgentDoubleWithMocks(t)
	httpCli.resp = &httpPKG.Response{StatusCode: 200, Body: []byte("page-content")}

	if err := ad.Read("https://example.com"); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(ad.memory.Contexts) == 0 || ad.memory.Contexts[len(ad.memory.Contexts)-1].Content != "page-content" {
		t.Fatalf("expected response body appended to memory")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_FunctionCallFlow(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ms := &mockSkill{}
	ad.skillSet["echo"] = ms

	ollamaCli.talkChunks = []string{`<tool>{"function":"echo","context":{"msg":"hi"},"abort_on_error":true}</tool>`}
	ad.AddUserMemory("trigger", nil)

	var outputs []string
	err := ad.talkToOllamaWithMemory(context.Background(), func(response string) error {
		outputs = append(outputs, response)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.called {
		t.Fatalf("expected skill execution")
	}
	if !strings.Contains(strings.Join(outputs, "\n"), "executed successfully") {
		t.Fatalf("expected success output in callback stream")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_InvalidFunctionJson(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ollamaCli.talkChunks = []string{"<tool>{invalid}</tool>"}
	ad.AddUserMemory("trigger", nil)

	err := ad.talkToOllamaWithMemory(context.Background(), func(response string) error { return nil })
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_SupervisorNonCompliant(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ad.config.SupervisorSwitch = true
	ad.config.SupervisorModel = "supervisor"
	ollamaCli.talkChunks = []string{"model response"}
	ad.AddUserMemory("trigger", nil)

	err := ad.talkToOllamaWithMemory(context.Background(), func(response string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "non-compliant") {
		t.Fatalf("expected non-compliant error, got: %v", err)
	}
}

func TestAgentDouble_RememberEmbeddingError(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedErr = errors.New("embedding failed")
	if err := ad.Remember(context.Background(), "x"); err == nil {
		t.Fatalf("expected embedding error")
	}
}

func TestAgentDouble_RecallMilvusError(t *testing.T) {
	ad, ollamaCli, milvusCli, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{{0.1}}}
	milvusCli.searchErr = errors.New("search failed")
	if _, err := ad.Recall(context.Background(), "q"); err == nil {
		t.Fatalf("expected recall error")
	}
}

var _ milvus.IClient = (*mockMilvusClient)(nil)

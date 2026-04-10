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
	called  bool
	err     error
	descErr error
}

func (m *mockSkill) GetDescription() (string, error) { return "mock-skill", m.descErr }
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
func (m *mockHTTPClient) SetAllowedURLList(urlList []string) {
	_ = urlList
}
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

func TestAgentDouble_MemorySnapshotWithLimit(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.AddUserMemory("u1", nil).
		AddAssistantMemory("a1", nil).
		AddToolMemory("t1", nil).
		AddUserMemory("u2", nil)

	snapshot := ad.MemorySnapshotWithLimit(2)
	if len(snapshot.Contexts) != 2 {
		t.Fatalf("expected snapshot len 2, got %d", len(snapshot.Contexts))
	}
	if snapshot.Contexts[0].Content != "t1" || snapshot.Contexts[1].Content != "u2" {
		t.Fatalf("expected latest contexts [t1 u2], got [%s %s]", snapshot.Contexts[0].Content, snapshot.Contexts[1].Content)
	}

	allSnapshot := ad.MemorySnapshotWithLimit(0)
	if len(allSnapshot.Contexts) != 4 {
		t.Fatalf("expected full snapshot len 4, got %d", len(allSnapshot.Contexts))
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

func TestAgentAndAgentDouble_BasicPromptHelpers(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)

	ad.Agent.SetCharacter("char").SetRole("role")
	ad.SetCharacter("double-char").SetRole("double-role")

	if ad.Agent.GetDescription() == "" {
		t.Fatalf("agent description should not be empty")
	}
	if ad.GetDescription() == "" {
		t.Fatalf("agent double description should not be empty")
	}

	ad.Agent.LearnSkill("s1", &mockSkill{})
	ad.LearnSkill("s2", &mockSkill{})
	if !strings.Contains(ad.Agent.toolPrompt(), "s1") {
		t.Fatalf("agent tool prompt should include learned skill")
	}
	if !strings.Contains(ad.toolPrompt(), "s2") {
		t.Fatalf("double tool prompt should include learned skill")
	}

	if ad.embeddingModelPrompt() == "" || ad.milvusPrompt() == "" || ad.loopPrompt() == "" {
		t.Fatalf("prompt helpers should not be empty")
	}
}

func TestAgentDouble_InitMemoryAndReset(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.Agent.LearnSkill("agent_skill", &mockSkill{})
	ad.LearnSkill("double_skill", &mockSkill{})
	ad.config.AgentMode = AgentModeLoop

	ad.InitMemory()
	if len(ad.memory.Contexts) == 0 {
		t.Fatalf("expected init memory entries")
	}

	ad.ResetMemory()
	if len(ad.memory.Contexts) == 0 {
		t.Fatalf("expected reset memory to reinitialize context")
	}
}

func TestAgentDouble_ListenAndWatch_AndThink(t *testing.T) {
	ad, ollamaCli, milvusCli, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{{0.1}}}
	milvusCli.searchResult = []string{"historical context"}
	ollamaCli.talkChunks = []string{"assistant answer"}

	err := ad.ListenAndWatch(context.Background(), "hello", nil, func(response string) error { return nil })
	if err != nil {
		t.Fatalf("listen and watch failed: %v", err)
	}

	foundContext := false
	for _, c := range ad.memory.Contexts {
		if c.Role == "system" && strings.Contains(c.Content, "Context:") {
			foundContext = true
			break
		}
	}
	if !foundContext {
		t.Fatalf("expected recalled context injected as system memory")
	}

	ollamaCli.talkChunks = []string{"thinking output"}
	if err := ad.Think(context.Background(), func(output any) error { return nil }); err != nil {
		t.Fatalf("think failed: %v", err)
	}

	ad.Learn("learned-info")
	if ad.memory.Contexts[len(ad.memory.Contexts)-1].Content != "learned-info" {
		t.Fatalf("expected learn to append assistant memory")
	}
}

func TestAgent_Close(t *testing.T) {
	milvusCli := &mockMilvusClient{}
	a := &Agent{milvusCli: milvusCli}
	if err := a.Close(); err != nil {
		t.Fatalf("close should not fail")
	}
}

func TestAgentDouble_Read_BadHTTPCode(t *testing.T) {
	ad, _, _, httpCli := newAgentDoubleWithMocks(t)
	httpCli.resp = &httpPKG.Response{StatusCode: 500, Body: []byte("x")}
	if err := ad.Read("https://example.com"); err == nil {
		t.Fatalf("expected bad http code error")
	}
}

var _ milvus.IClient = (*mockMilvusClient)(nil)

// --- mockCheckpoint ---

type mockCheckpoint struct {
	called bool
	err    error
}

func (m *mockCheckpoint) Do(_ *AgentDouble) error {
	m.called = true
	return m.err
}

// --- Additional coverage tests ---

func TestAgentOption_AddSkill(t *testing.T) {
	ms := &mockSkill{}
	a, err := NewAgent(context.Background(), func(opt *AgentOption) {
		opt.SetConfig(testConfig())
		opt.SetOllamaCli(&mockOllamaClient{})
		opt.SetMilvusCli(&mockMilvusClient{})
		opt.SetHttpCli(&mockHTTPClient{})
		opt.AddSkill("s1", ms)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := a.skillSet["s1"]; !ok {
		t.Fatalf("expected skill s1 in skillSet")
	}
}

func TestNewAgent_DefaultHTTPClient(t *testing.T) {
	_, err := NewAgent(context.Background(), func(opt *AgentOption) {
		opt.SetConfig(testConfig())
		opt.SetOllamaCli(&mockOllamaClient{})
		opt.SetMilvusCli(&mockMilvusClient{})
		// no httpCli provided — should use default
	})
	if err != nil {
		t.Fatalf("unexpected error with default http client: %v", err)
	}
}

func TestNewAgent_DefaultOllamaClient(t *testing.T) {
	_, err := NewAgent(context.Background(), func(opt *AgentOption) {
		opt.SetConfig(testConfig())
		opt.SetMilvusCli(&mockMilvusClient{})
		opt.SetHttpCli(&mockHTTPClient{})
		// no ollamaCli — should use default (no network call in constructor)
	})
	if err != nil {
		t.Fatalf("unexpected error with default ollama client: %v", err)
	}
}

func TestNewAgent_DefaultMilvusClientError(t *testing.T) {
	_, err := NewAgent(context.Background(), func(opt *AgentOption) {
		cfg := testConfig()
		cfg.MilvusHost = ""
		opt.SetConfig(cfg)
		opt.SetOllamaCli(&mockOllamaClient{})
		opt.SetHttpCli(&mockHTTPClient{})
	})
	if err == nil {
		t.Fatalf("expected error when default milvus init fails")
	}
}

func TestAgentDoubleOption_AddSkillAndSetCheckpoint(t *testing.T) {
	ollamaCli := &mockOllamaClient{}
	milvusCli := &mockMilvusClient{}
	agent, err := NewAgent(context.Background(), func(opt *AgentOption) {
		opt.SetConfig(testConfig())
		opt.SetOllamaCli(ollamaCli)
		opt.SetMilvusCli(milvusCli)
		opt.SetHttpCli(&mockHTTPClient{})
	})
	if err != nil {
		t.Fatalf("new agent failed: %v", err)
	}

	ms := &mockSkill{}
	cp := &mockCheckpoint{}
	ad, err := NewAgentDouble(context.Background(), func(opt *AgentDoubleOption) {
		opt.SetConfig(testConfig())
		opt.SetAgent(agent)
		opt.AddSkill("ds1", ms)
		opt.SetCheckpoint(cp)
	})
	if err != nil {
		t.Fatalf("new agent double failed: %v", err)
	}
	if _, ok := ad.skillSet["ds1"]; !ok {
		t.Fatalf("expected skill ds1 in skillSet")
	}
	if ad.checkpoint == nil {
		t.Fatalf("expected checkpoint to be set")
	}
}

func TestNewAgentDouble_NilAgentBuildFailure(t *testing.T) {
	_, err := NewAgentDouble(context.Background(), func(opt *AgentDoubleOption) {
		cfg := testConfig()
		cfg.MilvusHost = ""
		opt.SetConfig(cfg)
	})
	if err == nil {
		t.Fatalf("expected error when internal NewAgent build fails")
	}
}

func TestAgent_ToolPromptWithSkillError(t *testing.T) {
	a := &Agent{skillSet: map[string]skill.Skill{"bad": &mockSkill{descErr: errors.New("desc error")}}}
	if p := a.toolPrompt(); !strings.Contains(p, "failed to load description") {
		t.Fatalf("expected error message in tool prompt, got: %s", p)
	}
}

func TestAgentDouble_ToolPromptWithSkillError(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.skillSet["bad"] = &mockSkill{descErr: errors.New("desc error")}
	if p := ad.toolPrompt(); !strings.Contains(p, "failed to load description") {
		t.Fatalf("expected error message in double tool prompt, got: %s", p)
	}
}

func TestAgent_CommandSuccess(t *testing.T) {
	ms := &mockSkill{}
	a := &Agent{skillSet: map[string]skill.Skill{"s1": ms}}
	var got any
	err := a.Command(context.Background(), "s1", "ctx", func(output any) (any, error) {
		got = output
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.called {
		t.Fatalf("expected skill to be called")
	}
	if got != "mock-output" {
		t.Fatalf("expected mock-output, got: %v", got)
	}
}

func TestAgentDouble_CommandSuccess(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ms := &mockSkill{}
	ad.skillSet["s1"] = ms
	if err := ad.Command(context.Background(), "s1", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.called {
		t.Fatalf("expected skill called")
	}
}

func TestAgentDouble_CommandNotFound(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	if err := ad.Command(context.Background(), "missing", nil, nil); err == nil {
		t.Fatalf("expected command not found error")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_WithCheckpoint(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	cp := &mockCheckpoint{}
	ad.checkpoint = cp
	ollamaCli.talkChunks = []string{"response"}
	ad.AddUserMemory("trigger", nil)

	if err := ad.talkToOllamaWithMemory(context.Background(), func(r string) error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cp.called {
		t.Fatalf("expected checkpoint to be called")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_CheckpointError(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	expectedErr := errors.New("checkpoint failed")
	ad.checkpoint = &mockCheckpoint{err: expectedErr}
	ollamaCli.talkChunks = []string{"response"}
	ad.AddUserMemory("trigger", nil)

	err := ad.talkToOllamaWithMemory(context.Background(), func(r string) error { return nil })
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected checkpoint error, got: %v", err)
	}
}

func TestAgent_talkToOllama_ClientError(t *testing.T) {
	a := &Agent{config: testConfig(), ollamaCli: &mockOllamaClient{talkErr: errors.New("talk failed")}}
	_, err := a.talkToOllama("m", []*ollama.Message{{Role: "user", Content: "hi"}}, func(string) error { return nil })
	if err == nil {
		t.Fatalf("expected talk error")
	}
}

func TestAgent_talkToOllama_CallbackError(t *testing.T) {
	a := &Agent{config: testConfig(), ollamaCli: &mockOllamaClient{talkChunks: []string{"x"}}}
	expected := errors.New("callback failed")
	_, err := a.talkToOllama("m", []*ollama.Message{{Role: "user", Content: "hi"}}, func(string) error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected callback error, got: %v", err)
	}
}

func TestAgent_reviewResponse_BooleanOutcomes(t *testing.T) {
	a := &Agent{config: testConfig(), ollamaCli: &mockOllamaClient{talkChunks: []string{"true"}}}
	bad, err := a.reviewResponse("x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bad {
		t.Fatalf("expected true when model returns true")
	}

	a2 := &Agent{config: testConfig(), ollamaCli: &mockOllamaClient{talkChunks: []string{"false"}}}
	bad2, err := a2.reviewResponse("x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bad2 {
		t.Fatalf("expected false when model returns false")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_NonAbortSkillError(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ad.skillSet["err-skill"] = &mockSkill{err: errors.New("skill failed")}
	ollamaCli.talkChunks = []string{`<tool>{"function":"err-skill","context":{},"abort_on_error":false}</tool>`}
	ad.AddUserMemory("trigger", nil)

	var outputs []string
	err := ad.talkToOllamaWithMemory(context.Background(), func(response string) error {
		outputs = append(outputs, response)
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error with non-abort skill error, got: %v", err)
	}
	found := false
	for _, o := range outputs {
		if strings.Contains(o, "error") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected error message in output stream")
	}
}

func TestAgentDouble_talkToOllamaWithMemory_LoopRepeatBreak(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ad.config.AgentMode = AgentModeLoop
	ad.config.AgentLoopDuration = 0
	ollamaCli.talkChunks = []string{"same output"}
	ad.AddUserMemory("trigger", nil)

	if err := ad.talkToOllamaWithMemory(context.Background(), func(r string) error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentDouble_CompressContextByTokenBudget_ZeroBudget(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.config.ChatModelContextLimit = 0
	ad.AddUserMemory("test", nil)
	ad.compressContextByTokenBudget() // should early-return without panic
}

func TestAgentDouble_CompressContextByTokenBudget_DefaultReserveAndThreshold(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.config.ChatModelContextLimit = 5
	ad.config.ContextReserveTokens = 0   // triggers default
	ad.config.NearDuplicateThreshold = 0 // triggers default
	ad.AddAssistantMemory("msg1", nil)
	ad.AddAssistantMemory("msg2", nil)
	ad.compressContextByTokenBudget() // should not panic
}

func TestAgentDouble_ListenAndWatch_RecallError(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedErr = errors.New("embedding failed")
	if err := ad.ListenAndWatch(context.Background(), "query", nil, nil); err == nil {
		t.Fatalf("expected recall error when embedding fails")
	}
}

func TestAgentDouble_Read_HTTPError(t *testing.T) {
	ad, _, _, httpCli := newAgentDoubleWithMocks(t)
	httpCli.err = errors.New("connection refused")
	if err := ad.Read("https://example.com"); err == nil {
		t.Fatalf("expected error when http get fails")
	}
}

func TestAgentDouble_Read_EmptyBody(t *testing.T) {
	ad, _, _, httpCli := newAgentDoubleWithMocks(t)
	httpCli.resp = &httpPKG.Response{StatusCode: 200, Body: []byte{}}
	initialLen := len(ad.memory.Contexts)
	if err := ad.Read("https://example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ad.memory.Contexts) != initialLen {
		t.Fatalf("expected no memory added for empty body")
	}
}

func TestAgentDouble_Remember_EmptyEmbeddings(t *testing.T) {
	ad, ollamaCli, milvusCli, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{}}
	if err := ad.Remember(context.Background(), "info"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if milvusCli.insertCalled {
		t.Fatalf("expected no insert for empty embeddings")
	}
}

func TestAgentDouble_Recall_NoEmbeddings(t *testing.T) {
	ad, ollamaCli, _, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{}}
	result, err := ad.Recall(context.Background(), "q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty embeddings, got: %v", result)
	}
}

func TestAgentDouble_Recall_EmptySearchResults(t *testing.T) {
	ad, ollamaCli, milvusCli, _ := newAgentDoubleWithMocks(t)
	ollamaCli.embedResp = &ollama.EmbedResponse{Embeddings: [][]float32{{0.1}}}
	milvusCli.searchResult = []string{}
	result, err := ad.Recall(context.Background(), "q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil for empty search results, got: %v", result)
	}
}

func TestAgentDouble_ForgetMoreThanAvailable(t *testing.T) {
	ad, _, _, _ := newAgentDoubleWithMocks(t)
	ad.AddUserMemory("u1", nil)
	n := len(ad.memory.Contexts)
	ad.Forget(n + 100) // more than available
	if len(ad.memory.Contexts) != 0 {
		t.Fatalf("expected all memory removed, got: %d", len(ad.memory.Contexts))
	}
}

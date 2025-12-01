package ai_agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	httpPKG "github.com/luoxiaojun1992/ai-agent/pkg/http"
	"github.com/luoxiaojun1992/ai-agent/pkg/milvus"
	"github.com/luoxiaojun1992/ai-agent/pkg/ollama"
	"github.com/luoxiaojun1992/ai-agent/skill"
	"github.com/luoxiaojun1992/ai-agent/util/prompt"
)

type AgentMode string

const (
	AgentModeChat AgentMode = "chat"
	AgentModeLoop AgentMode = "loop"
)

type Config struct {
	ChatModel       string
	EmbeddingModel  string
	SupervisorModel string

	ModelTemperature float32

	SupervisorSwitch bool

	OllamaHost string

	MilvusHost       string
	MilvusCollection string

	HttpTimeout        time.Duration
	HttpAllowRedirects bool
	HttpMaxRedirects   int

	ChatModelContextLimit int

	AgentMode         AgentMode
	AgentLoopDuration time.Duration
}

type personalInfo struct {
	character string
	role      string
}

func (pi *personalInfo) characterPrompt() string {
	return "Personality: \n" + "You are " + pi.character
}

func (pi *personalInfo) rolePrompt() string {
	return "Role: \n" + "You are " + pi.role
}

func (pi *personalInfo) prompt() string {
	return pi.characterPrompt() + "\n" + pi.rolePrompt()
}

func (pi *personalInfo) setCharacter(character string) *personalInfo {
	pi.character = character
	return pi
}

func (pi *personalInfo) setRole(role string) *personalInfo {
	pi.role = role
	return pi
}

type AgentOption struct {
	config    *Config
	ollamaCli ollama.IClient
	milvusCli milvus.IClient
	httpCli   httpPKG.IClient
	character string
	role      string
	skillSet  map[string]skill.Skill
}

func (ao *AgentOption) SetConfig(config *Config) *AgentOption {
	ao.config = config
	return ao
}

func (ao *AgentOption) SetOllamaCli(ollamaCli ollama.IClient) *AgentOption {
	ao.ollamaCli = ollamaCli
	return ao
}

func (ao *AgentOption) SetMilvusCli(milvusCli milvus.IClient) *AgentOption {
	ao.milvusCli = milvusCli
	return ao
}

func (ao *AgentOption) SetHttpCli(httpCli httpPKG.IClient) *AgentOption {
	ao.httpCli = httpCli
	return ao
}

func (ao *AgentOption) SetCharacter(character string) *AgentOption {
	ao.character = character
	return ao
}

func (ao *AgentOption) SetRole(role string) *AgentOption {
	ao.role = role
	return ao
}

func (ao *AgentOption) AddSkill(name string, processor skill.Skill) *AgentOption {
	ao.skillSet[name] = processor
	return ao
}

type Agent struct {
	config *Config

	personalInfo *personalInfo
	skillSet     map[string]skill.Skill

	ollamaCli ollama.IClient
	milvusCli milvus.IClient
	httpCli   httpPKG.IClient
}

func NewAgent(ctx context.Context, optionFuncs ...func(option *AgentOption)) (*Agent, error) {
	option := &AgentOption{
		skillSet: make(map[string]skill.Skill),
	}
	for _, optionFunc := range optionFuncs {
		optionFunc(option)
	}
	if option.config == nil {
		return nil, errors.New("invalid agent config")
	}
	if option.ollamaCli == nil {
		option.SetOllamaCli(ollama.NewClient(&ollama.Config{
			Host: option.config.OllamaHost,
		}))
	}
	if option.milvusCli == nil {
		milvusCli, err := milvus.NewClient(ctx, &milvus.Config{
			Host: option.config.MilvusHost,
		})
		if err != nil {
			return nil, err
		}
		option.SetMilvusCli(milvusCli)
	}
	if option.httpCli == nil {
		httpCli := httpPKG.NewHTTPClient(option.config.HttpTimeout, option.config.HttpAllowRedirects, option.config.HttpMaxRedirects)
		option.SetHttpCli(httpCli)
	}

	return &Agent{
		config: option.config,
		personalInfo: &personalInfo{
			character: option.character,
			role:      option.role,
		},
		skillSet:  option.skillSet,
		ollamaCli: option.ollamaCli,
		milvusCli: option.milvusCli,
		httpCli:   option.httpCli,
	}, nil
}

func (a *Agent) SetCharacter(character string) *Agent {
	a.personalInfo.setCharacter(character)
	return a
}

func (a *Agent) SetRole(role string) *Agent {
	a.personalInfo.setRole(role)
	return a
}

func (a *Agent) GetDescription() string {
	return a.personalInfo.prompt()
}

func (a *Agent) toolPrompt() string {
	functionPromptList := make([]string, 0, len(a.skillSet))
	for skillName, skill := range a.skillSet {
		functionPromptList = append(functionPromptList, fmt.Sprintf("%s: %s", skillName, skill.GetDescription()))
	}
	allFunctionPrompt := strings.Join(functionPromptList, "\n\n")
	return fmt.Sprintf(`
When answering questions, if you need to call external tools or resources, please return the function call in JSON format embedded within your response. The JSON should strictly follow this structure:
<tool>{
  "function": "function_name",
  "context": {
    "parameter1": "value1",
    "parameter2": "value2"
  },
  "abort_on_error": true
}
</tool>
The value of context might be JSON object or other data structure strictly according to the payload definition of specific function.
For example, if you need to call a 'search' function to look up information about the weather in New York, you should include this JSON in your response:
<tool>
{
  "function": "search",
  "context": {
    "query": "weather in New York"
  },
  "abort_on_error": true
}
</tool>
Here is a list of supported functions (might also be called as skill or tool) you can call when needed:
%s
`, allFunctionPrompt)
}

func (a *Agent) LearnSkill(name string, processor skill.Skill) *Agent {
	a.skillSet[name] = processor
	return a
}

func (a *Agent) Command(ctx context.Context, skillName string, cmdCtx any, callback func(output any) (any, error)) error {
	processor, existed := a.skillSet[skillName]
	if !existed {
		return fmt.Errorf("skill [%s] hasn't been learned", skillName)
	}
	return processor.Do(ctx, cmdCtx, callback)
}

func (a *Agent) talkToOllama(model string, messages []*ollama.Message, callback func(response string) error) (string, error) {
	var responseContent strings.Builder
	var modelTemperature float32 = 0.1
	if a.config.ModelTemperature > 0.0 {
		modelTemperature = a.config.ModelTemperature
	}
	if err := a.ollamaCli.Talk(&ollama.ChatRequest{
		Model:    model,
		Messages: messages,
		Options: &ollama.ChatRequestOptions{
			Temperature: modelTemperature,
		},
	}, func(response string) error {
		if _, err := responseContent.WriteString(response); err != nil {
			return err
		}
		return callback(response)
	}); err != nil {
		return "", err
	}

	return responseContent.String(), nil
}

func (a *Agent) reviewResponse(response string) (bool, error) {
	checkResult, err := a.talkToOllama(a.config.SupervisorModel, []*ollama.Message{
		{
			Role: "system",
			Content: `Analyze the logical coherence of the following content.
If there are logical problems such as contradictions, unreasonable causal relationships, or incomplete reasoning, output 'true';
if there are no logical problems, output 'false'.
Content to be analyzed:` + "\n" + response,
		},
	}, func(_ string) error {
		return nil
	})
	if err != nil {
		return false, err
	}
	return checkResult == "true", nil
}

func (a *Agent) Close() error {
	return a.milvusCli.Close()
}

type MemoryCtx struct {
	Role    string
	Content string
	Images  []string
}

func (mc *MemoryCtx) toOllamaMessage() *ollama.Message {
	return &ollama.Message{
		Role:    mc.Role,
		Content: mc.Content,
		Images:  append(make([]string, 0, len(mc.Images)), mc.Images...),
	}
}

type Memory struct {
	Contexts []*MemoryCtx
}

func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) getContextLength() int {
	var length int
	for _, context := range m.Contexts {
		length += len(context.Content)
	}
	return length
}

type AgentDoubleOption struct {
	config     *Config
	agent      *Agent
	character  string
	role       string
	skillSet   map[string]skill.Skill
	checkpoint Checkpoint
}

func (ado *AgentDoubleOption) SetConfig(config *Config) *AgentDoubleOption {
	ado.config = config
	return ado
}

func (ado *AgentDoubleOption) SetAgent(agent *Agent) *AgentDoubleOption {
	ado.agent = agent
	return ado
}

func (ado *AgentDoubleOption) SetCharacter(character string) *AgentDoubleOption {
	ado.character = character
	return ado
}

func (ado *AgentDoubleOption) SetRole(role string) *AgentDoubleOption {
	ado.role = role
	return ado
}

func (ado *AgentDoubleOption) AddSkill(name string, processor skill.Skill) *AgentDoubleOption {
	ado.skillSet[name] = processor
	return ado
}

func (ado *AgentDoubleOption) SetCheckpoint(checkpoint Checkpoint) *AgentDoubleOption {
	ado.checkpoint = checkpoint
	return ado
}

type AgentDouble struct {
	config *Config

	Agent        *Agent
	personalInfo *personalInfo
	skillSet     map[string]skill.Skill
	memory       *Memory
	checkpoint   Checkpoint
}

func NewAgentDouble(ctx context.Context, optionFuncs ...func(option *AgentDoubleOption)) (*AgentDouble, error) {
	doubleOption := &AgentDoubleOption{
		skillSet: make(map[string]skill.Skill),
	}
	for _, optionFunc := range optionFuncs {
		optionFunc(doubleOption)
	}
	if doubleOption.config == nil {
		return nil, errors.New("invalid agent double config")
	}
	if doubleOption.agent == nil {
		agent, err := NewAgent(ctx, func(option *AgentOption) {
			option.SetConfig(doubleOption.config)
		})
		if err != nil {
			return nil, err
		}
		doubleOption.SetAgent(agent)
	}

	return &AgentDouble{
		config: doubleOption.config,
		Agent:  doubleOption.agent,
		personalInfo: &personalInfo{
			character: doubleOption.character,
			role:      doubleOption.role,
		},
		skillSet:   doubleOption.skillSet,
		memory:     NewMemory(),
		checkpoint: doubleOption.checkpoint,
	}, nil
}

func (ad *AgentDouble) embeddingModelPrompt() string {
	return fmt.Sprintf("The embedding model currently in use by the agent is: [%s].", ad.config.EmbeddingModel)
}

func (ad *AgentDouble) milvusPrompt() string {
	return fmt.Sprintf("The Milvus collection currently in use by the agent is: [%s].", ad.config.MilvusCollection)
}

func (ad *AgentDouble) loopPrompt() string {
	return "Determine if the conversation should continue strictly. If not, include strictly <loop_end/> in your response. If you find too may duplicate content, please exit immediately."
}

func (ad *AgentDouble) toolPrompt() string {
	functionPromptList := make([]string, 0, len(ad.skillSet))
	for skillName, skill := range ad.skillSet {
		functionPromptList = append(functionPromptList, fmt.Sprintf("%s: %s", skillName, skill.GetDescription()))
	}
	allFunctionPrompt := strings.Join(functionPromptList, "\n\n")
	return fmt.Sprintf(`
Here is a list of supported high priority functions (might also be called as skill or tool) you can call when needed:
%s
`, allFunctionPrompt)
}

func (ad *AgentDouble) SetCharacter(character string) *AgentDouble {
	ad.personalInfo.setCharacter(character)
	return ad
}

func (ad *AgentDouble) SetRole(role string) *AgentDouble {
	ad.personalInfo.setRole(role)
	return ad
}

func (ad *AgentDouble) GetDescription() string {
	return ad.Agent.GetDescription() + "\n" + ad.personalInfo.prompt()
}

func (ad *AgentDouble) LearnSkill(name string, processor skill.Skill) *AgentDouble {
	ad.skillSet[name] = processor
	return ad
}

func (ad *AgentDouble) Command(ctx context.Context, skillName string, cmdCtx any, callback func(output any) (any, error)) error {
	processor, existed := ad.skillSet[skillName]
	if !existed {
		return fmt.Errorf("high level skill [%s] hasn't been learned", skillName)
	}
	return processor.Do(ctx, cmdCtx, callback)
}

func (ad *AgentDouble) AddMemory(role, content string, images []string) *AgentDouble {
	ad.memory.Contexts = append(ad.memory.Contexts, &MemoryCtx{
		Role:    role,
		Content: content,
		Images:  images,
	})
	return ad
}

func (ad *AgentDouble) AddSystemMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("system", content, images)
}

func (ad *AgentDouble) AddAssistantMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("assistant", content, images)
}

func (ad *AgentDouble) AddUserMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("user", content, images)
}

func (ad *AgentDouble) AddToolMemory(content string, images []string) *AgentDouble {
	return ad.AddMemory("tool", content, images)
}

func (ad *AgentDouble) InitMemory() *AgentDouble {
	ado := ad.AddAssistantMemory(ad.Agent.personalInfo.prompt(), nil).
		AddAssistantMemory(ad.personalInfo.prompt(), nil).
		AddSystemMemory(ad.embeddingModelPrompt(), nil).
		AddSystemMemory(ad.milvusPrompt(), nil)
	if len(ad.Agent.skillSet) > 0 {
		ado.AddSystemMemory(ad.Agent.toolPrompt(), nil)
	}
	if len(ad.skillSet) > 0 {
		ado.AddSystemMemory(ad.toolPrompt(), nil)
	}
	if ad.config.AgentMode == AgentModeLoop {
		ado.AddSystemMemory(ad.loopPrompt(), nil)
	}
	return ado
}

func (ad *AgentDouble) talkToOllamaWithMemory(ctx context.Context, callback func(response string) error) error {
	var previousResponseCOntent string

	for {
		ollamaMessages := make([]*ollama.Message, 0, len(ad.memory.Contexts))
		for _, memCtx := range ad.memory.Contexts {
			ollamaMessages = append(ollamaMessages, memCtx.toOllamaMessage())
		}

		//todo select chat model

		responseContentStr, err := ad.Agent.talkToOllama(ad.config.ChatModel, ollamaMessages, callback)
		if err != nil {
			return err
		}

		if len(responseContentStr) <= 0 {
			return nil
		}

		if responseContentStr == previousResponseCOntent {
			return nil
		}

		if ad.config.SupervisorSwitch {
			isCompliant, err := ad.Agent.reviewResponse(responseContentStr)
			if err != nil {
				return err
			}
			if !isCompliant {
				return errors.New("response from model is non-compliant")
			}
		}

		previousResponseCOntent = responseContentStr
		ad.AddAssistantMemory(responseContentStr, nil)

		functionCallList, err := prompt.ParseFunctionCalling(responseContentStr)
		if err != nil {
			return err
		}
		for _, functionCall := range functionCallList {
			funcCallback := func(output any) (any, error) {
				resultOfFunCall := fmt.Sprintf("The result of function [%s]: %v", functionCall.Function, output)
				ad.AddToolMemory(resultOfFunCall, nil)
				err := callback(resultOfFunCall)
				return nil, err
			}
			var cmdErr error
			if _, existedHighCmd := ad.skillSet[functionCall.Function]; existedHighCmd {
				cmdErr = ad.Command(ctx, functionCall.Function, functionCall.Context, funcCallback)
			} else if _, existedCmd := ad.Agent.skillSet[functionCall.Function]; existedCmd {
				cmdErr = ad.Agent.Command(ctx, functionCall.Function, functionCall.Context, funcCallback)
			}
			if cmdErr != nil {
				errorOfFuncCall := fmt.Sprintf("The error [%s] happened during executing the function [%s].",
					cmdErr.Error(),
					functionCall.Function)
				ad.AddToolMemory(errorOfFuncCall, nil)
				if err := callback(errorOfFuncCall); err != nil {
					return err
				}
				if functionCall.AbortOnError {
					break
				}
				continue
			}

			successOfFuncCall := fmt.Sprintf("The function [%s] has been executed successfully.", functionCall.Function)
			ad.AddToolMemory(successOfFuncCall, nil)
			callback(successOfFuncCall)
		}

		if ad.checkpoint != nil {
			if err := ad.checkpoint.Do(ad); err != nil {
				return err
			}
		}

		//Forget memory due to the length limit of context
		if ad.memory.getContextLength() > ad.config.ChatModelContextLimit {
			ad.Forget(-1)
		}

		if ad.config.AgentMode != AgentModeLoop || prompt.ParseLoopEnd(responseContentStr) {
			break
		}

		time.Sleep(ad.config.AgentLoopDuration)
	}

	return nil
}

func (ad *AgentDouble) ListenAndWatch(ctx context.Context, message string, images []string, callback func(response string) error) error {
	//Search context
	ctxVectors, err := ad.Recall(ctx, message)
	if err != nil {
		return err
	}
	if len(ctxVectors) > 0 {
		ad.AddSystemMemory("Context: \n"+strings.Join(ctxVectors, "\n"), nil)
	}

	//Generate response
	ad.AddUserMemory(message, images)
	return ad.talkToOllamaWithMemory(ctx, callback)
}

func (ad *AgentDouble) Think(ctx context.Context, callback func(output any) error) error {
	ad.AddAssistantMemory("Let me think and output something", nil)
	return ad.talkToOllamaWithMemory(ctx, func(response string) error {
		return callback(response)
	})
}

func (ad *AgentDouble) Learn(info string) *AgentDouble {
	return ad.AddAssistantMemory(info, nil)
}

func (ad *AgentDouble) Read(url string) error {
	resp, err := ad.Agent.httpCli.Get(url, nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("bad http code while reading in agent double")
	}
	if len(resp.Body) > 0 {
		ad.AddAssistantMemory(string(resp.Body), nil)
	}
	return nil
}

func (ad *AgentDouble) Remember(ctx context.Context, info string) error {
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: info,
	})
	if err != nil {
		return err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		return ad.Agent.milvusCli.InsertVector(ctx, ad.config.MilvusCollection, info, embeddingResponse.Embeddings[0])
	}
	return nil
}

func (ad *AgentDouble) Recall(ctx context.Context, prompt string) ([]string, error) {
	embeddingResponse, err := ad.Agent.ollamaCli.EmbeddingPrompt(&ollama.EmbedRequest{
		Model: ad.config.EmbeddingModel,
		Input: prompt,
	})
	if err != nil {
		return nil, err
	}
	if len(embeddingResponse.Embeddings) > 0 && len(embeddingResponse.Embeddings[0]) > 0 {
		ctxVectors, err := ad.Agent.milvusCli.SearchVector(ctx, ad.config.MilvusCollection, embeddingResponse.Embeddings[0])
		if err != nil {
			return nil, err
		}
		if len(ctxVectors) > 0 {
			return ctxVectors, nil
		}
	}
	return nil, nil
}

func (ad *AgentDouble) Forget(number int) *AgentDouble {
	memoryLen := len(ad.memory.Contexts)
	if number < 0 {
		number = memoryLen
	}
	if number > memoryLen {
		number = memoryLen
	}

	leftMemoryLen := memoryLen - number

	if leftMemoryLen <= 0 {
		ad.memory.Contexts = nil
		return ad
	}

	ad.memory.Contexts = ad.memory.Contexts[:leftMemoryLen]
	return ad
}

func (ad *AgentDouble) ResetMemory() *AgentDouble {
	return ad.Forget(-1).InitMemory()
}

func (ad *AgentDouble) MemorySnapshot() *Memory {
	memorySnapshot := &Memory{}
	for _, memoryCtx := range ad.memory.Contexts {
		memorySnapshot.Contexts = append(memorySnapshot.Contexts, &MemoryCtx{
			Role:    memoryCtx.Role,
			Content: memoryCtx.Content,
			Images:  append(make([]string, 0, len(memoryCtx.Images)), memoryCtx.Images...),
		})
	}
	return memorySnapshot
}

func (ad *AgentDouble) LoadMemory(snapshot *Memory) *AgentDouble {
	newMemory := &Memory{}
	for _, memoryCtx := range snapshot.Contexts {
		newMemory.Contexts = append(newMemory.Contexts, &MemoryCtx{
			Role:    memoryCtx.Role,
			Content: memoryCtx.Content,
			Images:  append(make([]string, 0, len(memoryCtx.Images)), memoryCtx.Images...),
		})
	}
	ad.memory = newMemory
	return ad
}

package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	TotalDuration   int64       `json:"total_duration"`
	LoadDuration    int64       `json:"load_duration"`
	PromptEvalCount int64       `json:"prompt_eval_count"`
}

type ChatRequest struct {
	Model    string              `json:"model"`
	Messages []*Message          `json:"messages"`
	Options  *ChatRequestOptions `json:"options"`
}

type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

type ChatRequestOptions struct {
	Temperature float32 `json:"temperature"`
}

type StreamResponse struct {
	Model         string   `json:"model"`
	CreatedAt     string   `json:"created_at"`
	Message       *Message `json:"message"`
	Done          bool     `json:"done"`
	TotalDuration int64    `json:"total_duration"`
}

type IClient interface {
	EmbeddingPrompt(embedReq *EmbedRequest) (*EmbedResponse, error)
	Talk(chatReq *ChatRequest, callback func(response string) error) error
}

type Config struct {
	Host    string
	APIType string
	APIKey  string
}

type Client struct {
	config   *Config
	strategy apiStrategy
}

func NewClient(config *Config) *Client {
	if config == nil {
		config = &Config{}
	}
	var strategy apiStrategy = newOllamaAPIStrategy()
	if strings.EqualFold(strings.TrimSpace(config.APIType), "openai") {
		strategy = newOpenAICompatibleStrategy()
	}

	return &Client{
		config:   config,
		strategy: strategy,
	}
}

func (c *Client) EmbeddingPrompt(embedReq *EmbedRequest) (*EmbedResponse, error) {
	return c.strategy.EmbeddingPrompt(c.config, embedReq)
}

func (c *Client) Talk(chatReq *ChatRequest, callback func(response string) error) error {
	return c.strategy.Talk(c.config, chatReq, callback)
}

type apiStrategy interface {
	EmbeddingPrompt(config *Config, embedReq *EmbedRequest) (*EmbedResponse, error)
	Talk(config *Config, chatReq *ChatRequest, callback func(response string) error) error
}

type ollamaAPIStrategy struct{}

func newOllamaAPIStrategy() *ollamaAPIStrategy {
	return &ollamaAPIStrategy{}
}

func (s *ollamaAPIStrategy) EmbeddingPrompt(config *Config, embedReq *EmbedRequest) (*EmbedResponse, error) {
	jsonReq, _ := json.Marshal(embedReq)

	req, err := http.NewRequest("POST", strings.TrimRight(config.Host, "/")+"/api/embed", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setAuthHeaderIfNeeded(req, config)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error embedding prompt, status code %d", resp.StatusCode)
	}

	embedResponseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	embedResponse := &EmbedResponse{}
	if err := json.Unmarshal(embedResponseBytes, embedResponse); err != nil {
		return nil, err
	}

	return embedResponse, nil
}

func (s *ollamaAPIStrategy) Talk(config *Config, chatReq *ChatRequest, callback func(response string) error) error {
	jsonReq, _ := json.Marshal(chatReq)

	req, err := http.NewRequest("POST", strings.TrimRight(config.Host, "/")+"/api/chat", bytes.NewBuffer(jsonReq))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setAuthHeaderIfNeeded(req, config)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error talking to ollama, status code %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			continue
		}
		if streamResp.Message == nil {
			if streamResp.Done {
				break
			}
			continue
		}

		content := streamResp.Message.Content
		if err := callback(content); err != nil {
			return err
		}

		if streamResp.Done {
			break
		}
	}

	return scanner.Err()
}

type openAICompatibleStrategy struct{}

type openAIChatRequest struct {
	Model       string     `json:"model"`
	Messages    []*Message `json:"messages"`
	Temperature float32    `json:"temperature,omitempty"`
	Stream      bool       `json:"stream"`
}

type openAIChatStreamDelta struct {
	Content string `json:"content"`
}

type openAIChatStreamChoice struct {
	Delta        *openAIChatStreamDelta `json:"delta"`
	FinishReason string                 `json:"finish_reason"`
}

type openAIChatStreamResponse struct {
	Choices []*openAIChatStreamChoice `json:"choices"`
}

type openAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
}

type openAIEmbeddingResponse struct {
	Model string                 `json:"model"`
	Data  []*openAIEmbeddingData `json:"data"`
}

func newOpenAICompatibleStrategy() *openAICompatibleStrategy {
	return &openAICompatibleStrategy{}
}

func (s *openAICompatibleStrategy) EmbeddingPrompt(config *Config, embedReq *EmbedRequest) (*EmbedResponse, error) {
	reqBody := &openAIEmbeddingRequest{
		Model: embedReq.Model,
		Input: embedReq.Input,
	}
	jsonReq, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", strings.TrimRight(config.Host, "/")+"/v1/embeddings", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	setAuthHeaderIfNeeded(req, config)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error embedding prompt, status code %d", resp.StatusCode)
	}

	embedResponseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var openAIEmbedResp openAIEmbeddingResponse
	if err := json.Unmarshal(embedResponseBytes, &openAIEmbedResp); err != nil {
		return nil, err
	}

	result := &EmbedResponse{
		Model:      openAIEmbedResp.Model,
		Embeddings: make([][]float32, 0, len(openAIEmbedResp.Data)),
	}
	for _, item := range openAIEmbedResp.Data {
		if item == nil {
			continue
		}
		result.Embeddings = append(result.Embeddings, item.Embedding)
	}
	return result, nil
}

func (s *openAICompatibleStrategy) Talk(config *Config, chatReq *ChatRequest, callback func(response string) error) error {
	reqBody := &openAIChatRequest{
		Model:    chatReq.Model,
		Messages: chatReq.Messages,
		Stream:   true,
	}
	if chatReq.Options != nil {
		reqBody.Temperature = chatReq.Options.Temperature
	}
	jsonReq, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", strings.TrimRight(config.Host, "/")+"/v1/chat/completions", bytes.NewBuffer(jsonReq))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	setAuthHeaderIfNeeded(req, config)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error talking to ollama openai compatible endpoint, status code %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "[DONE]" {
			break
		}

		var streamResp openAIChatStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			log.Printf("openai compatible stream unmarshal failed: %v", err)
			continue
		}
		for _, choice := range streamResp.Choices {
			if choice == nil {
				continue
			}
			if choice.Delta != nil && choice.Delta.Content != "" {
				if err := callback(choice.Delta.Content); err != nil {
					return err
				}
			}
			if choice.FinishReason != "" {
				return nil
			}
		}
	}

	return scanner.Err()
}

func setAuthHeaderIfNeeded(req *http.Request, config *Config) {
	if config == nil {
		return
	}
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

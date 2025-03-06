package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	Model    string     `json:"model"`
	Messages []*Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	Host string
}

type Client struct {
	config *Config
}

func NewClient(config *Config) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) EmbeddingPrompt(embedReq *EmbedRequest) (*EmbedResponse, error) {
	jsonReq, _ := json.Marshal(embedReq)

	req, err := http.NewRequest("POST", c.config.Host+"/api/embed", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("error embedding prompt, status code %d", resp.StatusCode))
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

func (c *Client) Talk(chatReq *ChatRequest, callback func(response string) error) error {
	jsonReq, _ := json.Marshal(chatReq)

	req, err := http.NewRequest("POST", c.config.Host+"/api/chat", bytes.NewBuffer(jsonReq))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("error talking to ollama, status code %d", resp.StatusCode))
	}

	scanner := bufio.NewScanner(resp.Body)

	if err := scanner.Err(); err != nil {
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
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

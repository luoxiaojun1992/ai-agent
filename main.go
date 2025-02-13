package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	milvusClient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

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

const (
	OllamaHost = "http://localhost:11434"
	MilvusHost = "localhost:19530"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <milvus_collection> <model_name> <ollama_host> <milvus_host>")
		fmt.Println("Default ollama host: http://localhost:11434")
		os.Exit(0)
	}

	milvusCollection := os.Args[1]
	modelName := os.Args[2]

	ollamaHost := OllamaHost
	if len(os.Args) > 3 {
		ollamaHost = os.Args[3]
	}
	ollamaChatEndpoint := ollamaHost + "/api/chat"

	milvusHost := MilvusHost
	if len(os.Args) > 4 {
		milvusHost = os.Args[4]
	}

	//Connect milvus
	milvusCli, err := milvusClient.NewClient(context.Background(), milvusClient.Config{
		Address: milvusHost,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Milvus: %v", err)
	}

	var history []*Message

	fmt.Println("AI Agent stared")
	fmt.Println("Please input 'exit' to stop the agent.")
	fmt.Println("Please input 'clear' to delete all contexts.")

	for {
		fmt.Println("Prompt: ")

		var prompt string
		if _, err := fmt.Scanln(&prompt); err != nil {
			if err == io.EOF {
				fmt.Println("Exited.")
				os.Exit(0)
			} else {
				fmt.Println("Error reading input:", err)
				continue
			}
		}

		if prompt == "exit" {
			fmt.Println("Exited.")
			os.Exit(0)
		}

		if prompt == "clear" {
			history = nil
			continue
		}

		// Search contexts
		contextStr, err := searchContext(milvusCli, milvusCollection, nil) //todo
		if err != nil {
			fmt.Println("Error fetching contexts:", err)
			continue
		}
		if len(contextStr) > 0 {
			history = append(history, &Message{
				Role:    "system",
				Content: "Context: \n" + contextStr,
			})
		}

		fmt.Println("Generating...")

		msg := &Message{
			Role:    "user",
			Content: prompt,
		}
		history = append(history, msg)

		var responseContent strings.Builder

		if err := talkToOllama(ollamaChatEndpoint, &ChatRequest{
			Model:    modelName,
			Messages: history,
		}, func(content string) error {
			if _, err := responseContent.WriteString(content); err != nil {
				return err
			}
			fmt.Print(content)
			return nil
		}); err != nil {
			fmt.Println("Error talking to ollama:", err)
			continue
		}

		history = append(history, &Message{
			Role:    "assistant",
			Content: responseContent.String(),
		})

		fmt.Println("")
	}
}

func embeddingPrompt(endpoint string, ollamaReq *ChatRequest, callback func(vector []float32) error) error {
	//todo
	return nil
}

func talkToOllama(endpoint string, ollamaReq *ChatRequest, callback func(content string) error) error {
	jsonReq, _ := json.Marshal(ollamaReq)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil
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

func searchVector(milvusCli milvusClient.Client, collectionName string, vector []float32) ([]string, error) {
	var contents []string

	sp, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, err
	}
	resList, err := milvusCli.Search(
		context.Background(),
		collectionName,
		[]string{},
		"",
		[]string{"content"},
		[]entity.Vector{entity.FloatVector(vector)},
		"content_embedding",
		entity.L2,
		3,
		sp,
	)
	if err != nil {
		return nil, err
	}
	for _, res := range resList {
		contentColumn := res.Fields.GetColumn("content")
		for i := 0; i < res.ResultCount; i++ {
			content, err := contentColumn.GetAsString(i)
			if err != nil {
				return nil, err
			}
			contents = append(contents, content)
		}
	}

	return contents, nil
}

func searchContext(milvusCli milvusClient.Client, collectionName string, vector []float32) (string, error) {
	contextList, err := searchVector(milvusCli, collectionName, vector)
	if err != nil {
		return "", err
	}
	if len(contextList) > 0 {
		return strings.Join(contextList, "\n"), nil
	}
	return "", nil
}

// Deprecated
func readFiles(dir string) (map[string][]string, error) {
	var filesContent = make(map[string][]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			filesContent[path] = strings.Split(string(content), "\n")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesContent, nil
}

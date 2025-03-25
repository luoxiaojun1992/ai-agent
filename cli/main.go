package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	ai_agent "github.com/luoxiaojun1992/ai-agent"
	"github.com/luoxiaojun1992/ai-agent/skill/impl"
)

const (
	OllamaHost = "http://localhost:11434"
	MilvusHost = "localhost:19530"
)

func main() {
	ctx := context.Background()

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <milvus_collection> <embed_model_name> <chat_model_name> [<ollama_host> <milvus_host>]")
		fmt.Println("Default ollama host: http://localhost:11434")
		os.Exit(0)
	}

	milvusCollection := os.Args[1]
	chatModelName := os.Args[2]
	embedModelName := os.Args[3]

	ollamaHost := OllamaHost
	if len(os.Args) > 4 {
		ollamaHost = os.Args[4]
	}

	milvusHost := MilvusHost
	if len(os.Args) > 5 {
		milvusHost = os.Args[5]
	}

	agentDouble, err := ai_agent.NewAgentDouble(context.Background(), &ai_agent.Config{
		ChatModel:        chatModelName,
		EmbeddingModel:   embedModelName,
		OllamaHost:       ollamaHost,
		MilvusHost:       milvusHost,
		MilvusCollection: milvusCollection,
	}, nil)
	if err != nil {
		log.Fatalf("Failed to create AI agent: %v", err)
	}
	defer func() {
		if err := agentDouble.Agent.Close(); err != nil {
			log.Fatalf("Failed to close AI agent: %v", err)
		}
	}()
	agentDouble.Agent.SetCharacter("Kind")
	agentDouble.Agent.SetRole("General AI")
	agentDouble.Agent.LearnSkill("team", &impl.Team{})
	agentDouble.InitMemory()

	fmt.Println("AI agent stared")
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
			agentDouble.ResetMemory()
			continue
		}

		fmt.Println("Generating...")

		if err := agentDouble.ListenAndWatch(ctx, prompt, nil, func(response string) error {
			_, err := fmt.Print(response)
			return err
		}); err != nil {
			fmt.Println("\nError talking to AI agent:", err)
			continue
		}

		fmt.Println("")
	}
}

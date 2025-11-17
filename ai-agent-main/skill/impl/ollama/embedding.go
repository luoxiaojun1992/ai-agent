package ollama

import (
	"context"
	"errors"
	"fmt"

	ollamaPKG "github.com/luoxiaojun1992/ai-agent/ai-agent-main/pkg/ollama"
)

type Embedding struct {
	OllamaCli ollamaPKG.IClient
}

func (e *Embedding) GetDescription() string {
	return `Generate vector embeddings using Ollama embedding models. This skill converts text content into numerical vector representations suitable for semantic search and similarity comparisons.
Parameters:
- model: string - The name of the Ollama embedding model to use (e.g., "nomic-embed-text")
- content: string - The text content to generate embeddings for
Returns: Vector embedding array
Note: Requires Ollama service with embedding models installed`
}

func (e *Embedding) ShortDescription() string {
	return "Generate text embeddings with Ollama"
}

func (e *Embedding) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for ollama/embedding skill")
	}

	model, hasModel := params["model"]
	if !hasModel {
		return errors.New("not found model from params")
	}
	modelStr, isValidModel := model.(string)
	if !isValidModel {
		return errors.New("error converting model from params")
	}

	content, hasContent := params["content"]
	if !hasContent {
		return errors.New("not found content from params")
	}
	contentStr, isValidContent := content.(string)
	if !isValidContent {
		return errors.New("error converting content from params")
	}

	embeddingResponse, err := e.OllamaCli.EmbeddingPrompt(&ollamaPKG.EmbedRequest{
		Model: modelStr,
		Input: contentStr,
	})
	if err != nil {
		return err
	}
	
	// Extract embeddings from response
	if embeddingResponse == nil {
		return errors.New("empty embedding response")
	}
	
	_, err = callback(embeddingResponse)
	return err
}

package ollama

import (
	"context"
	"errors"

	ollamaPKG "github.com/luoxiaojun1992/ai-agent/pkg/ollama"
)

type Embedding struct {
	OllamaCli ollamaPKG.IClient
}

func (e *Embedding) GetDescription() string {
	//todo model
	return ""
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
	_, err = callback(embeddingResponse)
	return err
}

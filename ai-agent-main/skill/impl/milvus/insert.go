package milvus

import (
	"context"
	"errors"
	"fmt"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/pkg/milvus"
)

type Insert struct {
	MilvusCli milvusPKG.IClient
}

func (i *Insert) GetDescription() string {
	return `Insert vector embeddings and associated content into Milvus vector database. This skill stores text content along with its vector representation for semantic search capabilities.
Parameters:
- collection: string - The name of the Milvus collection to insert into
- content: string - The text content to be stored
- vector: []float32 - The vector embedding of the content
Returns: Success status
Note: Requires pre-configured Milvus connection and existing collection`
}

func (i *Insert) ShortDescription() string {
	return "Insert vectors into Milvus database"
}

func (i *Insert) Do(ctx context.Context, cmdCtx any, _ func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for milvus/insert skill")
	}

	collection, hasCollection := params["collection"]
	if !hasCollection {
		return errors.New("not found collection from params")
	}
	collectionStr, isValidCollection := collection.(string)
	if !isValidCollection {
		return errors.New("error converting collection from params")
	}

	content, hasContent := params["content"]
	if !hasContent {
		return errors.New("not found content from params")
	}
	contentStr, isValidContent := content.(string)
	if !isValidContent {
		return errors.New("error converting content from params")
	}

	vector, hasVector := params["vector"]
	if !hasVector {
		return errors.New("not found vector from params")
	}
	
	// Handle different vector types
	var vectorSlice []float32
	switch v := vector.(type) {
	case []float32:
		vectorSlice = v
	case []interface{}:
		vectorSlice = make([]float32, len(v))
		for i, val := range v {
			if floatVal, ok := val.(float64); ok {
				vectorSlice[i] = float32(floatVal)
			} else if float32Val, ok := val.(float32); ok {
				vectorSlice[i] = float32Val
			} else {
				return fmt.Errorf("error converting vector element at index %d: expected float32 or float64, got %T", i, val)
			}
		}
	default:
		return fmt.Errorf("error converting vector from params: expected []float32 or []interface{}, got %T", vector)
	}

	return i.MilvusCli.InsertVector(ctx, collectionStr, contentStr, vectorSlice)
}

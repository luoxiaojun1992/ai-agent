package milvus

import (
	"context"
	"errors"
	"fmt"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/ai-agent-main/pkg/milvus"
)

type Search struct {
	MilvusCli milvusPKG.IClient
}

func (s *Search) GetDescription() string {
	return `Search for similar vectors in Milvus vector database. This skill performs semantic search by finding vectors closest to the query vector using cosine similarity.
Parameters:
- collection: string - The name of the Milvus collection to search in
- vector: []float32 - The query vector to search for similar items
Returns: Array of similar content strings from the database
Note: Requires pre-configured Milvus connection and existing collection with indexed vectors`
}

func (s *Search) ShortDescription() string {
	return "Search vectors in Milvus database"
}

func (s *Search) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for milvus/search skill")
	}

	collection, hasCollection := params["collection"]
	if !hasCollection {
		return errors.New("not found collection from params")
	}
	collectionStr, isValidCollection := collection.(string)
	if !isValidCollection {
		return errors.New("error converting collection from params")
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

	ctxVectors, err := s.MilvusCli.SearchVector(ctx, collectionStr, vectorSlice)
	if err != nil {
		return err
	}
	_, err = callback(ctxVectors)
	return err
}

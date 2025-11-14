package milvus

import (
	"context"
	"errors"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/pkg/milvus"
)

type Search struct {
	MilvusCli milvusPKG.IClient
}

func (s *Search) GetDescription() string {
	//todo CollectionName
	return ""
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
	vectorSlice, isValidVector := vector.([]float32)
	if !isValidVector {
		return errors.New("error converting vector from params")
	}

	ctxVectors, err := s.MilvusCli.SearchVector(ctx, collectionStr, vectorSlice)
	if err != nil {
		return err
	}
	_, err = callback(ctxVectors)
	return err
}

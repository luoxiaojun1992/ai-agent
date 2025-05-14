package milvus

import (
	"context"
	"errors"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/pkg/milvus"
)

type Insert struct {
	MilvusCli milvusPKG.IClient
}

func (i *Insert) GetDescription() string {
	//todo CollectionName
	return ""
}

func (i *Insert) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for filesystem/file/reader skill")
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
	vectorSlice, isValidVector := vector.([]float32)
	if !isValidVector {
		return errors.New("error converting vector from params")
	}

	return i.MilvusCli.InsertVector(ctx, collectionStr, contentStr, vectorSlice)
}

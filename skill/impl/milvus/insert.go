package milvus

import (
	"context"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/pkg/milvus"
)

type Insert struct {
	MilvusCli      milvusPKG.IClient
	CollectionName string
}

func (i *Insert) GetDescription() string {
	//todo
	return ""
}

func (i *Insert) Do(_ context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo
	return nil
}

package milvus

import (
	"context"

	milvusPKG "github.com/luoxiaojun1992/ai-agent/pkg/milvus"
)

type Search struct {
	MilvusCli      milvusPKG.IClient
	CollectionName string
}

func (s *Search) GetDescription() string {
	//todo
	return ""
}

func (s *Search) Do(_ context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo
	return nil
}

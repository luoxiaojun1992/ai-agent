package milvus

import (
	"context"
	milvusClient "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type IClient interface {
	SearchVector(collectionName string, vector []float32) ([]string, error)
}

type Config struct {
	Host string
}

type Client struct {
	config *Config

	milvusCli milvusClient.Client
}

func NewClient(ctx context.Context, config *Config) (*Client, error) {
	milvusCli, err := milvusClient.NewClient(ctx, milvusClient.Config{
		Address: config.Host,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		config:    config,
		milvusCli: milvusCli,
	}, nil
}

func (c *Client) SearchVector(collectionName string, vector []float32) ([]string, error) {
	var contents []string

	sp, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, err
	}
	resList, err := c.milvusCli.Search(
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

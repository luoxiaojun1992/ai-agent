package milvus

import milvusClient "github.com/milvus-io/milvus-sdk-go/v2/client"

type IClient interface {
	searchVector(collectionName string, vector []float32) ([]string, error)
}

type Client struct {
	//todo
	milvusCli milvusClient.Client
}

//todo

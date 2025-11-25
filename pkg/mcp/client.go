package mcp

import (
	"context"
	"encoding/json"
	"errors"

	mcpClient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type IClient interface {
}

type Config struct {
	Host string
}

type Client struct {
	config       *Config
	sseMCPClient *mcpClient.SSEMCPClient
}

func NewClient(config *Config) (*Client, error) {
	sseMCPClient, err := mcpClient.NewSSEMCPClient(config.Host + "/sse")
	if err != nil {
		return nil, err
	}

	return &Client{
		config:       config,
		sseMCPClient: sseMCPClient,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return c.sseMCPClient.Start(ctx)
}

func (c *Client) Close() error {
	return c.sseMCPClient.Close()
}

func (c *Client) ListTools(ctx context.Context) ([]string, error) {
	result, err := c.sseMCPClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, err
	}

	toolJsonList := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		toolJson, err := tool.MarshalJSON()
		if err != nil {
			return nil, err
		}
		toolJsonList = append(toolJsonList, string(toolJson))
	}
	return toolJsonList, nil
}

func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) ([]string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = arguments
	result, err := c.sseMCPClient.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		return nil, errors.New("error while calling mcp tool")
	}
	var contentList []string
	for _, content := range result.Content {
		if content != nil {
			contentBytes, err := json.Marshal(content)
			if err != nil {
				return nil, err
			}
			contentList = append(contentList, string(contentBytes))
		}
	}
	return contentList, nil
}

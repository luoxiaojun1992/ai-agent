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

type ClientType string

const (
	ClientTypeSSE    ClientType = "sse"
	ClientTypeStream ClientType = "stream"
)

type Config struct {
	Host       string
	ClientType ClientType
}

type Client struct {
	config        *Config
	mcpClientImpl *mcpClient.Client
}

func NewClient(config *Config) (*Client, error) {
	mcpClientImpl, err := newMcpClient(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:        config,
		mcpClientImpl: mcpClientImpl,
	}, nil
}

func newMcpClient(config *Config) (*mcpClient.Client, error) {
	switch config.ClientType {
	case ClientTypeSSE:
		return mcpClient.NewSSEMCPClient(config.Host + "/sse")
	case ClientTypeStream:
		return mcpClient.NewStreamableHttpClient(config.Host)
	default:
		return nil, errors.New("invalid client type")
	}
}

func (c *Client) Initialize(ctx context.Context) error {
	// Start
	if err := c.mcpClientImpl.Start(ctx); err != nil {
		return err
	}

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-client",
		Version: "1.0.0",
	}

	if _, err := c.mcpClientImpl.Initialize(ctx, initRequest); err != nil {
		return err
	}

	// Test Ping
	return c.mcpClientImpl.Ping(ctx)
}

func (c *Client) Close() error {
	return c.mcpClientImpl.Close()
}

func (c *Client) ListTools(ctx context.Context) ([]string, error) {
	result, err := c.mcpClientImpl.ListTools(ctx, mcp.ListToolsRequest{})
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
	result, err := c.mcpClientImpl.CallTool(ctx, req)
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

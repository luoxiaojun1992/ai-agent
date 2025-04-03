package impl

import (
	"context"

	"github.com/luoxiaojun1992/ai-agent/pkg/mcp"
)

type MCP struct {
	MCPClient *mcp.Client
}

func (m *MCP) GetDescription() string {
	//todo
	return ""
}

func (m *MCP) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo
	return nil
}

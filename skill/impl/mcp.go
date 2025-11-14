package impl

import (
	"context"
	"errors"

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
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for mcp skill")
	}

	name, hasName := params["name"]
	if !hasName {
		return errors.New("not found name from params")
	}
	nameStr, isValidName := name.(string)
	if !isValidName {
		return errors.New("error converting name from params")
	}

	arguments, hasArguments := params["arguments"]
	if !hasArguments {
		return errors.New("not found arguments from params")
	}
	//todo test
	argumentMap, isValidArarguments := arguments.(map[string]interface{})
	if !isValidArarguments {
		return errors.New("error converting arguments from params")
	}

	result, err := m.MCPClient.CallTool(ctx, nameStr, argumentMap)
	if err != nil {
		return err
	}
	_, err = callback(result)
	return err
}

package impl

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/luoxiaojun1992/ai-agent/pkg/mcp"
)

type MCP struct {
	MCPClient *mcp.Client
}

func (m *MCP) GetDescription() string {
	description := `Call MCP (Model Context Protocol) tools and services. This skill enables interaction with external tools and services through the MCP protocol.
1. Tool list:
%s
2. Parameters:
- name: string - The name of the MCP tool or service to call
- arguments: map[string]interface{} - Arguments to pass to the MCP tool
3. Returns: Result from the MCP tool execution`

	//todo pass ctx from outside
	//todo return err
	ctx := context.Background()
	toolDescList, err := m.MCPClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list mcp tools: %v", err)
	}
	toolListDescription := strings.Join(toolDescList, "\n\n")
	return fmt.Sprintf(description, toolListDescription)
}

func (m *MCP) ShortDescription() string {
	return "Call MCP tools and services"
}

func (m *MCP) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	//todo remove debug log
	log.Println(cmdCtx)

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
	
	// Handle different types of arguments
	var argumentMap map[string]interface{}
	switch v := arguments.(type) {
	case map[string]interface{}:
		argumentMap = v
	default:
		return fmt.Errorf("error converting arguments from params: expected map[string]interface{}, got %T", arguments)
	}

	//todo remove debug log
	log.Println(nameStr)
	log.Println(argumentMap)

	result, err := m.MCPClient.CallTool(ctx, nameStr, argumentMap)
	if err != nil {
		return err
	}
	_, err = callback(result)
	return err
}

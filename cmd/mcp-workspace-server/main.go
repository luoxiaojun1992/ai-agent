package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/pathutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultPort         = "8080"
	workspaceRootDir    = "/workspace-root"
	serverCodeDir       = "/app"
	defaultWorkspaceDir = "default"
)

type config struct {
	port      string
	workspace string
}

func loadConfig() (*config, error) {
	workspace, err := resolveWorkspaceDir(workspaceRootDir, serverCodeDir, os.Getenv("WORKSPACE_DIR"))
	if err != nil {
		return nil, err
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	return &config{port: port, workspace: workspace}, nil
}

func resolveWorkspaceDir(allowedRootDir, codeDir, rawWorkspace string) (string, error) {
	if strings.TrimSpace(allowedRootDir) == "" {
		return "", errors.New("allowed root dir is required")
	}
	if strings.TrimSpace(codeDir) == "" {
		return "", errors.New("code dir is required")
	}

	absRoot, err := filepath.Abs(filepath.Clean(allowedRootDir))
	if err != nil {
		return "", fmt.Errorf("resolve allowed root dir: %w", err)
	}
	absCodeDir, err := filepath.Abs(filepath.Clean(codeDir))
	if err != nil {
		return "", fmt.Errorf("resolve code dir: %w", err)
	}

	workspaceInput := strings.TrimSpace(rawWorkspace)
	if workspaceInput == "" {
		workspaceInput = defaultWorkspaceDir
	}

	workspace, err := pathutil.ResolvePath(absRoot, workspaceInput)
	if err != nil {
		return "", fmt.Errorf("resolve workspace dir: %w", err)
	}

	if workspace == absRoot {
		return "", errors.New("workspace dir must be a subdirectory of allowed root dir")
	}
	if !sameOrSubPath(workspace, absRoot) {
		return "", errors.New("workspace dir must be under allowed root dir")
	}
	if sameOrSubPath(workspace, absCodeDir) {
		return "", errors.New("workspace dir cannot be server code dir")
	}

	return workspace, nil
}

func sameOrSubPath(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func main() {
	validateOnly := flag.Bool("validate-config", false, "validate env config then print workspace path")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid workspace config: %v", err)
	}

	if *validateOnly {
		fmt.Println(cfg.workspace)
		return
	}

	if err := os.MkdirAll(cfg.workspace, 0o775); err != nil {
		log.Fatalf("create workspace dir: %v", err)
	}

	mcpServer := server.NewMCPServer("workspace-filesystem", "1.0.0")
	registerWorkspaceTools(mcpServer, cfg.workspace)

	httpServer := server.NewStreamableHTTPServer(mcpServer)

	log.Printf("starting workspace MCP server on :%s", cfg.port)
	log.Printf("workspace root: %s", cfg.workspace)
	log.Printf("mcp endpoint: /mcp")
	if err := httpServer.Start(":" + cfg.port); err != nil {
		log.Fatalf("start server: %v", err)
	}
}

func registerWorkspaceTools(mcpServer *server.MCPServer, workspaceRoot string) {
	mcpServer.AddTool(mcp.NewTool(
		"workspace_info",
		mcp.WithDescription("Get the current workspace root directory."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		_ = request
		return mcp.NewToolResultJSON(map[string]any{
			"workspace": workspaceRoot,
		})
	})

	mcpServer.AddTool(mcp.NewTool(
		"list_directory",
		mcp.WithDescription("List files and directories under workspace. Path can be relative to workspace or absolute under workspace."),
		mcp.WithString("path", mcp.Description("Directory path. Defaults to current workspace root.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		target, err := resolveTargetPath(workspaceRoot, request.GetString("path", "."))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid path", err), nil
		}
		entries, err := os.ReadDir(target)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("read directory failed", err), nil
		}

		type item struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Size  int64  `json:"size"`
			MTime string `json:"mtime"`
		}

		items := make([]item, 0, len(entries))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				return mcp.NewToolResultErrorFromErr("stat directory entry failed", err), nil
			}
			kind := "file"
			if e.IsDir() {
				kind = "dir"
			}
			items = append(items, item{
				Name:  e.Name(),
				Type:  kind,
				Size:  info.Size(),
				MTime: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

		rel, _ := filepath.Rel(workspaceRoot, target)
		if rel == "." {
			rel = "/"
		}
		return mcp.NewToolResultJSON(map[string]any{
			"path":    rel,
			"entries": items,
		})
	})

	mcpServer.AddTool(mcp.NewTool(
		"read_file",
		mcp.WithDescription("Read file content from workspace."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to read.")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		target, err := resolveTargetPath(workspaceRoot, request.GetString("path", ""))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid path", err), nil
		}

		content, err := os.ReadFile(target)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("read file failed", err), nil
		}

		rel, _ := filepath.Rel(workspaceRoot, target)
		return mcp.NewToolResultJSON(map[string]any{
			"path":    rel,
			"content": string(content),
		}), nil
	})

	mcpServer.AddTool(mcp.NewTool(
		"write_file",
		mcp.WithDescription("Write text content into workspace file. Creates parent directories when missing."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to write.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Text content to write.")),
		mcp.WithBoolean("overwrite", mcp.Description("Whether to overwrite if file exists. Default true.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		target, err := resolveTargetPath(workspaceRoot, request.GetString("path", ""))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid path", err), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid content", err), nil
		}
		overwriteEnabled, err := parseOptionalBool(request, "overwrite", true)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid overwrite", err), nil
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o775); err != nil {
			return mcp.NewToolResultErrorFromErr("create parent directory failed", err), nil
		}

		flags := os.O_CREATE | os.O_WRONLY
		if overwriteEnabled {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_EXCL
		}
		file, err := os.OpenFile(target, flags, 0o664)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("open file failed", err), nil
		}
		defer file.Close()
		if _, err := file.WriteString(content); err != nil {
			return mcp.NewToolResultErrorFromErr("write file failed", err), nil
		}

		rel, _ := filepath.Rel(workspaceRoot, target)
		return mcp.NewToolResultJSON(map[string]any{
			"status": "ok",
			"path":   rel,
		}), nil
	})

	mcpServer.AddTool(mcp.NewTool(
		"create_directory",
		mcp.WithDescription("Create directory in workspace recursively."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to create.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		target, err := resolveTargetPath(workspaceRoot, request.GetString("path", ""))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid path", err), nil
		}

		if err := os.MkdirAll(target, 0o775); err != nil {
			return mcp.NewToolResultErrorFromErr("create directory failed", err), nil
		}

		rel, _ := filepath.Rel(workspaceRoot, target)
		return mcp.NewToolResultJSON(map[string]any{
			"status": "ok",
			"path":   rel,
		}), nil
	})

	mcpServer.AddTool(mcp.NewTool(
		"remove_path",
		mcp.WithDescription("Remove file or directory from workspace."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to remove.")),
		mcp.WithBoolean("recursive", mcp.Description("Required for non-empty directories.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_ = ctx
		target, err := resolveTargetPath(workspaceRoot, request.GetString("path", ""))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid path", err), nil
		}

		info, err := os.Stat(target)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("stat path failed", err), nil
		}

		recursive, err := parseOptionalBool(request, "recursive", false)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid recursive", err), nil
		}

		if info.IsDir() {
			if recursive {
				err = os.RemoveAll(target)
			} else {
				err = os.Remove(target)
			}
		} else {
			err = os.Remove(target)
		}
		if err != nil {
			return mcp.NewToolResultErrorFromErr("remove path failed", err), nil
		}

		rel, _ := filepath.Rel(workspaceRoot, target)
		return mcp.NewToolResultJSON(map[string]any{
			"status": "ok",
			"path":   rel,
		}), nil
	})
}

func parseOptionalBool(request mcp.CallToolRequest, key string, defaultValue bool) (bool, error) {
	value, ok := request.GetArguments()[key]
	if !ok {
		return defaultValue, nil
	}

	parsed, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be boolean", key)
	}
	return parsed, nil
}

func resolveTargetPath(workspaceRoot, requestedPath string) (string, error) {
	if strings.TrimSpace(requestedPath) == "" {
		return "", errors.New("path is required")
	}
	return pathutil.ResolvePath(workspaceRoot, requestedPath)
}

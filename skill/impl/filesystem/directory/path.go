package directory

import "github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/pathutil"

func resolvePath(rootDir, pathStr string) (string, error) {
	return pathutil.ResolvePath(rootDir, pathStr)
}

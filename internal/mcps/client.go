package mcps

import (
	"os"
	"path"
	"runtime"
	"strings"

	mcpc "github.com/mark3labs/mcp-go/client"
)

type McpClient struct {
	client   mcpc.MCPClient
	provider string
}

func isLocal(provider string) bool {
	return !strings.HasPrefix(provider, "http")
}

func NewClient(provider string) (*McpClient, error) {
	if isLocal(provider) {
		return NewLocalClient(provider)
	} else {
		return NewRemoteClient(provider)
	}
}

func NewLocalClient(provider string) (*McpClient, error) {
	exeName, args := buildExecutable(provider)

	client, err := mcpc.NewStdioMCPClient(exeName, []string{}, args...)
	if err != nil {
		return nil, err
	}
	return &McpClient{
		client:   client,
		provider: provider,
	}, nil
}

func NewRemoteClient(provider string) (*McpClient, error) {
	client, err := mcpc.NewSSEMCPClient(provider)
	if err != nil {
		return nil, err
	}
	return &McpClient{
		client:   client,
		provider: provider,
	}, nil
}

func buildExecutable(provider string) (string, []string) {
	cmds := strings.Split(provider, " ")
	ext := path.Ext(cmds[0])
	switch ext {
	case ".py":
		return buildPythonExecutable(cmds[0], cmds[1:])
	case ".js":
		return buildJavascriptExecutable(cmds[0], cmds[1:])
	case ".ts":
		return buildTypeScriptExecutable(cmds[0], cmds[1:])
	case ".go":
		return buildGoExecutable(cmds[0], cmds[1:])
	case ".sh", ".bash", ".ps1":
		return currentShell(), cmds
	default:
		return cmds[0], cmds[1:]
	}
}

func currentShell() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	}
	return "/bin/bash"
}

func shellExt() string {
	if runtime.GOOS == "windows" {
		return ".ps1"
	}
	return ".sh"
}

// run python script, use virtual env if exists
func buildPythonExecutable(provider string, args []string) (string, []string) {
	folder, name := path.Split(provider)
	startScriptFile := path.Join(folder, ".mcp.start") + shellExt()
	startScript := []string{
		"cd " + folder,
	}
	if _, err := os.Stat(path.Join(folder, ".venv")); os.IsNotExist(err) {
		// no virtual env, use system python
		// `python3` is specific because `python` may be python2
		startScript = append(startScript, "python3 "+name+" "+strings.Join(args, " "))
	} else {
		if runtime.GOOS == "windows" {
			startScript = append(startScript, ".venv\\Scripts\\activate.ps1")
		} else {
			startScript = append(startScript, "source .venv/bin/activate")
		}
		startScript = append(startScript, "python "+name+" "+strings.Join(args, " "))
	}

	os.WriteFile(startScriptFile, []byte(strings.Join(startScript, "\n")), 0755)

	return currentShell(), []string{startScriptFile}
}

// use node to run javascript
func buildJavascriptExecutable(provider string, args []string) (string, []string) {
	folder, name := path.Split(provider)
	startScriptFile := path.Join(folder, ".mcp.start") + shellExt()
	startScript := []string{}

	startScript = append(startScript, "cd "+folder)
	startScript = append(startScript, "node "+name+" "+strings.Join(args, " "))

	os.WriteFile(startScriptFile, []byte(strings.Join(startScript, "\n")), 0755)

	return currentShell(), []string{startScriptFile}
}

// use bun to run typescript
func buildTypeScriptExecutable(provider string, args []string) (string, []string) {
	folder, name := path.Split(provider)
	startScriptFile := path.Join(folder, ".mcp.start") + shellExt()
	startScript := []string{}

	startScript = append(startScript, "cd "+folder)
	startScript = append(startScript, "bun "+name+" "+strings.Join(args, " "))

	os.WriteFile(startScriptFile, []byte(strings.Join(startScript, "\n")), 0755)

	return currentShell(), []string{startScriptFile}
}

func buildGoExecutable(provider string, args []string) (string, []string) {
	folder, name := path.Split(provider)
	startScriptFile := path.Join(folder, ".mcp.start") + shellExt()
	startScript := []string{}

	startScript = append(startScript, "cd "+folder)
	startScript = append(startScript, "go run "+name+" "+strings.Join(args, " "))

	os.WriteFile(startScriptFile, []byte(strings.Join(startScript, "\n")), 0755)

	return currentShell(), []string{startScriptFile}
}

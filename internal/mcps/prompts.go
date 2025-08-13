package mcps

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	mcpc "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var mcpPrompts = sync.Map{}

func GetPrompt(name string) (string, bool) {
	value, ok := mcpPrompts.Load(name)
	if !ok {
		return "", false
	}
	return value.(string), true
}

func updateMcpPrompt(client mcpc.MCPClient) {
	ctx := context.Background()
	lists, err := client.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		slog.Warn("Failed to list MCP prompts", "error", err)
		return
	}
	for _, prompt := range lists.Prompts {
		promptContent, err := client.GetPrompt(ctx, mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Name: prompt.Name,
			},
		})
		if err != nil {
			continue
		}
		messages := make([]string, 0, len(promptContent.Messages))
		for _, message := range promptContent.Messages {
			if textContent, ok := message.Content.(mcp.TextContent); ok {
				messages = append(messages, textContent.Text)
			}
		}
		body := strings.Join(messages, "\n")
		slog.Debug("Update MCP prompt", "name", prompt.Name, "body", body[0:100])
		mcpPrompts.Store(prompt.Name, body)
	}
}

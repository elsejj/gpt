package mcps

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

type MCPs struct {
	toolToClient map[string]*McpClient
	clients      []*McpClient
	Tools        []openai.ChatCompletionToolParam
}

func New(providers ...string) (*MCPs, error) {
	mcps := &MCPs{
		toolToClient: make(map[string]*McpClient),
		clients:      make([]*McpClient, 0),
	}

	for _, provider := range providers {

		client, err := NewClient(provider)
		if err != nil {
			slog.Warn("failed to create client", "provider", provider, "error", err)
			mcps.Shutdown()
			return nil, err
		}

		mcps.clients = append(mcps.clients, client)
	}

	ctx := context.Background()
	tools := make([]openai.ChatCompletionToolParam, 0)
	for _, client := range mcps.clients {
		_, err := client.client.Initialize(ctx, mcp.InitializeRequest{})
		if err != nil {
			slog.Warn("failed to initialize client", "provider", client.provider, "error", err)
			mcps.Shutdown()
			return nil, err
		}
		resp, err := client.client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			slog.Warn("failed to list tools", "provider", client.provider, "error", err)
			mcps.Shutdown()
			return nil, err
		}
		for _, tool := range resp.Tools {
			mcps.toolToClient[tool.Name] = client
			params := map[string]any{
				"type":       tool.InputSchema.Type,
				"properties": tool.InputSchema.Properties,
				"required":   tool.InputSchema.Required,
			}
			description := tool.Description
			tools = append(tools, openai.ChatCompletionToolParam{
				Type: openai.F(openai.ChatCompletionToolTypeFunction),
				Function: openai.F(openai.FunctionDefinitionParam{
					Name:        openai.String(tool.Name),
					Description: openai.String(description),
					Parameters:  openai.F(shared.FunctionParameters(params)),
				}),
			})
		}
	}
	mcps.Tools = tools

	return mcps, nil
}

func (m *MCPs) Shutdown() {
	for _, client := range m.clients {
		client.client.Close()
	}
	m.clients = nil
	m.toolToClient = nil
}
func (m *MCPs) CallToolOpenAI(ctx context.Context, toolCall openai.ChatCompletionChunkChoicesDeltaToolCall) (openai.ChatCompletionMessageParamUnion, error) {
	toolName := toolCall.Function.Name
	callID := toolCall.ID
	args := make(map[string]any)

	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		return nil, err
	}

	return m.CallTool(ctx, callID, toolName, args)
}

func (m *MCPs) CallTool(ctx context.Context, callID string, toolName string, args map[string]any) (openai.ChatCompletionMessageParamUnion, error) {

	client, ok := m.toolToClient[toolName]
	if !ok {
		return nil, errors.New(toolName + " tool not found")
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args

	resp, err := client.client.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.IsError {
		errMsg := ""
		if len(resp.Content) > 0 {
			textContent, ok := resp.Content[0].(mcp.TextContent)
			if ok {
				errMsg = textContent.Text
			}
		}
		return nil, errors.New(toolName + " call tool error:" + errMsg)
	}

	if len(resp.Content) == 0 {
		return nil, errors.New(toolName + " no content")
	}

	callResult, ok := resp.Content[0].(mcp.TextContent)
	if !ok {
		return nil, errors.New(toolName + "invalid content type")
	}

	return openai.ToolMessage(callID, callResult.Text), nil
}

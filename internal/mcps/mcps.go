package mcps

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
)

// MCPs is a collection of MCP clients.
// It manages the lifecycle of the clients and provides a single entry point for calling tools.
type MCPs struct {
	toolToClient map[string]*McpClient
	clients      []*McpClient
	Tools        []openai.ChatCompletionToolUnionParam
}

// New creates a new MCPs instance.
// It initializes the clients and lists the available tools.
func New(providers ...string) (*MCPs, error) {
	mcps := &MCPs{
		toolToClient: make(map[string]*McpClient),
		clients:      make([]*McpClient, 0),
		Tools:        make([]openai.ChatCompletionToolUnionParam, 0),
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
	tools := make([]openai.ChatCompletionToolUnionParam, 0)
	for _, client := range mcps.clients {
		_, err := client.client.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: "2025-03-26",
			},
		})
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
			tools = append(tools, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(description),
				Parameters:  shared.FunctionParameters(params),
			}))
		}

		updateMcpPrompt(client.client)
	}
	mcps.Tools = tools

	return mcps, nil
}

// Shutdown closes all the MCP clients.
func (m *MCPs) Shutdown() {
	for _, client := range m.clients {
		client.client.Close()
	}
	m.clients = nil
	m.toolToClient = nil
}

// CallToolOpenAI calls a tool with the given name and arguments.
// It is a wrapper around CallTool that takes an openai.ChatCompletionChunkChoiceDeltaToolCall.
func (m *MCPs) CallToolOpenAI(ctx context.Context, toolCall openai.ChatCompletionChunkChoiceDeltaToolCall) (openai.ChatCompletionMessageParamUnion, error) {
	toolName := toolCall.Function.Name
	callID := toolCall.ID
	args := make(map[string]any)

	err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	if err != nil {
		return openai.ChatCompletionMessageParamUnion{}, err
	}

	return m.CallTool(ctx, callID, toolName, args)
}

// CallTool calls a tool with the given name and arguments.
// It returns the result of the tool call as a ChatCompletionMessageParamUnion.
func (m *MCPs) CallTool(ctx context.Context, callID string, toolName string, args map[string]any) (openai.ChatCompletionMessageParamUnion, error) {

	client, ok := m.toolToClient[toolName]
	if !ok {
		return openai.ChatCompletionMessageParamUnion{}, errors.New(toolName + " tool not found")
	}
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args

	resp, err := client.client.CallTool(ctx, req)
	if err != nil {
		return openai.ChatCompletionMessageParamUnion{}, err
	}

	if resp.IsError {
		errMsg := ""
		if len(resp.Content) > 0 {
			textContent, ok := resp.Content[0].(mcp.TextContent)
			if ok {
				errMsg = textContent.Text
			}
		}
		return openai.ChatCompletionMessageParamUnion{}, errors.New(toolName + " call tool error:" + errMsg)
	}

	if len(resp.Content) == 0 {
		return openai.ChatCompletionMessageParamUnion{}, errors.New(toolName + " no content")
	}

	callResult, ok := resp.Content[0].(mcp.TextContent)
	if !ok {
		return openai.ChatCompletionMessageParamUnion{}, errors.New(toolName + "invalid content type")
	}

	return openai.ToolMessage(callResult.Text, callID), nil
}

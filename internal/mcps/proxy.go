package mcps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/mark3labs/mcp-go/mcp"
)

type PromptDef struct {
	Name        string              `json:"name" yaml:"name"`
	Description string              `json:"description" yaml:"description"`
	Messages    []mcp.PromptMessage `json:"messages" yaml:"messages"`
}

type ToolDef struct {
	Name        string              `json:"name" yaml:"name"`
	URL         string              `json:"url" yaml:"url"`
	Method      string              `json:"method,omitempty" yaml:"method,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	InputSchema mcp.ToolInputSchema `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
}

// ProxyMCPClient 实现了 MCPClient 接口，将 HTTP 服务代理为 MCP 服务
type ProxyMCPClient struct {
	Tools      []ToolDef    `json:"tools" yaml:"tools"`
	Prompts    []PromptDef  `json:"prompts" yaml:"prompts"`
	httpClient *http.Client `json:"-" yaml:"-"`
}

// NewProxyMCPClient 从配置文件创建一个新的 ProxyMCPClient
func NewProxyMCPClient(configPath string) (*ProxyMCPClient, error) {

	client, err := loadClientConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client.httpClient = &http.Client{}

	return client, nil
}

func IsProxyMCPConfig(name string) bool {
	if !strings.HasSuffix(name, "mcp.json") && !strings.HasSuffix(name, "mcp.yaml") {
		return false
	}
	// test it's a file
	info, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// loadClientConfig 从文件加载配置
func loadClientConfig(configPath string) (*ProxyMCPClient, error) {

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ProxyMCPClient
	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".json":
		err = json.Unmarshal(data, &config)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Initialize 实现 MCPClient.Initialize
func (p *ProxyMCPClient) Initialize(
	ctx context.Context,
	request mcp.InitializeRequest,
) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{
		ProtocolVersion: "2025-03-26",
		ServerInfo: mcp.Implementation{
			Name:    "ProxyMCP",
			Version: "1.0.0",
		},
		Capabilities: mcp.ServerCapabilities{},
	}, nil
}

// Ping 实现 MCPClient.Ping
func (p *ProxyMCPClient) Ping(ctx context.Context) error {
	// 对于代理客户端，ping 总是成功的
	return nil
}

// ListTools 实现 MCPClient.ListTools
func (p *ProxyMCPClient) ListTools(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {

	tools := make([]mcp.Tool, len(p.Tools))
	for i, toolDef := range p.Tools {
		tools[i] = mcp.Tool{
			Name:        toolDef.Name,
			Description: toolDef.Description,
			InputSchema: toolDef.InputSchema,
		}
	}

	return &mcp.ListToolsResult{
		Tools: tools,
	}, nil
}

// ListToolsByPage 实现 MCPClient.ListToolsByPage
func (p *ProxyMCPClient) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return p.ListTools(ctx, request)
}

func toolRequest(tool *ToolDef, callRequest *mcp.CallToolRequest) (*http.Request, error) {

	parsedURL, err := url.Parse(tool.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL for tool: %s, error: %w", tool.Name, err)
	}
	qs := parsedURL.Query()
	body := bytes.NewBuffer(nil)
	if tool.Method == "GET" {
		switch args := callRequest.Params.Arguments.(type) {
		case string:
			qs.Set(callRequest.Params.Name, args)
		case map[string]any:
			for k, v := range args {
				qs.Set(k, fmt.Sprintf("%v", v))
			}
		default:
			return nil, fmt.Errorf("unsupported argument type for GET request: %T", args)
		}
		parsedURL.RawQuery = qs.Encode()
	} else {
		enc := json.NewEncoder(body)
		if err := enc.Encode(callRequest.Params.Arguments); err != nil {
			return nil, fmt.Errorf("failed to encode arguments: %w", err)
		}
	}

	return &http.Request{
		Method: tool.Method,
		URL:    parsedURL,
		Body:   io.NopCloser(body),
	}, nil
}

// CallTool 实现 MCPClient.CallTool
func (p *ProxyMCPClient) CallTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 查找工具定义
	var toolDef *ToolDef
	for _, tool := range p.Tools {
		if tool.Name == request.Params.Name {
			toolDef = &tool
			break
		}
	}

	if toolDef == nil {
		return nil, fmt.Errorf("tool not found: %s", request.Params.Name)
	}

	req, err := toolRequest(toolDef, &request)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool request: %w", err)
	}

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode >= 400 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(respBody)),
				},
			},
			IsError: true,
		}, nil
	}

	// 返回成功响应
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(respBody),
			},
		},
	}, nil
}

// ListPrompts 实现 MCPClient.ListPrompts
func (p *ProxyMCPClient) ListPrompts(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {

	prompts := make([]mcp.Prompt, len(p.Prompts))
	for i, promptDef := range p.Prompts {

		prompts[i] = mcp.Prompt{
			Name:        promptDef.Name,
			Description: promptDef.Description,
		}
	}

	return &mcp.ListPromptsResult{
		Prompts: prompts,
	}, nil
}

// ListPromptsByPage 实现 MCPClient.ListPromptsByPage
func (p *ProxyMCPClient) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return p.ListPrompts(ctx, request)
}

// GetPrompt 实现 MCPClient.GetPrompt
func (p *ProxyMCPClient) GetPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {

	var promptDef *PromptDef
	for _, prompt := range p.Prompts {
		if prompt.Name == request.Params.Name {
			promptDef = &prompt
			break
		}
	}

	if promptDef == nil {
		return nil, fmt.Errorf("prompt not found: %s", request.Params.Name)
	}

	return &mcp.GetPromptResult{
		Description: promptDef.Description,
		Messages:    promptDef.Messages,
	}, nil
}

// ListResources 实现 MCPClient.ListResources
func (p *ProxyMCPClient) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, fmt.Errorf("resource listing not implemented for proxy client")
}

// ListResourcesByPage 实现 MCPClient.ListResourcesByPage
func (p *ProxyMCPClient) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return p.ListResources(ctx, request)
}

// ListResourceTemplates 实现 MCPClient.ListResourceTemplates
func (p *ProxyMCPClient) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, fmt.Errorf("resource templates not implemented for proxy client")
}

// ListResourceTemplatesByPage 实现 MCPClient.ListResourceTemplatesByPage
func (p *ProxyMCPClient) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return p.ListResourceTemplates(ctx, request)
}

// ReadResource 实现 MCPClient.ReadResource
func (p *ProxyMCPClient) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, fmt.Errorf("resource reading not implemented for proxy client")
}

// Subscribe 实现 MCPClient.Subscribe
func (p *ProxyMCPClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	// 代理客户端不支持订阅
	return fmt.Errorf("subscription not supported by proxy client")
}

// Unsubscribe 实现 MCPClient.Unsubscribe
func (p *ProxyMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	// 代理客户端不支持订阅
	return fmt.Errorf("subscription not supported by proxy client")
}

// SetLevel 实现 MCPClient.SetLevel
func (p *ProxyMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	// 代理客户端忽略日志级别设置
	return nil
}

// Complete 实现 MCPClient.Complete
func (p *ProxyMCPClient) Complete(
	ctx context.Context,
	request mcp.CompleteRequest,
) (*mcp.CompleteResult, error) {
	// 代理客户端不支持补全
	return nil, fmt.Errorf("completion not supported by proxy client")
}

// Close 实现 MCPClient.Close
func (p *ProxyMCPClient) Close() error {
	// 清理资源
	return nil
}

// OnNotification 实现 MCPClient.OnNotification
func (p *ProxyMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
}

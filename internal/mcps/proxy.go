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
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/mark3labs/mcp-go/mcp"
)

// PromptDef defines the structure of a prompt in the proxy configuration.
type PromptDef struct {
	Name        string              `json:"name" yaml:"name"`
	Description string              `json:"description" yaml:"description"`
	Messages    []mcp.PromptMessage `json:"messages" yaml:"messages"`
}

// ToolDef defines the structure of a tool in the proxy configuration.
type ToolDef struct {
	Name        string              `json:"name" yaml:"name"`
	URL         string              `json:"url" yaml:"url"`
	Method      string              `json:"method,omitempty" yaml:"method,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	InputSchema mcp.ToolInputSchema `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
}

// ProxyMCPClient implements the MCPClient interface and proxies HTTP services as MCP services.
type ProxyMCPClient struct {
	Tools      []ToolDef    `json:"tools" yaml:"tools"`
	Prompts    []PromptDef  `json:"prompts" yaml:"prompts"`
	httpClient *http.Client `json:"-" yaml:"-"`
}

// NewProxyMCPClient creates a new ProxyMCPClient from a configuration file.
func NewProxyMCPClient(configPath string) (*ProxyMCPClient, error) {

	client, err := loadClientConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	client.httpClient = &http.Client{}

	return client, nil
}

// IsProxyMCPConfig checks if the given name is a proxy MCP configuration file.
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

// loadClientConfig loads the configuration from a file.
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

// Initialize implements the MCPClient.Initialize method.
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

// Ping implements the MCPClient.Ping method.
func (p *ProxyMCPClient) Ping(ctx context.Context) error {
	// 对于代理客户端，ping 总是成功的
	return nil
}

// ListTools implements the MCPClient.ListTools method.
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

// ListToolsByPage implements the MCPClient.ListToolsByPage method.
func (p *ProxyMCPClient) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return p.ListTools(ctx, request)
}

func asQSValue(v any) string {
	if v == nil {
		return ""
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		// special-case []byte -> string
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// []byte
			return string(rv.Bytes())
		}
		parts := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			parts = append(parts, fmt.Sprintf("%v", elem))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprintf("%v", v)
	}
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
				qs.Set(k, asQSValue(v))
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

// CallTool implements the MCPClient.CallTool method.
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

// ListPrompts implements the MCPClient.ListPrompts method.
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

// ListPromptsByPage implements the MCPClient.ListPromptsByPage method.
func (p *ProxyMCPClient) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return p.ListPrompts(ctx, request)
}

// GetPrompt implements the MCPClient.GetPrompt method.
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

// ListResources implements the MCPClient.ListResources method.
func (p *ProxyMCPClient) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, fmt.Errorf("resource listing not implemented for proxy client")
}

// ListResourcesByPage implements the MCPClient.ListResourcesByPage method.
func (p *ProxyMCPClient) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return p.ListResources(ctx, request)
}

// ListResourceTemplates implements the MCPClient.ListResourceTemplates method.
func (p *ProxyMCPClient) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, fmt.Errorf("resource templates not implemented for proxy client")
}

// ListResourceTemplatesByPage implements the MCPClient.ListResourceTemplatesByPage method.
func (p *ProxyMCPClient) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return p.ListResourceTemplates(ctx, request)
}

// ReadResource implements the MCPClient.ReadResource method.
func (p *ProxyMCPClient) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, fmt.Errorf("resource reading not implemented for proxy client")
}

// Subscribe implements the MCPClient.Subscribe method.
func (p *ProxyMCPClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	// 代理客户端不支持订阅
	return fmt.Errorf("subscription not supported by proxy client")
}

// Unsubscribe implements the MCPClient.Unsubscribe method.
func (p *ProxyMCPClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	// 代理客户端不支持订阅
	return fmt.Errorf("subscription not supported by proxy client")
}

// SetLevel implements the MCPClient.SetLevel method.
func (p *ProxyMCPClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	// 代理客户端忽略日志级别设置
	return nil
}

// Complete implements the MCPClient.Complete method.
func (p *ProxyMCPClient) Complete(
	ctx context.Context,
	request mcp.CompleteRequest,
) (*mcp.CompleteResult, error) {
	// 代理客户端不支持补全
	return nil, fmt.Errorf("completion not supported by proxy client")
}

// Close implements the MCPClient.Close method.
func (p *ProxyMCPClient) Close() error {
	// 清理资源
	return nil
}

// OnNotification implements the MCPClient.OnNotification method.
func (p *ProxyMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
}

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
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// OpenApiMcpClient implements the MCPClient interface and proxies OpenAPI services as MCP services.
type OpenApiMcpClient struct {
	doc    libopenapi.Document
	model  *v3.Document
	client *http.Client
}

// NewOpenApiMcpClient creates a new OpenApiMcpClient from a configuration file.
func NewOpenApiMcpClient(configPath string) (*OpenApiMcpClient, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse openapi spec: %w", err)
	}

	model, errs := doc.BuildV3Model()
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to build v3 model: %v", errs)
	}

	return &OpenApiMcpClient{
		doc:    doc,
		model:  &model.Model,
		client: &http.Client{},
	}, nil
}

// IsOpenAPIProxyConfig checks if the given name is an openapi proxy MCP configuration file.
func IsOpenAPIProxyConfig(name string) bool {
	return strings.HasSuffix(name, ".openapi.json") || strings.HasSuffix(name, ".openapi.yaml") || strings.HasSuffix(name, ".openapi.yml")
}

// Initialize implements the MCPClient.Initialize method.
func (p *OpenApiMcpClient) Initialize(
	ctx context.Context,
	request mcp.InitializeRequest,
) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{
		ProtocolVersion: "2025-03-26",
		ServerInfo: mcp.Implementation{
			Name:    "OpenAPIProxyMCP",
			Version: "1.0.0",
		},
		Capabilities: mcp.ServerCapabilities{},
	}, nil
}

// Ping implements the MCPClient.Ping method.
func (p *OpenApiMcpClient) Ping(ctx context.Context) error {
	return nil
}

// ListTools implements the MCPClient.ListTools method.
func (p *OpenApiMcpClient) ListTools(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	var tools []mcp.Tool

	for pair := p.model.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
		path := pair.Key()
		pathItem := pair.Value()
		if pathItem.Get != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodGet, path, pathItem.Get))
		}
		if pathItem.Post != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodPost, path, pathItem.Post))
		}
		if pathItem.Put != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodPut, path, pathItem.Put))
		}
		if pathItem.Delete != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodDelete, path, pathItem.Delete))
		}
		if pathItem.Patch != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodPatch, path, pathItem.Patch))
		}
		if pathItem.Head != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodHead, path, pathItem.Head))
		}
		if pathItem.Options != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodOptions, path, pathItem.Options))
		}
		if pathItem.Trace != nil {
			tools = append(tools, p.createToolFromOperation(http.MethodTrace, path, pathItem.Trace))
		}
	}

	return &mcp.ListToolsResult{
		Tools: tools,
	}, nil
}

func (p *OpenApiMcpClient) createToolFromOperation(method, path string, op *v3.Operation) mcp.Tool {
	properties := make(map[string]any)
	required := []string{}

	for _, param := range op.Parameters {
		if param.Required != nil && *param.Required {
			required = append(required, param.Name)
		}
		properties[param.Name] = map[string]any{
			"type":        param.Schema.Schema().Type,
			"description": param.Description,
		}
	}

	if op.RequestBody != nil {
		for pair := op.RequestBody.Content.First(); pair != nil; pair = pair.Next() {
			mediaType := pair.Key()
			mediaTypeObj := pair.Value()
			if strings.HasPrefix(mediaType, "application/json") {
				schema := mediaTypeObj.Schema.Schema()
				if schema != nil && len(schema.Type) > 0 && schema.Type[0] == "object" {
					for propPair := schema.Properties.First(); propPair != nil; propPair = propPair.Next() {
						propName := propPair.Key()
						prop := propPair.Value()
						properties[propName] = map[string]any{
							"type":        prop.Schema().Type,
							"description": prop.Schema().Description,
						}
					}
					required = append(required, schema.Required...)
				}
			}
		}
	}

	return mcp.Tool{
		Name:        fmt.Sprintf("%s %s", method, path),
		Description: op.Summary,
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// ListToolsByPage implements the MCPClient.ListToolsByPage method.
func (p *OpenApiMcpClient) ListToolsByPage(
	ctx context.Context,
	request mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return p.ListTools(ctx, request)
}

// CallTool implements the MCPClient.CallTool method.
func (p *OpenApiMcpClient) CallTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	parts := strings.SplitN(request.Params.Name, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tool name: %s", request.Params.Name)
	}
	method := parts[0]
	path := parts[1]

	pathItem, _ := p.model.Paths.PathItems.Get(path)
	if pathItem == nil {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	var op *v3.Operation
	switch method {
	case http.MethodGet:
		op = pathItem.Get
	case http.MethodPost:
		op = pathItem.Post
	case http.MethodPut:
		op = pathItem.Put
	case http.MethodDelete:
		op = pathItem.Delete
	case http.MethodPatch:
		op = pathItem.Patch
	case http.MethodHead:
		op = pathItem.Head
	case http.MethodOptions:
		op = pathItem.Options
	case http.MethodTrace:
		op = pathItem.Trace
	}

	if op == nil {
		return nil, fmt.Errorf("operation not found: %s %s", method, path)
	}

	serverURL := p.model.Servers[0].URL
	reqURL := serverURL + path

	body := &bytes.Buffer{}
	params, _ := request.Params.Arguments.(map[string]interface{})

	for _, param := range op.Parameters {
		if val, ok := params[param.Name]; ok {
			switch param.In {
			case "path":
				reqURL = strings.Replace(reqURL, "{"+param.Name+"}", fmt.Sprintf("%v", val), 1)
			case "query":
				parsedURL, err := url.Parse(reqURL)
				if err != nil {
					return nil, fmt.Errorf("failed to parse url: %w", err)
				}
				q := parsedURL.Query()
				q.Add(param.Name, fmt.Sprintf("%v", val))
				parsedURL.RawQuery = q.Encode()
				reqURL = parsedURL.String()
			}
		}
	}

	if op.RequestBody != nil {
		bodyParams := make(map[string]interface{})
		for k, v := range params {
			isParam := false
			for _, p := range op.Parameters {
				if p.Name == k {
					isParam = true
					break
				}
			}
			if !isParam {
				bodyParams[k] = v
			}
		}
		if len(bodyParams) > 0 {
			jsonBody, err := json.Marshal(bodyParams)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewBuffer(jsonBody)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

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
func (p *OpenApiMcpClient) ListPrompts(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return &mcp.ListPromptsResult{
		Prompts: []mcp.Prompt{},
	}, nil
}

// ListPromptsByPage implements the MCPClient.ListPromptsByPage method.
func (p *OpenApiMcpClient) ListPromptsByPage(
	ctx context.Context,
	request mcp.ListPromptsRequest,
) (*mcp.ListPromptsResult, error) {
	return p.ListPrompts(ctx, request)
}

// GetPrompt implements the MCPClient.GetPrompt method.
func (p *OpenApiMcpClient) GetPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListResources implements the MCPClient.ListResources method.
func (p *OpenApiMcpClient) ListResources(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListResourcesByPage implements the MCPClient.ListResourcesByPage method.
func (p *OpenApiMcpClient) ListResourcesByPage(
	ctx context.Context,
	request mcp.ListResourcesRequest,
) (*mcp.ListResourcesResult, error) {
	return p.ListResources(ctx, request)
}

// ListResourceTemplates implements the MCPClient.ListResourceTemplates method.
func (p *OpenApiMcpClient) ListResourceTemplates(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListResourceTemplatesByPage implements the MCPClient.ListResourceTemplatesByPage method.
func (p *OpenApiMcpClient) ListResourceTemplatesByPage(
	ctx context.Context,
	request mcp.ListResourceTemplatesRequest,
) (*mcp.ListResourceTemplatesResult, error) {
	return p.ListResourceTemplates(ctx, request)
}

// ReadResource implements the MCPClient.ReadResource method.
func (p *OpenApiMcpClient) ReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// Subscribe implements the MCPClient.Subscribe method.
func (p *OpenApiMcpClient) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return fmt.Errorf("not implemented")
}

// Unsubscribe implements the MCPClient.Unsubscribe method.
func (p *OpenApiMcpClient) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return fmt.Errorf("not implemented")
}

// SetLevel implements the MCPClient.SetLevel method.
func (p *OpenApiMcpClient) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return nil
}

// Complete implements the MCPClient.Complete method.
func (p *OpenApiMcpClient) Complete(
	ctx context.Context,
	request mcp.CompleteRequest,
) (*mcp.CompleteResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// Close implements the MCPClient.Close method.
func (p *OpenApiMcpClient) Close() error {
	return nil
}

// OnNotification implements the MCPClient.OnNotification method.
func (p *OpenApiMcpClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
}

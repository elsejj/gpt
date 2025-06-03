package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/elsejj/gpt/internal/utils"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func Chat(conf *utils.AppConf, w io.Writer) error {
	if conf == nil || conf.Prompt == nil {
		return fmt.Errorf("config or prompt is nil")
	}

	client := openai.NewClient(
		option.WithAPIKey(conf.LLM.ApiKey),
		option.WithBaseURL(conf.LLM.Gateway),
		option.WithHeaderAdd("x-portkey-provider", conf.LLM.Provider),
	)

	ctx := context.Background()
	var messages []openai.ChatCompletionMessageParamUnion
	if conf.Prompt.System != "" {
		messages = append(messages, openai.SystemMessage(conf.Prompt.System))
	}
	if len(conf.Prompt.Images) == 0 {
		messages = append(messages, openai.UserMessage(conf.Prompt.User))
	} else {
		parts := []openai.ChatCompletionContentPartUnionParam{
			openai.TextPart(conf.Prompt.User),
		}
		for _, img := range conf.Prompt.Images {
			url := DataURLOfImageFile(img)
			if len(url) > 0 {
				parts = append(parts, openai.ImagePart(url))
			}
		}
		messages = append(messages, openai.UserMessageParts(parts...))
	}

	messages, usage, err := llmToolCall(ctx, client, messages, conf, w)
	if err != nil {
		return err
	}

	w.Write([]byte("\n"))

	slog.Debug("allMessages", "messages", messages)

	if conf.Prompt.WithUsage {
		slog.Info("Usage", "prompt", usage.PromptTokens, "completion", usage.CompletionTokens, "provider", conf.LLM.Provider, "model", conf.LLM.Model)
	}
	return nil
}

func DataURLOfImageFile(filePath string) string {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	base64Body := base64.StdEncoding.EncodeToString(body)
	mimeType := http.DetectContentType(body)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Body)
}

func ExtractCodeBlock(text []byte) []byte {
	codeStart := bytes.Index(text, []byte("```"))
	if codeStart < 0 {
		// no code block
		return text
	}
	nextLine := bytes.Index(text[codeStart:], []byte("\n"))
	if nextLine < 0 {
		codeStart += 3
	} else {
		codeStart += nextLine + 1
	}
	codeEnd := bytes.Index(text[codeStart:], []byte("```"))
	if codeEnd < 0 {
		// no code block
		return text
	}
	return text[codeStart : codeStart+codeEnd]
}

func llmToolCall(ctx context.Context, client *openai.Client, messages []openai.ChatCompletionMessageParamUnion, conf *utils.AppConf, w io.Writer) ([]openai.ChatCompletionMessageParamUnion, openai.CompletionUsage, error) {
	var totalUsage openai.CompletionUsage
	for {

		// let model to think whether to call tool
		req := openai.ChatCompletionNewParams{
			Model:    openai.F(conf.LLM.Model),
			Messages: openai.F(messages),
		}
		if conf.Prompt.MCPServers != nil && len(conf.Prompt.MCPServers.Tools) > 0 {
			req.Tools = openai.F(conf.Prompt.MCPServers.Tools)
			req.ToolChoice = openai.F(openai.ChatCompletionToolChoiceOptionUnionParam(openai.ChatCompletionToolChoiceOptionAutoAuto))
		}

		if conf.Prompt.WithUsage {
			req.StreamOptions = openai.F(
				openai.ChatCompletionStreamOptionsParam{
					IncludeUsage: openai.F(true),
				},
			)
		}

		if conf.Prompt.JsonMode {
			req.ResponseFormat = openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONObjectParam{
					Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
				},
			)
		}

		req.Temperature = openai.F(conf.Prompt.Temperature)

		body, _ := req.MarshalJSON()
		slog.Debug("Request", "body", string(body))

		s := client.Chat.Completions.NewStreaming(ctx, req)
		if s.Err() != nil {
			return messages, totalUsage, s.Err()
		}

		var usage openai.CompletionUsage
		toolCalls := make(map[int64]*openai.ChatCompletionChunkChoicesDeltaToolCall)
		for s.Next() {
			cur := s.Current()
			slog.Debug("stream", "chunk", cur.JSON.RawJSON())
			for _, c := range cur.Choices {
				w.Write([]byte(c.Delta.Content))
				for _, toolCall := range c.Delta.ToolCalls {
					tc, ok := toolCalls[c.Index]
					if !ok {
						tc = &toolCall
						toolCalls[c.Index] = tc
					} else {
						tc.Function.Arguments += toolCall.Function.Arguments
					}
				}
			}
			usage = cur.Usage
		}
		s.Close()

		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.CompletionTokens += usage.CompletionTokens
		totalUsage.TotalTokens += usage.TotalTokens

		// there are no tool calls
		if len(toolCalls) == 0 {
			slog.Debug("no tool call required")
			break
		}

		w.Write([]byte("\n"))
		toolCallMessages := make([]openai.ChatCompletionMessageParamUnion, 0)
		assistantToolCalls := make([]openai.ChatCompletionMessageToolCallParam, 0)
		for _, toolCall := range toolCalls {
			slog.Info("Model call ", "tool", toolCall.Function.Name, "args", toolCall.Function.Arguments)
			toolResult, err := conf.Prompt.MCPServers.CallToolOpenAI(ctx, *toolCall)
			if err != nil {
				slog.Error("Error calling tool", "tool", toolCall.Function.Name)
				return messages, totalUsage, err
			}
			assistantToolCalls = append(assistantToolCalls, openai.ChatCompletionMessageToolCallParam{
				ID: openai.F(toolCall.ID),
				Function: openai.F(openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      openai.F(toolCall.Function.Name),
					Arguments: openai.F(toolCall.Function.Arguments),
				}),
				Type: openai.F(openai.ChatCompletionMessageToolCallTypeFunction),
			})
			toolCallMessages = append(toolCallMessages, toolResult)
			slog.Info("Model call result ", "tool", toolCall.Function.Name, "result", toolResult)

		}
		assistantMessage := openai.ChatCompletionAssistantMessageParam{
			Role:      openai.F(openai.ChatCompletionAssistantMessageParamRoleAssistant),
			ToolCalls: openai.F(assistantToolCalls),
		}

		// there are tool results
		messages = append(messages, assistantMessage)
		messages = append(messages, toolCallMessages...)
	}

	return messages, totalUsage, nil
}

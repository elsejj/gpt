package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/elsejj/gpt/internal/utils"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/spf13/viper"
)

func debugChunk(c string) {
	var verbose = viper.GetInt("verbose")
	if verbose >= 3 {
		// detailed debug info
		slog.Debug("stream chunk", "chunk", c)
	} else if verbose >= 2 {
		// only log reason content
		var chunk map[string]any
		err := json.Unmarshal([]byte(c), &chunk)
		if err != nil {
			slog.Debug("stream chunk", "chunk", c)
			return
		}
		reasonContent := MGet(chunk, "choices.0.delta.reasoning_content", "")
		if len(reasonContent) > 0 {
			fmt.Printf("\033[37m%s\033[0m", reasonContent)
		}
	}

}

// Chat sends the user's prompt to the LLM and writes the response to the provided writer.
// It also handles tool calls and image data.
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
			openai.TextContentPart(conf.Prompt.User),
		}
		for _, img := range conf.Prompt.Images {
			url := DataURLOfImageFile(img)
			if len(url) > 0 {
				parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: url,
				}))
			}
		}
		messages = append(messages, openai.UserMessage(parts))
	}

	messages, usage, err := llmToolCall(ctx, &client, messages, conf, w)
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

// DataURLOfImageFile reads an image file and returns a data URL.
func DataURLOfImageFile(filePath string) string {
	body, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	base64Body := base64.StdEncoding.EncodeToString(body)
	mimeType := http.DetectContentType(body)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Body)
}

// ExtractCodeBlock extracts the first code block from the given text.
// It returns the code block without the backticks.
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

// llmToolCall handles the tool calling logic.
// It sends the request to the LLM, and if the LLM returns a tool call, it executes the tool and sends the result back to the LLM.
// It returns the final messages, the total usage, and any error that occurred.
func llmToolCall(ctx context.Context, client *openai.Client, messages []openai.ChatCompletionMessageParamUnion, conf *utils.AppConf, w io.Writer) ([]openai.ChatCompletionMessageParamUnion, openai.CompletionUsage, error) {
	var totalUsage openai.CompletionUsage
	for {

		// let model to think whether to call tool
		req := openai.ChatCompletionNewParams{
			Model:    conf.LLM.Model,
			Messages: messages,
		}
		if conf.Prompt.MCPServers != nil && len(conf.Prompt.MCPServers.Tools) > 0 {
			req.Tools = conf.Prompt.MCPServers.Tools
			req.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{}
		}

		if conf.Prompt.WithUsage {
			req.StreamOptions = openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.Bool(true),
			}
		}

		if conf.Prompt.JsonMode {
			req.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &openai.ResponseFormatJSONObjectParam{
					Type: "json_object",
				},
			}
		}

		req.Temperature = openai.Float(conf.Prompt.Temperature)

		body, _ := req.MarshalJSON()
		slog.Debug("Request", "body", string(body))

		s := client.Chat.Completions.NewStreaming(ctx, req)
		if s.Err() != nil {
			return messages, totalUsage, s.Err()
		}

		var usage openai.CompletionUsage
		toolCalls := make(map[int64]*openai.ChatCompletionChunkChoiceDeltaToolCall)
		for s.Next() {
			cur := s.Current()
			debugChunk(cur.RawJSON())
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
		assistantToolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0)
		for _, toolCall := range toolCalls {
			slog.Info("Model call ", "tool", toolCall.Function.Name, "args", toolCall.Function.Arguments)
			toolResult, err := conf.Prompt.MCPServers.CallToolOpenAI(ctx, *toolCall)
			if err != nil {
				slog.Error("Error calling tool", "tool", toolCall.Function.Name)
				return messages, totalUsage, err
			}
			assistantToolCalls = append(assistantToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: toolCall.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
					Type: "function",
				},
			})
			toolCallMessages = append(toolCallMessages, toolResult)
			slog.Info("Model call result ", "tool", toolCall.Function.Name, "result", toolResult)

		}
		assistantMessage := openai.ChatCompletionAssistantMessageParam{
			ToolCalls: assistantToolCalls,
		}

		// there are tool results
		messages = append(messages, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistantMessage})
		messages = append(messages, toolCallMessages...)
	}

	return messages, totalUsage, nil
}

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

	req := openai.ChatCompletionNewParams{
		Model:    openai.F(conf.LLM.Model),
		Messages: openai.F(messages),
	}

	if conf.Prompt.JsonMode {
		req.ResponseFormat = openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONObjectParam{
				Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
			},
		)
	}
	if conf.Prompt.WithUsage {
		req.StreamOptions = openai.F(
			openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.F(true),
			},
		)
	}

	if conf.Prompt.Verbose {
		body, _ := req.MarshalJSON()
		slog.Info("Request", "body", string(body))
	}

	s := client.Chat.Completions.NewStreaming(ctx, req)
	if s.Err() != nil {
		return s.Err()
	}

	var usage openai.CompletionUsage
	for s.Next() {
		cur := s.Current()
		if conf.Prompt.Verbose {
			slog.Info("stream", "chunk", cur.JSON.RawJSON())
		}
		for _, c := range cur.Choices {
			w.Write([]byte(c.Delta.Content))
		}
		usage = cur.Usage
	}
	s.Close()

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

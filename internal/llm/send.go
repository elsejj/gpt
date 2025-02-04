package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
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
		w.Write([]byte("Request: "))
		w.Write(body)
		w.Write([]byte("\n"))
	}

	s := client.Chat.Completions.NewStreaming(ctx, req)
	var usage openai.CompletionUsage
	for s.Next() {
		cur := s.Current()
		for _, c := range cur.Choices {
			w.Write([]byte(c.Delta.Content))
		}
		usage = cur.Usage
	}
	s.Close()

	if conf.Prompt.WithUsage {
		fmt.Fprintf(w, "\n(Prompt: %d, Completion: %d)\n", usage.PromptTokens, usage.CompletionTokens)
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

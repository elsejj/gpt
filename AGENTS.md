这是一个用 `Go` 语言编写的 `cli` 工具, 用于向兼任 OpenAI 协议的 LLM 服务发送请求.

# CI/CD

它使用 github 进行代码托管, 并通过 github action 进行编译发布, 当项目打上类似 `v0.2.7` 的标签时, 会触发编译, 参考 `.github/workflows/go.yml`

- `cmd/version.txt` 是当前的版本, 发布时, 需要先递增其中的版本号, 然后用 `git tag` 打标签后推送标签

# 代码结构

- `main.go`: 应用程序入口, 但仅简单调用 `cmd` 包
- `cmd`: 这个包是 cli 入口, 它使用 [Cobra: A Commander for modern Go CLI interactions](https://pkg.go.dev/github.com/spf13/cobra) 作为框架
  - `cmd/root.go`: 应用的入口, 请阅读其中的命令行参数生成部分了解可以使用的参数.
- `internal`: 这是一些内部使用的包
  - `internal/llm`: 对 [OpenAI Go SDK](https://pkg.go.dev/github.com/openai/openai-go/v3) 的封装以发送/接收 LLM 请求, 处理 MCP 调用等
  - `internal/mcps`: 将多个 MCP 服务, 聚合一个, 供大模型使用
  - `internal/utils`: 一些工具函数

其它的目录都不包含代码, 可以忽略

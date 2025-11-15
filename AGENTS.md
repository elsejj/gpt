This is a `cli` tool written in `Go` language for sending requests to LLM services that are compatible with the OpenAI protocol.

# Project Structure

- `main.go`: application entry point, just calls `cmd.Execute()`
- `cmd`: this package is the CLI entry point, using [Cobra: A Commander for modern Go CLI interactions](https://pkg.go.dev/github.com/spf13/cobra) as the framework
  - `cmd/root.go`: the entry point of the application, please read the command-line argument generation section to understand the available parameters.
- `internal`: these are some internal packages
  - `internal/llm`: a wrapper around the [OpenAI Go SDK](https://pkg.go.dev/github.com/openai/openai-go/v3) for sending/receiving LLM requests, handling MCP calls, etc. it provide a function named `Chat` to send requests to LLM services
  - `internal/mcps`: aggregates multiple MCP services into one for use by large models
  - `internal/utils`: some utility functions

other directories don't need to care about them.

# When implementing new features

1. Analyze user needs and design the feature
2. Review the existing codebase to understand how similar features are implemented
3. Implement the feature
4. Write tests to ensure the feature works as expected
5. Update documentation to reflect the new feature
6. Perform code reviews and testing to ensure quality

# Build and Test

- use `make local` to build a local binary, also use it as check syntax errors
- use `go test ./...` to run tests
- use default `GOCACHE` is okay, don't need to set it manually

Follow standard Go 1.20 practices: tabs for indentation, exported identifiers use CamelCase, and packages stay lower\*snake for clarity. Run gofmt -w on touched files and goimports if available to keep imports tidy

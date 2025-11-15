# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.12] - 2025-11-15

### Added

- `Tool` now supports an post action, which can be used to execute some after llm response is generated. There are following actions supported:
  - `output`: output the content to stdout (default), let action be empty or `output`
  - `copy`: copy the content to clipboard, let action to be `copy`
  - `save`: save the content to a file, let action be any file name
  - `execute`:
    - when action is `execute`, `run`, or `exec`, the content will be treated as a shell command and executed directly.
    - when action is a non-empty string other than above, it will be treated as a shell command template, and the content will be passed as argument to the command. For example, if action is `echo`, and content is `hello`, then the command executed will be `echo "hello"`.
  - see
    - [tr.toml](samples/tools/tr.toml) : a translation tool that copy the translation result to clipboard. `gpt -t tr "Hello, how are you?"` will copy `你好，你好吗？` to clipboard
    - [pa.toml](samples/tools/pa.toml) : a command assistant that build shell command from user input and execute it directly. `gpt -t pa "list all files in current directory"` will execute `ls -a` command directly.
- `-t` or `--tool` option to specify a tool file to use. `-T` is changed to set temperature.

## [0.2.11] - 2025-10-31

### Added

- Add '--tool' '-t' option to enable a tool use mode.

Tool is a pre-defined system prompt, model, and other configurations to do specific tasks. see [Tool](internal/tools/tools.go) for more details.

A example tool 'tr' is located at [tr.toml](samples/tools/tr.toml), which translate between chinese and english, you can use it like:

```bash
gpt -t samples/tools/tr.toml "Hello, how are you?"
```

This will output the translation result, `你好，你好吗？`

the `samples/tools/tr.toml` can be copied to `$HOME/.gpt/tools/tr.toml`, then you can use it like:

```bash
gpt -t tr "Hello, how are you?"
```

## [0.2.10] - 2025-10-26

### Added

- Added `reasonEffort` configuration to control how llm generate response

## [0.2.9] - 2024-10-15

### Changed

- Upgraded the `openai` SDK to v3.
- Improved changelog format.

## [0.2.8]

### Changed

- Verbose level 2 now outputs reason content, and verbose level 3 outputs the raw chunk response.

## [0.2.7]

### Added

- Allow using a `ProxyMCPClient` to expose any HTTP service as an MCP server, for example `gpt -M samples/qqwry.mcp.yaml "where is 120.197.169.198's location"`.

## [0.2.6]

### Changed

- Upgraded the `openai` SDK to v2.

### Added

- Allow referencing MCP prompts by name, for example `gpt -M "mcp_url" -s p1 user_prompt`.

## [0.2.5]

### Changed

- Upgraded the `openai` SDK to v1.

### Added

- Added streaming HTTP transport support for MCP, indicated by URLs without `sse`.

## [0.2.4]

### Added

- Enabled multi-round MCP calls.

## [0.2.3]

### Changed

- Improved compatibility.

### Added

- Support specifying only the model name.

## [0.2.2]

### Changed

- Improved compatibility with MCP servers.

## [0.2.1]

### Fixed

- Fixed MCP SSE so it starts before use.

## [0.2.0]

### Added

- Added support for the Model Content Protocol (MCP); `gpt` now accepts the `-M` option to specify the MCP server.

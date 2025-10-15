# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

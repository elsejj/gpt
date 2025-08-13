# v0.2.6

- 'openai' SDK upgrade to v2
- 'mcp' prompt can be used by name, eg, if some MCP have a prompt named 'p1', now can refer it like `gpt -M "mcp_url" -s p1 user_prompt`

# v0.2.5

- `openai` SDK upgrade to v1
- `mcp` support stream http transport, will be indicated by URL don't have `sse` in it.

## v0.2.4

- enable multiple round 'mcp' call

## v0.2.3

- more compatible
- support only model name

## v0.2.2

- more compatible with `mcp` server

## v0.2.1

### Fixed

- Fix `mcp` sse, it should `Start` before to use.

## v0.2.0

### Added

- Added support for `Model Content Protocol`(mcp), the `gpt` now can be passed `-M` option to specify the mcp server.

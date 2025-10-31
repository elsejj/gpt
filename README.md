# Purpose

The `gpt` is a cli tool that allows you send something to a OpenAI GPT API compatible service and get the response back.

# Usage

## simple usage

```bash
gpt 'hello, who are you?'
```

## prompt from file

```bash
gpt samples/hello.md
```

Please note there is a `@samples/json.txt` in the `samples/hello.md` file. Which will be loaded and replaced with the content of the file.

## with images

```bash
gpt -i samples/cat.jpg samples/image.md
```

`-i` flag is used to specify the image file. can be used multiple times.

## with system prompt

```bash
gpt -s samples/system.md "who are you?"
```

`-s` flag is used to specify the system prompt. in this case, it force translate the user input to chinese language instead answering the question directly.

## with mcp server

input

```bash
gpt -M "/xxxx/server.py" 'what is result of 223020320+2321?'
or
gpt -M "http://127.0.0.1:8000/sse" 'what is result of 223020320+2321?'

```

output

```
I am calculating the result of 223020320 + 2321.
2025/03/05 18:18:44 INFO Model call tool=add args="{\"a\":223020320,\"b\":2321}"
2025/03/05 18:18:44 INFO Model call result tool=add result="{Content:[{223022641 text}] Role:tool ToolCallID:call_74245828}"
The result of 223020320 + 2321 is 223022641.

```

the `server.py` is a simple mcp server script, with a tool named `add`, see [MCP Python SDK
](https://github.com/modelcontextprotocol/python-sdk) for more details.

`-M` flag is used to specify the mcp server script. If you have a mcp server running, you can use this flag to send the request to the script. Ensure that the script is executable and correctly configured to process the input.

There are two kinds of mcp server:

- Local mcp server, communicate with STDIN/STDOUT

  - Local mcp server will be started as a child process.
  - Each `-M` flag will start a new mcp server, the flag will be split by ` `(space), and the first part is the executable path, the rest is the arguments.
  - Multiple `-M` flags can be used to start multiple mcp servers. All tools will be passed to the LLM.
  - The executable can be:
    - `.py` python script, `python3` will be used to run the script, `.venv` will be used if exists.
    - `.js` javascript script, `node` will be used to run the script.
    - `.ts` typescript script, `bun` will be used to run the script.
    - `.go` go script, `go run` will be used to run the script.
    - `.sh` `.bash` `.ps1` will be run as shell script.
    - any other executable file.

- Remote mcp server, communicate with HTTP sse or HTTP stream

  - `-M` flag will be used to specify the mcp server url, and the request will be sent to the url.

- Proxy http service as mcp

  For some existing HTTP services, they can be used as MCP services by writing an MCP configuration. see [samples/qqwry.mcp.yaml](samples/qqwry.mcp.yaml), it's proxy a IP information HTTP service as MCP, eg. `gpt -M samples/qqwry.mcp.yaml "where is 120.197.169.198's location"`

## with tool

Tool is a pre-defined system prompt, model, and other configurations to do specific tasks. see [Tool](internal/tools/tools.go) for more details.

A example tool 'tr' is located at [tr.toml](samples/tools/tr.toml), which translate between chinese <-> english, you can use it like:

```bash
gpt -T samples/tools/tr.toml "Hello, how are you?"
```

This will output the translation result, `你好，你好吗？`

the `samples/tools/tr.toml` can be copied to `$HOME/.gpt/tools/tr.toml`, then you can use it like:

```bash
gpt -T tr "Hello, how are you?"
```

# Installation

```bash
go get -u github.com/elsejj/gpt
```

# Configuration

After installation, you can do first run to generate the configuration file.

```bash
gpt
```

It's show you the app version, and the configuration file path. Then you can edit the configuration file to set the API key, Gateway URL (aka Open AI Base URL) and other settings.

## LLM gateway

You can use a LLM gateway such as [Portkey-AI](https://github.com/Portkey-AI/gateway) to serve the Non-OpenAI compatible API like Gemini, Claude, etc.

I had made a fork of the Portkey-AI gateway, which is available at [llm-gateway](https://github.com/elsejj/llm-gateway/tree/keystore), enable one key to visit multiple API services.

# Integration Example

## Powershell/bash Copilot

1. copy `samples/powershell.md` / `samples/bash.md` to configuration folder
2. for powershell, create a function in your profile `$PROFILE.CurrentUserCurrentHost`:

```powershell
function pa {
    $cmd = gpt -u -s powershell.md $args
    Write-Host $cmd
    Set-Clipboard -Value $cmd.Trim()
}
```

3. for bash, add the following line to your `.bashrc`:

```bash
alias pa='gpt -u -s bash.md'
```

now you can use `pa` to generate the powershell command. for example:

- `pa list all image files by date desc` will generate `Get-ChildItem | Sort-Object LastWriteTime -Descending` and copy to clipboard.
- `pa list recent 10 files` will generate `ls -lt | head -n 10` and copy to clipboard.

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

## Powershell Copilot

1. copy 'samples/powershell.md' to configuration folder
2. create a function in your profile `$PROFILE.CurrentUserCurrentHost`:

```powershell
function pa {
    $cmd = gpt -s powershell.md $args
    Write-Host $cmd
    Set-Clipboard -Value $cmd.Trim()
}
```

now you can use `pa` to generate the powershell command. for example:

- `pa list all image files by date desc` will generate `Get-ChildItem | Sort-Object LastWriteTime -Descending` and copy to clipboard.

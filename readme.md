# Grok Chat Proxy

A reverse proxy server that adapts [Grok Chat](https://grok.com/) to comply with the OpenAI API format.

## Features

- OpenAI API compatible interface to Grok AI
- Support for both regular mode and "think" mode
- File upload support for large prompts
- Multiple browser session management for concurrent requests
- Optional API key authentication
- Streaming responses

## Requirements

You need Chrome installed on your system. The proxy uses the Chrome DevTools Protocol to communicate with the browser.

## Installation

1. Go to Github Actions and download the latest build file.
2. Unzip the file.
3. Make the file executable (if you are on Linux, Mac or Unix-like OS):
```bash
chmod +x app-linux-amd64
```

## Usage

> **IMPORTANT**: You need to create a `cookies` file in the same directory as the executable.
> This file contains the cookies for your Grok account, and every line is one account. (This relates to the number of sessions you can have.)
> You can get the cookies by logging into Grok and copying them from your browser's developer tools (F12).

Assuming the executable is named `app-windows-amd64.exe`, go to the directory where the file is located and run:

```bash
./app-windows-amd64.exe [options]
```

Here available options are:

- `-p`: Use private mode (grok chat will not save your conversations)
- `-h`: Use headless mode (browser will not be visible)
- `-i <api-key>`: Set API key for authentication
- `-port <port>`: Set the server port (default: 9867)

I suggest that you use normal mode for the first time to check if there's cloudflare protection and pass it manually. If you are not coming into any issues, you can use the headless mode.

## Limitations

- Only streaming mode is supported
- Only text responses are supported (no image generation)
- Large prompts may be slower as they require file uploads
- Not very skilled at goroutines, so file an issue if you find any bugs and willing to help

## License

[MIT License](LICENSE)
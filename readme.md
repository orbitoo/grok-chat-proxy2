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

Assuming the executable is named `app-windows-amd64.exe`, go to the directory where the file is located and run:

```bash
./app-windows-amd64.exe [options]
```

Here available options are:

- `-c`: Use cookies to log in (See [Use Cookies](#use-cookies))
- `-p`: Use private mode (grok chat will not save your conversations)
- `-h`: Use headless mode (browser will not be visible)
- `-i <api-key>`: Set API key for authentication
- `-n <number>`: Set the number of sessions you want to use and log in manually (See [Manual Login](#manual-login))
- `-port <port>`: Set the server port (default: 9867)

I suggest that you use normal mode for the first time to check if there's cloudflare protection and pass it manually. If you are not coming into any issues, you can use the headless mode.

> If you call `./app-windows-amd64.exe -c -n <number>`, it will refer to the `cookies` file and ignore the `-n` option.

### Use Cookies

To get the cookies:
1. Go to [Grok Chat](https://grok.com)
2. Press F12 to open the developer tools and switch to the "Network" tab
3. Refresh the page and look for a request named `grok.com`
4. Click on it and copy the cookies from the "Request Headers" section
5. Now, create a file named `cookies` in the same directory as the executable
6. Paste the cookies into the file, one line for one account
7. Save the file

```bash
./app-windows-amd64.exe -c
```

This will use the cookies from the `cookies` file to log in automatically and the number of sessions will be the same as the number of lines in the `cookies` file.

### Manual Login

If you don't want to use cookies, you can log in manually, and the browser will save the cookies for you.
To do this, you need to use `-n <number>` option to set the number of sessions you want to use.

For example, if you want to use 2 sessions, run the following command:

```bash
./app-windows-amd64.exe -n 2
```

Then, there will be 2 browser windows opened, and you need to log in to each of them. After logging in, the cookies will be saved to `userdata` folder automatically.

### Once you logged in (manually or using cookies)

In the future, you can start the proxy without the `-n` or `-c` option, and it will use the saved user data.

## Limitations

- Need chrome
- Only streaming mode is supported
- Only text responses are supported (no image generation)
- Large prompts may be slower as they require file uploads
- Not very skilled at goroutines, so file an issue if you find any bugs and willing to help

## License

[MIT License](LICENSE)
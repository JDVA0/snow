# How to Use the Snow LSP Server

## What is an LSP Server?

The Snow LSP Server is not a standalone executable that you run directly on files. Instead, it's a **Language Server Protocol** server that communicates with editor clients (like VS Code, Neovim, IntelliJ, etc.) through JSON-RPC over stdin/stdout.

## Correct Usage

### 1. With Editor Clients (Recommended)

The LSP server is designed to be used with editor clients that support LSP:

#### VS Code
You would need to create a VS Code extension that:
1. Starts the `snow-lsp` server as a subprocess
2. Sends LSP requests (initialize, textDocument/didOpen, etc.)
3. Receives and displays diagnostics, completions, etc.

#### Neovim
With `nvim-lspconfig`, you would configure:
```lua
require('lspconfig').snow_lsp.setup {
  cmd = { "/path/to/snow-lsp" },
  filetypes = { "snow" },
  root_dir = function() return vim.loop.cwd() end,
}
```

#### Other Editors
Similar configurations exist for:
- Emacs (with `lsp-mode`)
- Sublime Text (with LSP package)
- IntelliJ (with plugin development)

### 2. For Testing

#### Manual Testing
Run the provided example client:
```bash
cd LSP
go run client_example.go
```

This demonstrates the LSP protocol communication flow.

#### Unit Tests
Run the test suite:
```bash
cd LSP
go test ./test
```

## How It Works

1. **Editor starts the LSP server** as a subprocess
2. **Editor sends initialization request** to the server
3. **Server responds with capabilities** (what features it supports)
4. **Editor sends document events** (didOpen, didChange, didClose)
5. **Server analyzes the code** and sends back diagnostics
6. **Editor displays the diagnostics** to the user

## Example Communication Flow

```
Editor                    LSP Server
  |                           |
  |--- initialize --------->  |
  |<-- capabilities ---------|
  |                           |
  |--- textDocument/didOpen ->|
  |   (sends file content)    |
  |                           |
  |                           |
  |--- textDocument/didChange->|
  |   (sends updated content) |
  |                           |
  |<-- diagnostics -----------|
  |   (errors, warnings)      |
  |                           |
  |--- shutdown ------------>|
  |--- exit ----------------->|
```

## Current Capabilities (Phase 1)

The current implementation supports:
- Document synchronization (open, change, close)
- Basic syntax error detection
- Static analysis (undefined variables, unused variables)
- Diagnostic reporting

## Next Steps

To make this usable in practice, you would need to:

1. **Create editor extensions** for your preferred editor
2. **Implement more LSP features** (completion, hover, go-to-definition)
3. **Add language syntax highlighting** to editors
4. **Test with real projects**

## Testing with Real Files

To test the LSP server with real Snow files, use the provided client example and modify the `textDocument.content` in `client_example.go` to point to your actual files.

## Building the Server

```bash
cd LSP
go build -o snow-lsp ./server
```

The resulting `snow-lsp` binary is what editor clients would execute.

## Note on Direct Execution

Running `./snow-lsp file.snow` directly won't work because:
- LSP servers communicate via JSON-RPC, not command-line arguments
- They expect to receive structured JSON requests over stdin
- They need to run continuously as a background process

The LSP server is designed to be a **service** that editors connect to, not a **tool** that you run directly on files.
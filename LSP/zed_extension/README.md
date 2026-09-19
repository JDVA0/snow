# Snow Language Extension for Zed

This extension provides Snow language support for the Zed editor, currently focused on LSP integration.

## Important Note

Zed uses **Tree-sitter** for syntax highlighting, not TextMate grammars (.tmLanguage.json). This extension currently provides **LSP integration only**. Full syntax highlighting support would require developing a Tree-sitter grammar for Snow, which is a more complex undertaking.

## Installation

### 1. Build the LSP Server
First, build the Snow LSP server:
```bash
cd /home/julian/Escritorio/Snow/LSP
go build -o snow-lsp ./server
```

### 2. Install the Extension in Zed

#### Manual Installation
1. Copy the `zed_extension` folder to your Zed extensions directory:
   ```bash
   cp -r zed_extension ~/.config/zed/extensions/snow-lang
   ```

2. Update the LSP path in `~/.config/zed/extensions/snow-lang/extension.toml`:
   ```toml
   [language_servers.snow.binary]
   path = "/home/julian/Escritorio/Snow/LSP/snow-lsp"
   ```

3. Restart Zed

## Features

### Current Features (Phase 1)
- **LSP Integration**: Full integration with the Snow LSP server
- **Error Detection**: Real-time syntax error detection
- **Static Analysis**: Detection of undefined variables and unused variables
- **Document Synchronization**: Automatic analysis when files change

### Syntax Highlighting
Currently, Snow files will use generic text highlighting in Zed. For proper syntax highlighting, a Tree-sitter grammar would need to be developed.

### Planned Features (Future Phases)
- **Tree-sitter Grammar**: Proper syntax highlighting
- **Autocompletion**: Intelligent code completion
- **Go to Definition**: Navigate to symbol definitions
- **Hover Documentation**: Show symbol information on hover
- **Symbol Highlighting**: Highlight references to symbols
- **Code Formatting**: Automatic code formatting

## Usage

### Creating Snow Files
1. Create a new file with `.snow` extension (e.g., `app.snow`)
2. The LSP server will automatically analyze your code
3. Errors and warnings will appear in Zed's diagnostics panel

### Example Snow Code
```python
using api

api.cors()

api.get("/", fn(req): api.text("Servidor Snow activo"))

fn ver_usuario(req):
    return api.json({id: req.params.id, activo: true})

api.get("/usuarios/:id", ver_usuario)

api.serve(8080)
```

## Configuration

The extension can be configured in Zed's settings:

```json
{
  "lsp": {
    "snow-lsp": {
      "initialization_options": {},
      "settings": {}
    }
  }
}
```

## Troubleshooting

### LSP Server Not Starting
- Verify the path to `snow-lsp` is correct in `extension.toml`
- Check that `snow-lsp` has execute permissions: `chmod +x snow-lsp`
- Check Zed's developer console (Ctrl+Shift+I) for error messages

### No Diagnostics Appearing
- Verify the LSP server is running
- Check that the Snow code has valid syntax
- Look at Zed's developer console for LSP communication errors

### No Syntax Highlighting
- This is expected in the current version
- A Tree-sitter grammar needs to be developed for proper highlighting
- Snow files will use generic text highlighting

## Development

### File Structure
```
zed_extension/
├── extension.toml          # Extension configuration (LSP only)
└── README.md              # This file
```

### Tree-sitter Grammar Development
To add proper syntax highlighting, you would need to:

1. Create a Tree-sitter grammar for Snow (in Rust)
2. Build it using the WASI SDK (required by Zed)
3. Host it in a Git repository
4. Reference it in `extension.toml` like:
   ```toml
   [grammars.snow]
   repository = "https://github.com/JDVA0/tree-sitter-snow"
   rev = "commit-sha"
   ```

This is a significant undertaking and is not part of the current Phase 1 implementation.

## Contributing

This is a basic extension for the Phase 1 LSP implementation. As more LSP features are implemented (Phase 2+), the extension can be enhanced to support additional features like:
- Completion items
- Hover information
- Go to definition
- Symbol references
- Code formatting

For syntax highlighting, consider contributing to a future Tree-sitter grammar project for Snow.

## License

This extension follows the same license as the Snow project (MIT).

## Credits

- Snow Language: JDVA0
- Extension for Zed: Initial implementation (LSP integration only)
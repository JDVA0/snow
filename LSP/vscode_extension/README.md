# Snow Language Extension for VS Code

This extension provides Snow language support for VS Code, including:

- **Syntax highlighting** for Snow files
- **Language icon** (snowflake) for .snow files
- **LSP integration** with the Snow Language Server
- **Code snippets** for common Snow patterns
- **Language configuration** (indentation, brackets, etc.)

## Features

### Current Features (Phase 1)
- **Syntax highlighting**: TextMate grammar for Snow syntax
- **File icon**: Snowflake icon for .snow files
- **LSP integration**: Full integration with the Snow LSP server
- **Autocompletion**: 56 completion items (keywords, modules, functions, constants)
- **Error detection**: Real-time syntax error detection
- **Static analysis**: Detection of undefined variables and unused variables

### LSP Capabilities
- Document synchronization (open, change, close)
- Syntax error detection
- Static analysis via Snow's built-in checker
- Intelligent code completion

## Installation

### Manual Installation

1. Build the Snow LSP server:
```bash
cd /home/julian/Escritorio/Snow/LSP
go build -o snow-lsp ./server
```

2. Copy the extension folder to your VS Code extensions directory:
```bash
cp -r vscode_extension ~/.vscode/extensions/snow
```

3. Restart VS Code

### Configuration

The extension can be configured in VS Code settings:

```json
{
  "snow.lsp.path": "/home/julian/Escritorio/Snow/LSP/snow-lsp",
  "snow.lsp.enabled": true
}
```

## Usage

### Creating Snow Files
1. Create a new file with `.snow` extension (e.g., `app.snow`)
2. VS Code will automatically apply Snow syntax highlighting
3. The LSP server will start automatically
4. Autocompletion will be available

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

## Autocompletion

The LSP server provides 56 completion items:
- **26 keywords**: using, import, fn, if, for, while, etc.
- **12 standard modules**: sys, fs, api, http, db, etc.
- **15 built-in functions**: print(), len(), str(), int(), etc.
- **3 constants**: true, false, nil

## Troubleshooting

### LSP Server Not Starting
- Verify the path to `snow-lsp` is correct in settings
- Check that `snow-lsp` has execute permissions: `chmod +x snow-lsp`
- Check VS Code's developer console for error messages

### No Autocompletion
- Verify the LSP server is running
- Check that the file has `.snow` extension
- Try manual trigger with `Ctrl+Space`
- Check VS Code's Output panel for "Snow LSP" channel

### No Syntax Highlighting
- Ensure the file has `.snow` extension
- Reload VS Code after installing the extension
- Check that the grammar file is in the correct location

## Development

### File Structure
```
vscode_extension/
├── package.json              # Extension manifest
├── extension.ts              # Main extension code
├── tsconfig.json            # TypeScript configuration
├── language-configuration.json # Language configuration
├── icons/
│   └── snowflake.svg         # Language icon
├── syntaxes/
│   └── snow.tmLanguage.json  # Syntax highlighting grammar
└── README.md                # This file
```

### Building the Extension

The extension needs to be compiled from TypeScript to JavaScript:

```bash
cd vscode_extension
npm install -g typescript
tsc
```

## Future Enhancements

As more LSP features are implemented (Phase 2+), the extension can be enhanced to support:
- Go to definition
- Hover documentation
- Symbol references
- Code formatting
- And more LSP features

## License

This extension follows the same license as the Snow project (MIT).

## Credits

- Snow Language: JDVA0
- Extension for VS Code: Initial implementation
# Blizzard Language for VS Code

This extension adds Blizzard syntax highlighting, the Blizzard file icon, and a
bundled LSP client for completion and diagnostics in `.blizz` files.

## Features

- TextMate syntax highlighting and Blizzard language configuration.
- Blizzard icon association for `.blizz` files.
- Diagnostics for parser and static-analysis errors.
- Completion for keywords, built-ins, constants, modules, and module members.
- `http.` completion for `get`, `post`, `put`, `delete`, `patch`, and `request`.
- String-method completion after a string value, for example `"blizzard".upper()`.
- Highlighting for spread/rest syntax: `...items` and `fn collect(...rest):`.

## Build and install

Build the LSP binary first:

```bash
cd ../
go build -o blizzard-lsp ./server
```

Then build and install the VSIX:

```bash
cd vscode_extension
pnpm install
pnpm run compile
npx @vscode/vsce package --no-dependencies
code --install-extension blizzard-0.2.0.vsix --force
```

The final package bundles `vscode-languageclient` and does not include
`node_modules`, source maps, or shell scripts. `pnpm audit --prod` should be
run before publishing a new package.

## Configuration

```json
{
  "blizzard.lsp.enabled": true,
  "blizzard.lsp.path": ""
}
```

An empty path uses `LSP/blizzard-lsp` relative to the first workspace folder. Set an
absolute path when the binary is stored elsewhere. The extension verifies that
the configured path is a regular file before starting the language server.

## Troubleshooting

### LSP Server Not Starting

- Verify the path to `blizzard-lsp` is correct in settings.
- Check that `blizzard-lsp` has execute permissions: `chmod +x blizzard-lsp`.
- Check VS Code's developer console for error messages.

### No Autocompletion

- Verify the LSP server is running.
- Check that the file has a `.blizz` extension.
- Try manual trigger with `Ctrl+Space`.
- Check VS Code's Output panel for the `Blizzard LSP` channel.

If completion shows only generic items, the extension is usually launching an
old LSP binary. Rebuild `LSP/blizzard-lsp`, reinstall the VSIX, and reload VS Code.

### No Syntax Highlighting

- Ensure the file has a `.blizz` extension.
- Reload VS Code after installing the extension.
- Check that the grammar file is in the correct location.

## Development

```bash
pnpm run compile
cd ..
go test ./analyzer ./protocol ./server ./test
```

The extension entrypoint is `extension.ts`; the generated runtime bundle is
`out/extension.js`. Do not commit `node_modules`, VSIX files, source maps, or
local test scripts.

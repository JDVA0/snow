# Snow LSP

The Snow Language Server uses standard JSON-RPC over stdin/stdout with
`Content-Length` framing. It is used by the VS Code extension and can also be
connected to other editors that support custom language servers.

## Current support

- Full document synchronization for open, change, and close events.
- Syntax diagnostics from the Snow parser.
- Static diagnostics from `snow.Check`, including undefined and unused names,
  unreachable code, and unknown modules.
- Completion for Snow keywords, built-ins, constants, standard modules, and
  module members.
- Module member completion after a dot, for example `http.` suggests `get`,
  `post`, `put`, `delete`, `patch`, and `request`.
- Trigger characters: `.`, `:`, and space.

The language features covered by the current parser, compiler, and tests also
include `const` immutable bindings, `always` cleanup blocks, ternary
expressions (`value if condition else other_value`), `enumerate()` and `zip()`.
The package manager provides `snowman update` for refreshing locked packages.

The server currently does not provide hover, go-to-definition, references,
formatting, or rename support.

## Build

```bash
cd LSP
go build -o snow-lsp ./server
```

The resulting `LSP/snow-lsp` must be executable. The server is an editor
service; it does not accept a `.snow` filename as a command-line argument.

## Tests

Run the LSP packages that form the server and its tests:

```bash
cd LSP
go test ./analyzer ./protocol ./server ./test
```

The repository root contains the language implementation tests:

```bash
go test ./...
```

The server completion regression test covers `http.` member completion in
`server/completion_test.go`.

## VS Code extension

The extension lives in `LSP/vscode_extension`. Build and package it with pnpm:

```bash
cd LSP/vscode_extension
pnpm install
pnpm run compile
npx @vscode/vsce package --no-dependencies
code --install-extension snow-0.1.0.vsix --force
```

The package bundles `vscode-languageclient`; `node_modules`, source maps, and
development metadata are excluded from the VSIX. The extension contributes the
Snow language, TextMate grammar, and Snowflake file icon.

By default the extension looks for `LSP/snow-lsp` in the opened workspace. A
custom executable can be configured with:

```json
{
  "snow.lsp.enabled": true,
  "snow.lsp.path": "/absolute/path/to/snow-lsp"
}
```

Leave `snow.lsp.path` empty to use the workspace-relative default. After
installing or rebuilding, use `Developer: Reload Window` in VS Code.

## Manual protocol check

Requests must use standard LSP framing:

```text
Content-Length: <number of UTF-8 bytes>

<JSON-RPC payload>
```

The server writes logs to stderr and protocol messages to stdout, so editor
clients can safely consume stdout.

## Project layout

```text
LSP/
├── analyzer/           # Parser and static diagnostics
├── protocol/           # LSP payload types
├── server/             # JSON-RPC server and completion logic
├── test/               # Analyzer and protocol tests
└── vscode_extension/   # VS Code client, grammar, and icon
```

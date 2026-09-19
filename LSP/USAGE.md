# Using the Snow LSP

The Snow LSP is a long-running editor service. An editor starts `snow-lsp` as
a subprocess and exchanges JSON-RPC messages over stdin/stdout using standard
`Content-Length` framing.

## Recommended setup: VS Code

From the repository root:

```bash
cd LSP
go build -o snow-lsp ./server

cd vscode_extension
pnpm install
pnpm run compile
npx @vscode/vsce package --no-dependencies
code --install-extension snow-0.1.0.vsix --force
```

Open a `.snow` file and reload VS Code if the extension was already installed.
The extension starts `LSP/snow-lsp` relative to the workspace by default. For
another location, set `snow.lsp.path` to an executable file. Set
`snow.lsp.enabled` to `false` to disable the client.

## Completion

Completion is requested automatically after `.`, `:`, or a space. For example:

```snow
using http

response = http.
```

Typing `http.` offers `get`, `post`, `put`, `delete`, `patch`, and `request`.
The server also offers members for the other built-in modules, plus language
keywords, built-ins, and constants.

## Diagnostics

Diagnostics are published after `didOpen` and `didChange`. They include parser
errors and static checker findings such as undefined names, unused variables,
unreachable code, and unknown modules.

## Build and test the server

```bash
cd LSP
go test ./analyzer ./protocol ./server ./test
go build -o snow-lsp ./server
```

The root project can be tested with:

```bash
go test ./...
```

Do not run `./snow-lsp file.snow`: LSP servers receive document text from the
editor, not source filenames as positional arguments.

## Other editors

Editors that support custom LSP commands can launch the binary with no
arguments, use language/file type `snow`, and send full document synchronization
(`openClose: true`, `change: 1`). The server currently supports completion and
diagnostics; hover, navigation, formatting, and rename are not implemented.

import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import {
    LanguageClient,
    LanguageClientOptions,
    ServerOptions,
} from 'vscode-languageclient/node';

let lspClient: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext) {
    console.log('Blizzard extension is now active!');

    startLSP(context);
}

function startLSP(context: vscode.ExtensionContext) {
    const config = vscode.workspace.getConfiguration('blizzard');
    const lspEnabled = config.get<boolean>('lsp.enabled', true);

    if (!lspEnabled) {
        console.log('Blizzard LSP is disabled in settings');
        return;
    }

    const configuredPath = config.get<string>('lsp.path', '').trim();
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    const lspPath = configuredPath || (workspaceRoot ? path.join(workspaceRoot, 'LSP', 'blizzard-lsp') : '');

    if (!lspPath || !fs.existsSync(lspPath)) {
        console.error(`Blizzard LSP server not found at: ${lspPath}`);
        vscode.window.showErrorMessage('Blizzard LSP server not found. Build LSP/blizzard-lsp or configure blizzard.lsp.path.');
        return;
    }

    const serverStat = fs.statSync(lspPath);
    if (!serverStat.isFile()) {
        vscode.window.showErrorMessage('Blizzard LSP path must point to an executable file.');
        return;
    }

    const serverOptions: ServerOptions = {
        command: lspPath,
        args: [],
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [{ scheme: 'file', language: 'blizzard' }],
        synchronize: {
            configurationSection: 'blizzard',
            fileEvents: [
                vscode.workspace.createFileSystemWatcher('**/*.blizz'),
            ],
        },
    };

    const client = new LanguageClient(
        'blizzard-lsp',
        'Blizzard LSP',
        serverOptions,
        clientOptions,
    );

    context.subscriptions.push(client);
    client.start();
    lspClient = client;
    console.log('Blizzard LSP started');
}

export function deactivate(): Thenable<void> | undefined {
    if (lspClient) {
        return lspClient.stop();
    }
    return undefined;
}
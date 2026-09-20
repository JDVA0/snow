const vscode = require('vscode');
const path = require('path');
const fs = require('fs');

function activate(context) {
    console.log('Blizzard extension is now active!');

    // Register Blizzard language
    const blizzardLanguageConfig = vscode.languages.registerLanguageConfiguration('blizzard', {
        comments: {
            lineComment: '#'
        },
        brackets: [
            ['{', '}'],
            ['[', ']'],
            ['(', ')']
        ],
        autoClosingPairs: [
            ['{', '}'],
            ['[', ']'],
            ['(', ')'],
            ['"', '"'],
            ["'", "'"]
        ],
        surroundingPairs: [
            ['{', '}'],
            ['[', ']'],
            ['(', ')'],
            ['"', '"'],
            ["'", "'"]
        ]
    });
    context.subscriptions.push(blizzardLanguageConfig);

    console.log('Blizzard language configuration registered');
}

function deactivate() {
    console.log('Blizzard extension deactivated');
}

module.exports = {
    activate,
    deactivate
};
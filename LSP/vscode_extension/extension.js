const vscode = require('vscode');
const path = require('path');
const fs = require('fs');

function activate(context) {
    console.log('Snow extension is now active!');

    // Register Snow language
    const snowLanguageConfig = vscode.languages.registerLanguageConfiguration('snow', {
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
    context.subscriptions.push(snowLanguageConfig);

    console.log('Snow language configuration registered');
}

function deactivate() {
    console.log('Snow extension deactivated');
}

module.exports = {
    activate,
    deactivate
};
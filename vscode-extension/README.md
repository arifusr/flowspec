# FlowSpec Language — VS Code Extension

Syntax highlighting, Go to Definition, and Hover preview for FlowSpec DSL (`.flow`) files.

## Features

### Syntax Highlighting

Full colorization for FlowSpec keywords, HTTP methods, strings, variables, annotations, and operators.

### Go to Definition (Ctrl+Click / F12)

Jump from usage to definition:

| Pattern | Jumps to |
|---------|----------|
| `run RequestName` | `request RequestName {` definition |
| `use auth AuthName` | `auth AuthName {` definition |
| `use fragment FragName` | `fragment FragName {` definition |
| `import path/to/file.flow` | The referenced file |
| `body from file "path.json"` | The JSON file |
| `include flows/other.flow` | The flow file |

### Hover Preview

Hover over `run RequestName` to see:

```
request RequestName
  POST http://localhost:8080/api/endpoint
  @tags(crud, smoke)

📁 requests/endpoint.flow:3
```

## Installation

### From VSIX (recommended)

```bash
cd vscode-extension
npm install
npm run compile
npx vsce package
code --install-extension flowspec-language-0.1.0.vsix
```

### Development mode

```bash
cd vscode-extension
npm install
npm run compile
# Press F5 in VS Code to launch Extension Development Host
```

## Requirements

- VS Code 1.80+
- Workspace must contain `.flow` files

## File Association

The extension auto-activates for files with `.flow` extension.

## Documentation

Full FlowSpec DSL documentation: https://github.com/arifusr/flowspec

import * as vscode from 'vscode';
import { FlowSpecIndex, Definition } from './indexer';

let index: FlowSpecIndex;

export function activate(context: vscode.ExtensionContext) {
  index = new FlowSpecIndex();
  index.buildIndex();

  const outputChannel = vscode.window.createOutputChannel('FlowSpec');
  outputChannel.appendLine(`FlowSpec extension activated. Indexed ${index.size()} definitions.`);

  // Watch for file changes
  const watcher = vscode.workspace.createFileSystemWatcher('**/*.flow');
  watcher.onDidChange(() => { index.buildIndex(); outputChannel.appendLine(`Re-indexed: ${index.size()} definitions`); });
  watcher.onDidCreate(() => { index.buildIndex(); outputChannel.appendLine(`Re-indexed: ${index.size()} definitions`); });
  watcher.onDidDelete(() => { index.buildIndex(); outputChannel.appendLine(`Re-indexed: ${index.size()} definitions`); });
  context.subscriptions.push(watcher);

  // Go to Definition — register for flowspec language AND all .flow files by glob
  const selector: vscode.DocumentSelector = [
    { language: 'flowspec', scheme: 'file' },
    { pattern: '**/*.flow', scheme: 'file' },
    { language: 'plaintext', pattern: '**/*.flow', scheme: 'file' }
  ];

  const definitionProvider = vscode.languages.registerDefinitionProvider(
    selector,
    new FlowSpecDefinitionProvider(outputChannel)
  );
  context.subscriptions.push(definitionProvider);

  // Hover
  const hoverProvider = vscode.languages.registerHoverProvider(
    selector,
    new FlowSpecHoverProvider()
  );
  context.subscriptions.push(hoverProvider);

  // Reference Provider — find all usages of a request
  const referenceProvider = vscode.languages.registerReferenceProvider(
    selector,
    new FlowSpecReferenceProvider()
  );
  context.subscriptions.push(referenceProvider);
}

export function deactivate() {}

class FlowSpecDefinitionProvider implements vscode.DefinitionProvider {
  private output: vscode.OutputChannel;
  constructor(output: vscode.OutputChannel) { this.output = output; }

  provideDefinition(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.ProviderResult<vscode.Definition> {
    const line = document.lineAt(position).text;
    const target = getTargetAtPosition(line, position.character);

    if (!target) {
      return null;
    }

    this.output.appendLine(`Go to def: "${target.name}" (${target.type})`);

    // Check for file path references (import, body from file/schema, include)
    const fileRef = getFileReference(line);
    if (fileRef) {
      const resolved = resolveFilePath(document.uri, fileRef);
      if (resolved) {
        this.output.appendLine(`  → file: ${resolved}`);
        return new vscode.Location(vscode.Uri.file(resolved), new vscode.Position(0, 0));
      }
    }

    // If we're ON a definition (request/auth/fragment), show all usages instead
    const defMatch = line.match(/^\s*(request|auth|fragment)\s+(\w+)/);
    if (defMatch && defMatch[2] === target.name) {
      this.output.appendLine(`  → At definition site, finding usages...`);
      const usages = findUsages(target.name);
      this.output.appendLine(`  → Found ${usages.length} usage(s)`);
      if (usages.length > 0) {
        return usages;
      }
      return null;
    }

    // Normal case: at usage site (run X), go to definition
    const def = index.lookup(target.name, target.type);
    if (def) {
      this.output.appendLine(`  → ${def.file}:${def.line + 1}`);
      return new vscode.Location(
        vscode.Uri.file(def.file),
        new vscode.Position(def.line, 0)
      );
    }

    this.output.appendLine(`  → NOT FOUND in index`);
    return null;
  }
}

class FlowSpecHoverProvider implements vscode.HoverProvider {
  provideHover(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.ProviderResult<vscode.Hover> {
    const line = document.lineAt(position).text;
    const target = getTargetAtPosition(line, position.character);

    if (!target) return null;

    const def = index.lookup(target.name, target.type);
    if (!def) return null;

    const md = new vscode.MarkdownString();
    md.appendCodeblock(formatDefinitionPreview(def), 'flowspec');
    md.appendMarkdown(`\n\n📁 *${vscode.workspace.asRelativePath(def.file)}:${def.line + 1}*`);

    return new vscode.Hover(md);
  }
}

interface TargetRef {
  name: string;
  type: 'request' | 'auth' | 'fragment';
}

function getTargetAtPosition(line: string, character: number): TargetRef | null {
  // run RequestName or run RequestName(...) — anywhere in the line
  const runMatch = line.match(/\brun\s+(\w+)/);
  if (runMatch) {
    const name = runMatch[1];
    const nameStart = line.indexOf(runMatch[0]) + 4; // after "run "
    const nameEnd = nameStart + name.length;
    // Accept if cursor is anywhere on the line that contains `run X`
    // (makes Ctrl+Click work on the request name OR on `run` keyword)
    if (character >= line.indexOf(runMatch[0]) && character <= nameEnd) {
      return { name, type: 'request' };
    }
    // Also match if cursor is exactly on the name
    if (character >= nameStart && character <= nameEnd) {
      return { name, type: 'request' };
    }
    // Fallback: if line has `run X`, and user clicked anywhere reasonable
    return { name, type: 'request' };
  }

  // use auth AuthName — anywhere in line
  const authMatch = line.match(/\buse\s+auth\s+(\w+)/);
  if (authMatch) {
    return { name: authMatch[1], type: 'auth' };
  }

  // use fragment FragName — anywhere in line
  const fragMatch = line.match(/\buse\s+fragment\s+(\w+)/);
  if (fragMatch) {
    return { name: fragMatch[1], type: 'fragment' };
  }

  // Also support clicking on the request/auth/fragment definition name itself
  const defMatch = line.match(/^\s*(request|auth|fragment)\s+(\w+)/);
  if (defMatch) {
    const kind = defMatch[1] as 'request' | 'auth' | 'fragment';
    return { name: defMatch[2], type: kind };
  }

  return null;
}

function getFileReference(line: string): string | null {
  // import path/to/file.flow
  const importMatch = line.match(/^\s*import\s+(.+\.flow)\s*$/);
  if (importMatch) return importMatch[1].trim();

  // body from file "path" or body from schema "path"
  const bodyMatch = line.match(/body\s+from\s+(?:file|schema)\s+"([^"]+)"/);
  if (bodyMatch) return bodyMatch[1];

  // include path/to/flow.flow
  const includeMatch = line.match(/^\s*include\s+(.+\.flow)\s*$/);
  if (includeMatch) return includeMatch[1].trim();

  return null;
}

function resolveFilePath(currentUri: vscode.Uri, relativePath: string): string | null {
  const fs = require('fs');
  const path = require('path');

  // Try relative to workspace root
  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (workspaceFolders) {
    for (const folder of workspaceFolders) {
      const resolved = path.join(folder.uri.fsPath, relativePath);
      if (fs.existsSync(resolved)) return resolved;
    }
  }

  // Try relative to current file
  const dir = path.dirname(currentUri.fsPath);
  const resolved = path.join(dir, relativePath);
  if (fs.existsSync(resolved)) return resolved;

  return null;
}

function formatDefinitionPreview(def: Definition): string {
  let preview = `${def.kind} ${def.name}`;
  if (def.method) preview += `\n  ${def.method} ${def.url || ''}`;
  if (def.tags && def.tags.length > 0) preview += `\n  @tags(${def.tags.join(', ')})`;
  return preview;
}

function findUsages(name: string): vscode.Location[] {
  const locations: vscode.Location[] = [];
  const fs = require('fs');
  const path = require('path');

  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (!workspaceFolders) return locations;

  const searchDir = (dir: string) => {
    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          if (entry.name.startsWith('.') || entry.name === 'node_modules' ||
              entry.name === 'reports' || entry.name === 'build') continue;
          searchDir(fullPath);
        } else if (entry.name.endsWith('.flow')) {
          const content = fs.readFileSync(fullPath, 'utf-8');
          const lines = content.split('\n');
          for (let i = 0; i < lines.length; i++) {
            // Match usage: `run Name`, `use auth Name`, `use fragment Name`
            const regex = new RegExp(`\\b(run\\s+${name}|use\\s+auth\\s+${name}|use\\s+fragment\\s+${name})\\b`);
            if (regex.test(lines[i])) {
              const col = lines[i].indexOf(name);
              locations.push(new vscode.Location(
                vscode.Uri.file(fullPath),
                new vscode.Position(i, col >= 0 ? col : 0)
              ));
            }
          }
        }
      }
    } catch { /* ignore */ }
  };

  for (const folder of workspaceFolders) {
    searchDir(folder.uri.fsPath);
  }

  return locations;
}

class FlowSpecReferenceProvider implements vscode.ReferenceProvider {
  provideReferences(
    document: vscode.TextDocument,
    position: vscode.Position,
    _context: vscode.ReferenceContext
  ): vscode.ProviderResult<vscode.Location[]> {
    const line = document.lineAt(position).text;

    // Get the name at cursor — works on definition site or usage site
    let name: string | null = null;

    const defMatch = line.match(/^\s*(request|auth|fragment|flow)\s+(\w+)/);
    if (defMatch) {
      name = defMatch[2];
    }

    const runMatch = line.match(/\brun\s+(\w+)/);
    if (!name && runMatch) {
      name = runMatch[1];
    }

    if (!name) {
      // Try word at cursor
      const wordRange = document.getWordRangeAtPosition(position);
      if (wordRange) {
        name = document.getText(wordRange);
      }
    }

    if (!name) return null;

    // Search all .flow files for references to this name
    const locations: vscode.Location[] = [];
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders) return locations;

    const fs = require('fs');
    const path = require('path');

    const searchDir = (dir: string) => {
      try {
        const entries = fs.readdirSync(dir, { withFileTypes: true });
        for (const entry of entries) {
          const fullPath = path.join(dir, entry.name);
          if (entry.isDirectory()) {
            if (entry.name.startsWith('.') || entry.name === 'node_modules' ||
                entry.name === 'reports' || entry.name === 'build') continue;
            searchDir(fullPath);
          } else if (entry.name.endsWith('.flow')) {
            const content = fs.readFileSync(fullPath, 'utf-8');
            const lines = content.split('\n');
            for (let i = 0; i < lines.length; i++) {
              // Match `run Name`, `use auth Name`, `use fragment Name`, or definition
              const regex = new RegExp(`\\b(run\\s+${name}|use\\s+auth\\s+${name}|use\\s+fragment\\s+${name}|request\\s+${name}|auth\\s+${name}|fragment\\s+${name})\\b`);
              if (regex.test(lines[i])) {
                locations.push(new vscode.Location(
                  vscode.Uri.file(fullPath),
                  new vscode.Position(i, lines[i].indexOf(name!))
                ));
              }
            }
          }
        }
      } catch { /* ignore */ }
    };

    for (const folder of workspaceFolders) {
      searchDir(folder.uri.fsPath);
    }

    return locations;
  }
}

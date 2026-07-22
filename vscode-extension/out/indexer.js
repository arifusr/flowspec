"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FlowSpecIndex = void 0;
const vscode = require("vscode");
const fs = require("fs");
const path = require("path");
class FlowSpecIndex {
    constructor() {
        this.definitions = new Map();
    }
    buildIndex() {
        this.definitions.clear();
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders)
            return;
        for (const folder of workspaceFolders) {
            this.scanDirectory(folder.uri.fsPath);
        }
    }
    size() {
        return this.definitions.size;
    }
    lookup(name, kind) {
        // Try exact match with kind prefix
        if (kind) {
            const key = `${kind}:${name}`;
            if (this.definitions.has(key))
                return this.definitions.get(key);
        }
        // Try request (most common)
        const reqKey = `request:${name}`;
        if (this.definitions.has(reqKey))
            return this.definitions.get(reqKey);
        // Try auth
        const authKey = `auth:${name}`;
        if (this.definitions.has(authKey))
            return this.definitions.get(authKey);
        // Try fragment
        const fragKey = `fragment:${name}`;
        if (this.definitions.has(fragKey))
            return this.definitions.get(fragKey);
        return undefined;
    }
    scanDirectory(dir) {
        try {
            const entries = fs.readdirSync(dir, { withFileTypes: true });
            for (const entry of entries) {
                const fullPath = path.join(dir, entry.name);
                if (entry.isDirectory()) {
                    // Skip hidden dirs, node_modules, reports, build
                    if (entry.name.startsWith('.') || entry.name === 'node_modules' ||
                        entry.name === 'reports' || entry.name === 'build') {
                        continue;
                    }
                    this.scanDirectory(fullPath);
                }
                else if (entry.name.endsWith('.flow')) {
                    this.parseFile(fullPath);
                }
            }
        }
        catch {
            // ignore permission errors
        }
    }
    parseFile(filePath) {
        try {
            const content = fs.readFileSync(filePath, 'utf-8');
            const lines = content.split('\n');
            let currentTags = [];
            for (let i = 0; i < lines.length; i++) {
                const line = lines[i].trim();
                // Collect tags
                const tagMatch = line.match(/@tags\(([^)]+)\)/);
                if (tagMatch) {
                    currentTags = tagMatch[1].split(',').map(t => t.trim());
                    continue;
                }
                // request Name or request Name(params)
                const reqMatch = line.match(/^request\s+(\w+)/);
                if (reqMatch) {
                    const def = {
                        name: reqMatch[1],
                        kind: 'request',
                        file: filePath,
                        line: i,
                        tags: [...currentTags],
                    };
                    // Look for method + URL on next lines
                    for (let j = i + 1; j < Math.min(i + 10, lines.length); j++) {
                        const methodMatch = lines[j].trim().match(/^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+"([^"]+)"/);
                        if (methodMatch) {
                            def.method = methodMatch[1];
                            def.url = methodMatch[2];
                            break;
                        }
                    }
                    this.definitions.set(`request:${def.name}`, def);
                    currentTags = [];
                    continue;
                }
                // auth Name
                const authMatch = line.match(/^auth\s+(\w+)/);
                if (authMatch) {
                    this.definitions.set(`auth:${authMatch[1]}`, {
                        name: authMatch[1],
                        kind: 'auth',
                        file: filePath,
                        line: i,
                        tags: [...currentTags],
                    });
                    currentTags = [];
                    continue;
                }
                // fragment Name
                const fragMatch = line.match(/^fragment\s+(\w+)/);
                if (fragMatch) {
                    this.definitions.set(`fragment:${fragMatch[1]}`, {
                        name: fragMatch[1],
                        kind: 'fragment',
                        file: filePath,
                        line: i,
                        tags: [...currentTags],
                    });
                    currentTags = [];
                    continue;
                }
                // flow Name
                const flowMatch = line.match(/^flow\s+(\w+)/);
                if (flowMatch) {
                    this.definitions.set(`flow:${flowMatch[1]}`, {
                        name: flowMatch[1],
                        kind: 'flow',
                        file: filePath,
                        line: i,
                        tags: [...currentTags],
                    });
                    currentTags = [];
                    continue;
                }
            }
        }
        catch {
            // ignore read errors
        }
    }
}
exports.FlowSpecIndex = FlowSpecIndex;
//# sourceMappingURL=indexer.js.map
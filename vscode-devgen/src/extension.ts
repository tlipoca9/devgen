import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';

// Import generated tools configuration
import toolsConfigData from './tools-config.json';

// Annotation pattern: toolname:@annotation or toolname:@annotation(params)
const ANNOTATION_PATTERN = /(\w+):@([\w.]+)(?:\(([^)]*)\))?/g;

// Types from tools-config.json
interface AnnotationMeta {
    doc: string;
    paramType?: string | string[]; // 'string' | 'number' | 'list' | 'enum' | 'bool' or array of types
    placeholder?: string;
    values?: string[];
    valueDocs?: { [key: string]: string };
    maxArgs?: number; // Maximum number of arguments allowed (for enum types)
}

interface ToolConfig {
    typeAnnotations: string[];
    fieldAnnotations: string[];
    outputSuffix: string;
    annotations: { [name: string]: AnnotationMeta };
}

interface ToolsConfig {
    [toolName: string]: ToolConfig;
}

const toolsConfig: ToolsConfig = toolsConfigData as ToolsConfig;

interface ParsedAnnotation {
    tool: string;
    name: string;
    params: string | undefined;
    range: vscode.Range;
    fullMatch: string;
}

interface TypeInfo {
    name: string;
    range: vscode.Range;
    annotations: ParsedAnnotation[];
    fields: FieldInfo[];
}

interface FieldInfo {
    name: string;
    range: vscode.Range;
    annotations: ParsedAnnotation[];
}

let diagnosticCollection: vscode.DiagnosticCollection;

export function activate(context: vscode.ExtensionContext) {
    console.log('DevGen extension activated');

    diagnosticCollection = vscode.languages.createDiagnosticCollection('devgen');
    context.subscriptions.push(diagnosticCollection);

    // Register completion provider
    const completionProvider = vscode.languages.registerCompletionItemProvider(
        'go',
        new DevGenCompletionProvider(),
        ':', '@', '(', ','
    );
    context.subscriptions.push(completionProvider);

    // Register hover provider
    const hoverProvider = vscode.languages.registerHoverProvider(
        'go',
        new DevGenHoverProvider()
    );
    context.subscriptions.push(hoverProvider);

    // Update diagnostics on document change
    context.subscriptions.push(
        vscode.workspace.onDidChangeTextDocument(e => {
            if (e.document.languageId === 'go') {
                updateDiagnostics(e.document);
            }
        })
    );

    // Update diagnostics on document open
    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument(doc => {
            if (doc.languageId === 'go') {
                updateDiagnostics(doc);
            }
        })
    );

    // Update diagnostics for all open Go files
    vscode.workspace.textDocuments.forEach(doc => {
        if (doc.languageId === 'go') {
            updateDiagnostics(doc);
        }
    });
}

export function deactivate() {
    if (diagnosticCollection) {
        diagnosticCollection.dispose();
    }
}

function isDiagnosticsEnabled(): boolean {
    const config = vscode.workspace.getConfiguration('devgen');
    return config.get<boolean>('enableDiagnostics') ?? true;
}

// Validate parameter value against allowed types
function validateParamValue(value: string, types: string[]): boolean {
    for (const type of types) {
        switch (type) {
            case 'number':
                if (/^-?\d+(\.\d+)?$/.test(value)) return true;
                break;
            case 'bool':
                if (value === 'true' || value === 'false') return true;
                break;
            case 'string':
            case 'list':
                // string and list accept any non-empty value
                return true;
        }
    }
    return false;
}

function parseAnnotations(document: vscode.TextDocument): ParsedAnnotation[] {
    const annotations: ParsedAnnotation[] = [];
    const text = document.getText();

    // Only match annotations at the beginning of comments (after // or /* and optional whitespace)
    const commentPattern = /\/\/\s*(\w+:@[\w.]+(?:\([^)]*\))?)|\/\*\s*(\w+:@[\w.]+(?:\([^)]*\))?)/g;
    let commentMatch;

    while ((commentMatch = commentPattern.exec(text)) !== null) {
        const annotationText = commentMatch[1] || commentMatch[2];
        if (!annotationText) continue;

        const annMatch = annotationText.match(/(\w+):@([\w.]+)(?:\(([^)]*)\))?/);
        if (!annMatch) continue;

        const absoluteStart = commentMatch.index + commentMatch[0].indexOf(annotationText);
        const startPos = document.positionAt(absoluteStart);
        const endPos = document.positionAt(absoluteStart + annotationText.length);

        annotations.push({
            tool: annMatch[1],
            name: annMatch[2],
            params: annMatch[3],
            range: new vscode.Range(startPos, endPos),
            fullMatch: annMatch[0]
        });
    }

    return annotations;
}

function parseTypes(document: vscode.TextDocument): TypeInfo[] {
    const types: TypeInfo[] = [];
    const text = document.getText();
    const lines = text.split('\n');

    const typePattern = /^type\s+(\w+)\s+struct\s*\{/;
    let currentType: TypeInfo | null = null;
    let braceCount = 0;
    let docCommentStart = -1;
    let docCommentLines: string[] = [];

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        const trimmed = line.trim();

        if (trimmed.startsWith('//')) {
            if (docCommentStart === -1) {
                docCommentStart = i;
                docCommentLines = [];
            }
            docCommentLines.push(trimmed);
        } else if (trimmed === '' && docCommentStart !== -1) {
            docCommentStart = -1;
            docCommentLines = [];
        } else {
            const typeMatch = line.match(typePattern);
            if (typeMatch) {
                const typeName = typeMatch[1];
                const typeAnnotations = parseAnnotationsFromLines(document, docCommentLines, docCommentStart);

                currentType = {
                    name: typeName,
                    range: new vscode.Range(i, 0, i, line.length),
                    annotations: typeAnnotations,
                    fields: []
                };
                braceCount = 1;
            }

            if (!trimmed.startsWith('//')) {
                docCommentStart = -1;
                docCommentLines = [];
            }
        }

        if (currentType && braceCount > 0) {
            for (const char of line) {
                if (char === '{') braceCount++;
                if (char === '}') braceCount--;
            }

            if (braceCount > 0 && !line.match(typePattern)) {
                const fieldMatch = line.match(/^\s+(\w+)\s+\S+/);
                if (fieldMatch) {
                    const fieldAnnotations: ParsedAnnotation[] = [];

                    let j = i - 1;
                    while (j >= 0 && lines[j].trim().startsWith('//')) {
                        const commentLine = lines[j].trim();
                        ANNOTATION_PATTERN.lastIndex = 0;
                        let annMatch;
                        while ((annMatch = ANNOTATION_PATTERN.exec(commentLine)) !== null) {
                            const lineStart = document.offsetAt(new vscode.Position(j, 0));
                            const commentStart = lines[j].indexOf('//');
                            const absoluteStart = lineStart + commentStart + annMatch.index;
                            const startPos = document.positionAt(absoluteStart);
                            const endPos = document.positionAt(absoluteStart + annMatch[0].length);

                            fieldAnnotations.push({
                                tool: annMatch[1],
                                name: annMatch[2],
                                params: annMatch[3],
                                range: new vscode.Range(startPos, endPos),
                                fullMatch: annMatch[0]
                            });
                        }
                        j--;
                    }

                    if (fieldAnnotations.length > 0) {
                        currentType.fields.push({
                            name: fieldMatch[1],
                            range: new vscode.Range(i, 0, i, line.length),
                            annotations: fieldAnnotations
                        });
                    }
                }
            }

            if (braceCount === 0) {
                types.push(currentType);
                currentType = null;
            }
        }
    }

    return types;
}

function parseAnnotationsFromLines(
    document: vscode.TextDocument,
    lines: string[],
    startLine: number
): ParsedAnnotation[] {
    const annotations: ParsedAnnotation[] = [];

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];
        ANNOTATION_PATTERN.lastIndex = 0;
        let match;

        while ((match = ANNOTATION_PATTERN.exec(line)) !== null) {
            const lineNum = startLine + i;
            const lineText = document.lineAt(lineNum).text;
            const commentStart = lineText.indexOf('//');
            const absoluteStart = commentStart + match.index;

            annotations.push({
                tool: match[1],
                name: match[2],
                params: match[3],
                range: new vscode.Range(
                    lineNum, absoluteStart,
                    lineNum, absoluteStart + match[0].length
                ),
                fullMatch: match[0]
            });
        }
    }

    return annotations;
}

function updateDiagnostics(document: vscode.TextDocument) {
    if (!isDiagnosticsEnabled()) {
        diagnosticCollection.delete(document.uri);
        return;
    }

    // Skip test files
    if (document.fileName.endsWith('_test.go')) {
        diagnosticCollection.delete(document.uri);
        return;
    }

    const diagnostics: vscode.Diagnostic[] = [];
    const types = parseTypes(document);
    const allAnnotations = parseAnnotations(document);

    // Check for unknown tools and annotations, and validate parameters
    for (const ann of allAnnotations) {
        const toolConfig = toolsConfig[ann.tool];

        if (!toolConfig) {
            continue;
        }

        const allKnownAnnotations = [
            ...toolConfig.typeAnnotations,
            ...toolConfig.fieldAnnotations
        ];

        if (!allKnownAnnotations.includes(ann.name)) {
            diagnostics.push(new vscode.Diagnostic(
                ann.range,
                `Unknown annotation '${ann.name}' for tool '${ann.tool}'. Known annotations: ${allKnownAnnotations.join(', ')}`,
                vscode.DiagnosticSeverity.Warning
            ));
            continue;
        }

        // Validate annotation parameters
        const annMeta = toolConfig.annotations[ann.name];
        if (annMeta) {
            const paramType = annMeta.paramType;
            const hasParams = ann.params !== undefined && ann.params.trim() !== '';

            if (paramType) {
                // Annotation requires parameters
                if (!hasParams) {
                    const typeDesc = Array.isArray(paramType) ? paramType.join(' or ') : paramType;
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        `Annotation '${ann.tool}:@${ann.name}' requires a ${typeDesc} parameter`,
                        vscode.DiagnosticSeverity.Error
                    ));
                } else {
                    // Validate parameter value based on type
                    const paramValue = ann.params!.trim();
                    const types = Array.isArray(paramType) ? paramType : [paramType];

                    if (types.includes('enum') && annMeta.values) {
                        // Validate enum values (can be comma-separated)
                        const providedValues = paramValue.split(',').map(v => v.trim()).filter(v => v);
                        
                        // Check maxArgs limit
                        if (annMeta.maxArgs && providedValues.length > annMeta.maxArgs) {
                            diagnostics.push(new vscode.Diagnostic(
                                ann.range,
                                `Annotation '${ann.tool}:@${ann.name}' accepts at most ${annMeta.maxArgs} argument(s), got ${providedValues.length}`,
                                vscode.DiagnosticSeverity.Error
                            ));
                        }
                        
                        const invalidValues = providedValues.filter(v => !annMeta.values!.includes(v));
                        if (invalidValues.length > 0) {
                            diagnostics.push(new vscode.Diagnostic(
                                ann.range,
                                `Invalid option(s) '${invalidValues.join(', ')}' for '${ann.tool}:@${ann.name}'. Valid options: ${annMeta.values.join(', ')}`,
                                vscode.DiagnosticSeverity.Error
                            ));
                        }
                    } else if (!validateParamValue(paramValue, types)) {
                        const typeDesc = types.join(' or ');
                        diagnostics.push(new vscode.Diagnostic(
                            ann.range,
                            `Annotation '${ann.tool}:@${ann.name}' requires a ${typeDesc} parameter, got '${paramValue}'`,
                            vscode.DiagnosticSeverity.Error
                        ));
                    }
                    // string and list types accept any non-empty value
                }
            } else {
                // Annotation does not accept parameters
                if (hasParams) {
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        `Annotation '${ann.tool}:@${ann.name}' does not accept parameters`,
                        vscode.DiagnosticSeverity.Error
                    ));
                }
            }
        }
    }

    // Check for missing generated files
    const filePath = document.uri.fsPath;
    const dir = path.dirname(filePath);
    const pkgName = path.basename(dir);

    for (const type of types) {
        for (const ann of type.annotations) {
            const toolConfig = toolsConfig[ann.tool];
            if (!toolConfig) continue;

            if (toolConfig.typeAnnotations.includes(ann.name)) {
                const expectedFile = path.join(dir, pkgName + toolConfig.outputSuffix);

                if (!fs.existsSync(expectedFile)) {
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        `Generated file not found: ${path.basename(expectedFile)}. Run '${ann.tool}' to generate.`,
                        vscode.DiagnosticSeverity.Information
                    ));
                }
            }
        }
    }

    // Check for field annotations without type annotation
    for (const type of types) {
        const typeTools = new Set(type.annotations.map(a => a.tool));

        for (const field of type.fields) {
            for (const ann of field.annotations) {
                const toolConfig = toolsConfig[ann.tool];
                if (!toolConfig) continue;

                if (toolConfig.fieldAnnotations.includes(ann.name) && !typeTools.has(ann.tool)) {
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        `Field annotation '${ann.tool}:@${ann.name}' requires type annotation '${ann.tool}:@${toolConfig.typeAnnotations[0]}' on the struct`,
                        vscode.DiagnosticSeverity.Warning
                    ));
                }
            }
        }
    }

    diagnosticCollection.set(document.uri, diagnostics);
}

class DevGenCompletionProvider implements vscode.CompletionItemProvider {
    provideCompletionItems(
        document: vscode.TextDocument,
        position: vscode.Position,
        _token: vscode.CancellationToken,
        _context: vscode.CompletionContext
    ): vscode.CompletionList | vscode.CompletionItem[] | undefined {
        const lineText = document.lineAt(position).text;
        const linePrefix = lineText.substring(0, position.character);

        // Check if we're in a comment
        if (!linePrefix.includes('//') && !linePrefix.includes('/*')) {
            return undefined;
        }

        const items: vscode.CompletionItem[] = [];

        // Check if user is typing inside annotation params: toolname:@annotation(...)
        const insideParamsMatch = linePrefix.match(/(\w+):@([\w.]+)\(([^)]*)$/);
        if (insideParamsMatch) {
            const toolName = insideParamsMatch[1];
            const annName = insideParamsMatch[2];
            const existingParams = insideParamsMatch[3];

            const toolConfig = toolsConfig[toolName];
            if (!toolConfig) return undefined;

            const annMeta = toolConfig.annotations[annName];
            if (!annMeta || annMeta.paramType !== 'enum' || !annMeta.values) {
                return undefined;
            }

            // Get already selected options
            const selectedOptions = existingParams.split(',').map(s => s.trim()).filter(s => s);
            
            // Check if maxArgs limit reached
            if (annMeta.maxArgs && selectedOptions.length >= annMeta.maxArgs) {
                return undefined;
            }

            for (const opt of annMeta.values) {
                if (!selectedOptions.includes(opt)) {
                    const item = new vscode.CompletionItem(opt, vscode.CompletionItemKind.EnumMember);
                    item.detail = `${annName} option`;
                    const docText = annMeta.valueDocs?.[opt] || opt;
                    item.documentation = new vscode.MarkdownString(docText);
                    
                    // Add ", " prefix if there are existing params and no trailing comma/space
                    if (existingParams.length > 0 && !existingParams.match(/,\s*$/)) {
                        item.insertText = ', ' + opt;
                    } else if (existingParams.match(/,$/)) {
                        // Comma without space, add space before option
                        item.insertText = ' ' + opt;
                    }
                    item.sortText = '0' + opt;
                    item.preselect = items.length === 0;
                    items.push(item);
                }
            }

            return items.length > 0 ? new vscode.CompletionList(items, false) : undefined;
        }

        // Check if user is typing after "toolname:@" - only show completion right after @
        const afterAtMatch = linePrefix.match(/(\w+):@$/);
        if (afterAtMatch) {
            const toolName = afterAtMatch[1];
            const toolConfig = toolsConfig[toolName];

            if (toolConfig) {
                const allAnnotations = [
                    ...toolConfig.typeAnnotations,
                    ...toolConfig.fieldAnnotations
                ];

                for (const ann of allAnnotations) {
                    const annMeta = toolConfig.annotations[ann];
                    const item = new vscode.CompletionItem(ann, vscode.CompletionItemKind.Keyword);
                    item.detail = `${toolName} annotation`;
                    item.sortText = '0' + ann;
                    const docText = annMeta?.doc || ann;
                    item.documentation = new vscode.MarkdownString(docText);

                    // Add parameter snippet based on paramType
                    if (annMeta?.paramType && annMeta.paramType !== 'enum') {
                        const placeholder = annMeta.placeholder || 'value';
                        item.insertText = new vscode.SnippetString(`${ann}(\${1:${placeholder}})`);
                    } else if (annMeta?.paramType === 'enum') {
                        // For enum params, just insert the name, user will add () and select options
                        item.insertText = ann;
                    } else {
                        item.insertText = ann;
                    }

                    items.push(item);
                }
            }

            return items.length > 0 ? new vscode.CompletionList(items, false) : undefined;
        }

        // Check if user is typing a tool name (after "// ")
        const toolNameMatch = linePrefix.match(/\/\/\s*(\w*)$/);
        if (toolNameMatch) {
            for (const toolName of Object.keys(toolsConfig)) {
                const item = new vscode.CompletionItem(
                    `${toolName}:@`,
                    vscode.CompletionItemKind.Module
                );
                item.detail = 'DevGen tool';
                item.insertText = new vscode.SnippetString(`${toolName}:@\${1}`);
                items.push(item);
            }
            return items;
        }

        return undefined;
    }
}

class DevGenHoverProvider implements vscode.HoverProvider {
    provideHover(
        document: vscode.TextDocument,
        position: vscode.Position,
        _token: vscode.CancellationToken
    ): vscode.Hover | undefined {
        const wordRange = document.getWordRangeAtPosition(position, /\w+:@[\w.]+(?:\([^)]*\))?/);
        if (!wordRange) {
            return undefined;
        }

        const word = document.getText(wordRange);
        const match = word.match(/(\w+):@([\w.]+)(?:\(([^)]*)\))?/);
        if (!match) {
            return undefined;
        }

        const [, tool, annotation, params] = match;
        const toolConfig = toolsConfig[tool];

        if (!toolConfig) {
            return new vscode.Hover(`Unknown tool: ${tool}`);
        }

        const annMeta = toolConfig.annotations[annotation];
        const isTypeAnnotation = toolConfig.typeAnnotations.includes(annotation);
        const isFieldAnnotation = toolConfig.fieldAnnotations.includes(annotation);

        const markdown = new vscode.MarkdownString();
        markdown.appendMarkdown(`**${tool}:@${annotation}**\n\n`);

        if (annMeta?.doc) {
            markdown.appendMarkdown(`${annMeta.doc}\n\n`);
        }

        if (isTypeAnnotation) {
            markdown.appendMarkdown(`*Type-level annotation*\n\n`);
            markdown.appendMarkdown(`Generated file: \`*${toolConfig.outputSuffix}\`\n`);
        } else if (isFieldAnnotation) {
            markdown.appendMarkdown(`*Field-level annotation*\n`);
        } else {
            markdown.appendMarkdown(`⚠️ Unknown annotation for \`${tool}\`\n`);
        }

        if (params) {
            markdown.appendMarkdown(`\n\nParameters: \`${params}\``);
        }

        // Show available values for enum params
        if (annMeta?.paramType === 'enum' && annMeta.values) {
            markdown.appendMarkdown(`\n\n**Options:** ${annMeta.values.join(', ')}`);
        }

        return new vscode.Hover(markdown, wordRange);
    }
}

import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';

// Import config loader for dynamic configuration
import { ConfigLoader, getConfigLoader, ToolsConfig, ToolConfig, AnnotationMeta } from './config-loader';

// Annotation pattern: toolname:@annotation or toolname:@annotation(params)
const ANNOTATION_PATTERN = /(\w+):@([\w.]+)(?:\(([^)]*)\))?/g;

// Re-export types for compatibility
interface LSPConfig {
    enabled: boolean;
    provider: string;      // "gopls"
    feature: string;       // "method", "type", "symbol"
    signature?: string;    // Required signature pattern, e.g., "func() error"
    resolveFrom?: string;  // "fieldType", "receiverType"
}

// Dynamic tools configuration - updated when devgen.toml changes
let toolsConfig: ToolsConfig = {};

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
    typeName: string;      // Field type name (e.g., "Address", "*Address", "Status")
    isPointer: boolean;
    range: vscode.Range;
    annotations: ParsedAnnotation[];
}

// Method info from LSP
interface MethodInfo {
    name: string;
    signature: string;
    receiverType: string;
    location: vscode.Location;
}

// Cache for method lookups
interface MethodCache {
    methods: Map<string, MethodInfo[]>;  // typeName -> methods
    timestamp: number;
}

let diagnosticCollection: vscode.DiagnosticCollection;
let methodCache: MethodCache = { methods: new Map(), timestamp: 0 };
let outputChannel: vscode.OutputChannel;
let configLoader: ConfigLoader;
const CACHE_TTL = 5000; // 5 seconds

export async function activate(context: vscode.ExtensionContext) {
    console.log('DevGen extension activated');
    
    // Create output channel for debugging
    outputChannel = vscode.window.createOutputChannel('DevGen');
    context.subscriptions.push(outputChannel);

    // Initialize config loader and load dynamic configuration
    configLoader = getConfigLoader();
    await configLoader.initialize(context);
    toolsConfig = configLoader.getToolsConfig();
    
    // Listen for config changes
    configLoader.onConfigChanged((newConfig) => {
        toolsConfig = newConfig;
        outputChannel.appendLine('DevGen: Configuration reloaded');
        // Refresh diagnostics for all open Go files
        vscode.workspace.textDocuments.forEach(doc => {
            if (doc.languageId === 'go') {
                updateDiagnostics(doc);
            }
        });
    });

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

    // Update diagnostics on document change (debounced)
    let diagnosticTimeout: NodeJS.Timeout | undefined;
    context.subscriptions.push(
        vscode.workspace.onDidChangeTextDocument(e => {
            if (e.document.languageId === 'go') {
                if (diagnosticTimeout) {
                    clearTimeout(diagnosticTimeout);
                }
                diagnosticTimeout = setTimeout(() => {
                    updateDiagnostics(e.document);
                }, 500);
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

    // Update diagnostics on document save (trigger LSP validation)
    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument(doc => {
            if (doc.languageId === 'go') {
                // Clear cache on save to get fresh LSP data
                methodCache = { methods: new Map(), timestamp: 0 };
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
                braceCount = 0; // Will be set by the brace counting below
            }

            if (!trimmed.startsWith('//')) {
                docCommentStart = -1;
                docCommentLines = [];
            }
        }

        if (currentType) {
            for (const char of line) {
                if (char === '{') braceCount++;
                if (char === '}') braceCount--;
            }

            if (braceCount > 0 && !line.match(typePattern)) {
                // Parse field with type: FieldName TypeName or FieldName *TypeName
                const fieldMatch = line.match(/^\s+(\w+)\s+(\*?)(\w+)/);
                if (fieldMatch) {
                    const fieldAnnotations: ParsedAnnotation[] = [];

                    let j = i - 1;
                    while (j >= 0 && lines[j].trim().startsWith('//')) {
                        const originalLine = lines[j];
                        ANNOTATION_PATTERN.lastIndex = 0;
                        let annMatch;
                        while ((annMatch = ANNOTATION_PATTERN.exec(originalLine)) !== null) {
                            const lineStart = document.offsetAt(new vscode.Position(j, 0));
                            const absoluteStart = lineStart + annMatch.index;
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

                    currentType.fields.push({
                        name: fieldMatch[1],
                        typeName: fieldMatch[3],
                        isPointer: fieldMatch[2] === '*',
                        range: new vscode.Range(i, 0, i, line.length),
                        annotations: fieldAnnotations
                    });
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

// ============================================================================
// LSP Integration: Method lookup using gopls
// ============================================================================

/**
 * Get methods for a type using gopls workspace symbol search
 */
async function getMethodsForType(
    document: vscode.TextDocument,
    typeName: string
): Promise<MethodInfo[]> {
    // Check cache
    const now = Date.now();
    if (now - methodCache.timestamp < CACHE_TTL) {
        const cached = methodCache.methods.get(typeName);
        if (cached) {
            return cached;
        }
    }

    const methods: MethodInfo[] = [];

    try {
        // Method 1: Parse current document directly for method definitions
        // This is the most reliable way for local types
        const text = document.getText();
        // Match: func (x Type) MethodName() or func (x *Type) MethodName()
        const methodRegex = new RegExp(
            `func\\s*\\(\\s*\\w+\\s+\\*?${typeName}\\s*\\)\\s*(\\w+)\\s*\\(([^)]*)\\)\\s*([^{]*)`,
            'g'
        );
        
        let match;
        while ((match = methodRegex.exec(text)) !== null) {
            const methodName = match[1];
            const params = match[2] || '';
            const returnType = match[3]?.trim() || '';
            
            if (!methods.some(m => m.name === methodName)) {
                const pos = document.positionAt(match.index);
                methods.push({
                    name: methodName,
                    signature: `func(${params}) ${returnType}`.trim(),
                    receiverType: typeName,
                    location: new vscode.Location(document.uri, pos)
                });
            }
        }

        // Method 2: Use document symbols from gopls as fallback
        if (methods.length === 0) {
            const documentSymbols = await vscode.commands.executeCommand<vscode.DocumentSymbol[]>(
                'vscode.executeDocumentSymbolProvider',
                document.uri
            );

            if (documentSymbols) {
                findMethodsInSymbols(documentSymbols, typeName, document.uri, methods);
            }
        }

        // Method 3: Try workspace symbol search for cross-package types
        if (methods.length === 0) {
            const searchPatterns = [
                `${typeName}.`,
                `(*${typeName}).`,
                `(${typeName}).`,
            ];

            for (const pattern of searchPatterns) {
                const symbols = await vscode.commands.executeCommand<vscode.SymbolInformation[]>(
                    'vscode.executeWorkspaceSymbolProvider',
                    pattern
                );

                if (symbols && symbols.length > 0) {
                    for (const symbol of symbols) {
                        if (symbol.kind === vscode.SymbolKind.Method || 
                            symbol.kind === vscode.SymbolKind.Function) {
                            const methodMatch = symbol.name.match(/\(\*?(\w+)\)\.(\w+)/);
                            if (methodMatch && methodMatch[1] === typeName) {
                                const methodName = methodMatch[2];
                                if (!methods.some(m => m.name === methodName)) {
                                    methods.push({
                                        name: methodName,
                                        signature: '',
                                        receiverType: typeName,
                                        location: symbol.location
                                    });
                                }
                            }
                        }
                    }
                }
                
                if (methods.length > 0) break;
            }
        }

        // Get method signatures using hover for methods without signatures
        for (const method of methods) {
            if (!method.signature || method.signature === '') {
                const signature = await getMethodSignature(method.location.uri, method.location.range.start, method.name);
                if (signature) {
                    method.signature = signature;
                }
            }
        }

        // Update cache
        methodCache.methods.set(typeName, methods);
        methodCache.timestamp = now;

    } catch (error) {
        console.error('Error getting methods for type:', typeName, error);
    }

    return methods;
}

function findMethodsInSymbols(
    symbols: vscode.DocumentSymbol[],
    typeName: string,
    uri: vscode.Uri,
    methods: MethodInfo[]
): void {
    for (const symbol of symbols) {
        // gopls returns methods as Function kind with name like "(*Type).Method" or "(Type).Method"
        if (symbol.kind === vscode.SymbolKind.Method || 
            symbol.kind === vscode.SymbolKind.Function) {
            
            // Try to match gopls format: "(*Type).MethodName" or "(Type).MethodName"
            const goplsMatch = symbol.name.match(/^\(\*?(\w+)\)\.(\w+)$/);
            if (goplsMatch) {
                const receiverType = goplsMatch[1];
                const methodName = goplsMatch[2];
                
                if (receiverType === typeName) {
                    if (!methods.some(m => m.name === methodName)) {
                        methods.push({
                            name: methodName,
                            signature: symbol.detail || '',
                            receiverType: typeName,
                            location: new vscode.Location(uri, symbol.selectionRange || symbol.range)
                        });
                    }
                }
                continue;
            }
            
            // Fallback: try other formats
            const nameMatch = symbol.name.match(/^\((\w+)\s+\*?(\w+)\)\s+(\w+)|^(\w+)$/);
            if (nameMatch) {
                const receiverType = nameMatch[2] || '';
                const methodName = nameMatch[3] || nameMatch[4] || symbol.name;
                
                if (receiverType === typeName) {
                    if (!methods.some(m => m.name === methodName)) {
                        methods.push({
                            name: methodName,
                            signature: symbol.detail || '',
                            receiverType: typeName,
                            location: new vscode.Location(uri, symbol.selectionRange || symbol.range)
                        });
                    }
                }
            }
        }
        
        if (symbol.children) {
            findMethodsInSymbols(symbol.children, typeName, uri, methods);
        }
    }
}

/**
 * Get method signature using hover provider
 */
async function getMethodSignature(
    uri: vscode.Uri,
    position: vscode.Position,
    methodName: string
): Promise<string | undefined> {
    try {
        const hovers = await vscode.commands.executeCommand<vscode.Hover[]>(
            'vscode.executeHoverProvider',
            uri,
            position
        );

        if (hovers && hovers.length > 0) {
            for (const hover of hovers) {
                for (const content of hover.contents) {
                    let text = '';
                    if (typeof content === 'string') {
                        text = content;
                    } else if ('value' in content) {
                        text = content.value;
                    }
                    
                    // gopls hover format for methods:
                    // ```go
                    // func (s Status) Validate() error
                    // ```
                    // or just: func (s Status) Validate() error
                    
                    // Extract function signature - look for the method definition
                    const lines = text.split('\n');
                    for (const line of lines) {
                        // Match: func (receiver) MethodName(params) returnType
                        const sigMatch = line.match(/func\s*\([^)]*\)\s*(\w+)\s*(\([^)]*\))\s*(.*)/);
                        if (sigMatch && sigMatch[1] === methodName) {
                            const params = sigMatch[2];
                            const returnType = sigMatch[3]?.trim() || '';
                            return `func${params} ${returnType}`.trim();
                        }
                        
                        // Also try simpler pattern
                        const simpleMatch = line.match(/func\s*\([^)]*\)\s*\w+(\([^)]*\))\s*(\w+)?/);
                        if (simpleMatch) {
                            const params = simpleMatch[1];
                            const returnType = simpleMatch[2] || '';
                            return `func${params} ${returnType}`.trim();
                        }
                    }
                }
            }
        }
    } catch (error) {
        console.error('Error getting method signature:', methodName, error);
    }
    return undefined;
}

/**
 * Find type definition location using gopls
 */
async function findTypeDefinition(
    document: vscode.TextDocument,
    typeName: string
): Promise<vscode.Location | undefined> {
    try {
        // Search for type definition in workspace
        const symbols = await vscode.commands.executeCommand<vscode.SymbolInformation[]>(
            'vscode.executeWorkspaceSymbolProvider',
            typeName
        );

        if (symbols) {
            for (const symbol of symbols) {
                if ((symbol.kind === vscode.SymbolKind.Struct ||
                     symbol.kind === vscode.SymbolKind.Class ||
                     symbol.kind === vscode.SymbolKind.Interface) &&
                    symbol.name === typeName) {
                    return symbol.location;
                }
            }
        }
    } catch (error) {
        console.error('Error finding type definition:', error);
    }
    return undefined;
}

/**
 * Validate if a method exists and matches required signature
 */
async function validateMethod(
    document: vscode.TextDocument,
    typeName: string,
    methodName: string,
    requiredSignature?: string
): Promise<{ exists: boolean; valid: boolean; actualSignature?: string; message?: string }> {
    const methods = await getMethodsForType(document, typeName);
    
    const method = methods.find(m => m.name === methodName);
    
    if (!method) {
        return {
            exists: false,
            valid: false,
            message: `Method '${methodName}' not found on type '${typeName}'`
        };
    }

    if (requiredSignature && method.signature) {
        // Normalize signatures for comparison
        const normalizedRequired = normalizeSignature(requiredSignature);
        const normalizedActual = normalizeSignature(method.signature);
        
        if (normalizedActual && !signatureMatches(normalizedActual, normalizedRequired)) {
            return {
                exists: true,
                valid: false,
                actualSignature: method.signature,
                message: `Method '${methodName}' has signature '${method.signature}', expected '${requiredSignature}'`
            };
        }
    }

    return { exists: true, valid: true, actualSignature: method.signature };
}

function normalizeSignature(sig: string): string {
    // Remove "func" prefix and whitespace
    return sig.replace(/^func\s*/, '').replace(/\s+/g, ' ').trim();
}

function signatureMatches(actual: string, required: string): boolean {
    // Simple matching: check if actual contains required pattern
    // For "() error", we check if method takes no params and returns error
    const normalizedActual = normalizeSignature(actual);
    const normalizedRequired = normalizeSignature(required);
    
    // Handle "() error" pattern
    if (normalizedRequired === '() error') {
        return normalizedActual.includes('()') && normalizedActual.includes('error');
    }
    
    return normalizedActual.includes(normalizedRequired);
}

// ============================================================================
// Diagnostics
// ============================================================================

async function updateDiagnostics(document: vscode.TextDocument) {
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
                    const paramTypes = Array.isArray(paramType) ? paramType : [paramType];

                    if (paramTypes.includes('enum') && annMeta.values) {
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
                    } else if (!validateParamValue(paramValue, paramTypes)) {
                        const typeDesc = paramTypes.join(' or ');
                        diagnostics.push(new vscode.Diagnostic(
                            ann.range,
                            `Annotation '${ann.tool}:@${ann.name}' requires a ${typeDesc} parameter, got '${paramValue}'`,
                            vscode.DiagnosticSeverity.Error
                        ));
                    }
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

    // LSP-based validation for @method annotations
    await validateMethodAnnotations(document, types, diagnostics);

    diagnosticCollection.set(document.uri, diagnostics);
}

/**
 * Validate @method annotations using LSP
 */
async function validateMethodAnnotations(
    document: vscode.TextDocument,
    types: TypeInfo[],
    diagnostics: vscode.Diagnostic[]
): Promise<void> {
    for (const type of types) {
        for (const field of type.fields) {
            for (const ann of field.annotations) {
                const toolConfig = toolsConfig[ann.tool];
                if (!toolConfig) continue;

                const annMeta = toolConfig.annotations[ann.name];
                if (!annMeta?.lsp?.enabled || annMeta.lsp.feature !== 'method') continue;

                const methodName = ann.params?.trim();
                if (!methodName) continue;

                // Get the type to validate against
                let targetType = field.typeName;
                if (annMeta.lsp.resolveFrom === 'fieldType') {
                    targetType = field.typeName;
                }

                // Validate method exists and has correct signature
                const validation = await validateMethod(
                    document,
                    targetType,
                    methodName,
                    annMeta.lsp.signature
                );

                if (!validation.exists) {
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        validation.message || `Method '${methodName}' not found on type '${targetType}'`,
                        vscode.DiagnosticSeverity.Error
                    ));
                } else if (!validation.valid) {
                    diagnostics.push(new vscode.Diagnostic(
                        ann.range,
                        validation.message || `Method '${methodName}' has invalid signature`,
                        vscode.DiagnosticSeverity.Warning
                    ));
                }
            }
        }
    }
}

// ============================================================================
// Completion Provider
// ============================================================================

class DevGenCompletionProvider implements vscode.CompletionItemProvider {
    async provideCompletionItems(
        document: vscode.TextDocument,
        position: vscode.Position,
        _token: vscode.CancellationToken,
        _context: vscode.CompletionContext
    ): Promise<vscode.CompletionList | vscode.CompletionItem[] | undefined> {
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
            if (!annMeta) return undefined;

            // LSP-based completion for @method
            if (annMeta.lsp?.enabled && annMeta.lsp.feature === 'method') {
                return await this.provideMethodCompletion(document, position, existingParams, annMeta);
            }

            // Enum completion
            if (annMeta.paramType !== 'enum' || !annMeta.values) {
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

    /**
     * Provide method completion using LSP
     */
    private async provideMethodCompletion(
        document: vscode.TextDocument,
        position: vscode.Position,
        existingParams: string,
        annMeta: AnnotationMeta
    ): Promise<vscode.CompletionList | undefined> {
        const items: vscode.CompletionItem[] = [];

        // Find the field type at current position
        const fieldInfo = this.findFieldAtPosition(document, position);
        if (!fieldInfo) return undefined;

        // Get methods for the field type
        const methods = await getMethodsForType(document, fieldInfo.typeName);

        // Filter methods by required signature
        const requiredSig = annMeta.lsp?.signature;
        
        for (const method of methods) {
            // Filter by signature if required
            if (requiredSig && method.signature) {
                if (!signatureMatches(method.signature, requiredSig)) {
                    continue;
                }
            }

            // Skip if already in params
            if (existingParams.includes(method.name)) {
                continue;
            }

            const item = new vscode.CompletionItem(method.name, vscode.CompletionItemKind.Method);
            item.detail = `${fieldInfo.typeName}.${method.name}`;
            
            const doc = new vscode.MarkdownString();
            doc.appendCodeblock(`func (${fieldInfo.typeName}) ${method.name}${method.signature || '()'}`, 'go');
            if (requiredSig) {
                doc.appendMarkdown(`\n\nRequired signature: \`${requiredSig}\``);
            }
            item.documentation = doc;
            
            item.sortText = '0' + method.name;
            items.push(item);
        }

        return items.length > 0 ? new vscode.CompletionList(items, false) : undefined;
    }

    /**
     * Find field info at the given position
     */
    private findFieldAtPosition(document: vscode.TextDocument, position: vscode.Position): FieldInfo | undefined {
        const types = parseTypes(document);
        
        for (const type of types) {
            for (const field of type.fields) {
                // Check if position is in the field's annotation comments
                // Look at lines above the field
                const fieldLine = field.range.start.line;
                if (position.line < fieldLine && position.line >= fieldLine - 10) {
                    return field;
                }
            }
        }
        
        return undefined;
    }
}

// ============================================================================
// Hover Provider
// ============================================================================

class DevGenHoverProvider implements vscode.HoverProvider {
    async provideHover(
        document: vscode.TextDocument,
        position: vscode.Position,
        _token: vscode.CancellationToken
    ): Promise<vscode.Hover | undefined> {
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

        // Show LSP info for @method
        if (annMeta?.lsp?.enabled && params) {
            markdown.appendMarkdown(`\n\n---\n`);
            markdown.appendMarkdown(`**LSP Integration:** ${annMeta.lsp.provider}\n\n`);
            
            if (annMeta.lsp.signature) {
                markdown.appendMarkdown(`Required signature: \`${annMeta.lsp.signature}\`\n`);
            }

            // Try to get method info
            const fieldInfo = this.findFieldAtPosition(document, position);
            if (fieldInfo && params) {
                const validation = await validateMethod(
                    document,
                    fieldInfo.typeName,
                    params.trim(),
                    annMeta.lsp.signature
                );

                if (validation.exists) {
                    markdown.appendMarkdown(`\n✅ Method found`);
                    if (validation.actualSignature) {
                        markdown.appendMarkdown(`: \`${validation.actualSignature}\``);
                    }
                    if (!validation.valid) {
                        markdown.appendMarkdown(`\n⚠️ ${validation.message}`);
                    }
                } else {
                    markdown.appendMarkdown(`\n❌ ${validation.message}`);
                }
            }
        }

        return new vscode.Hover(markdown, wordRange);
    }

    private findFieldAtPosition(document: vscode.TextDocument, position: vscode.Position): FieldInfo | undefined {
        const types = parseTypes(document);
        
        for (const type of types) {
            for (const field of type.fields) {
                const fieldLine = field.range.start.line;
                if (position.line < fieldLine && position.line >= fieldLine - 10) {
                    return field;
                }
            }
        }
        
        return undefined;
    }
}

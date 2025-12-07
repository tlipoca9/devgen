import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { parse as parseToml } from 'smol-toml';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Types matching Go genkit/config.go structures

interface AnnotationParams {
    type?: string | string[];
    values?: string[];
    placeholder?: string;
    maxArgs?: number;
    docs?: { [key: string]: string };
}

interface LSPConfig {
    enabled?: boolean;
    provider?: string;
    feature?: string;
    signature?: string;
    resolveFrom?: string;
}

interface AnnotationConfig {
    name: string;
    type: string;  // "type" or "field"
    doc: string;
    params?: AnnotationParams;
    lsp?: LSPConfig;
}

interface ToolConfigToml {
    output_suffix?: string;
    annotations?: AnnotationConfig[];
}

interface PluginConfig {
    name: string;
    path: string;
    type?: 'source' | 'plugin';
}

interface DevgenToml {
    plugins?: PluginConfig[];
    tools?: { [toolName: string]: ToolConfigToml };
}

// VSCode extension internal types
export interface AnnotationMeta {
    doc: string;
    paramType?: string | string[];
    placeholder?: string;
    values?: string[];
    valueDocs?: { [key: string]: string };
    maxArgs?: number;
    lsp?: {
        enabled: boolean;
        provider: string;
        feature: string;
        signature?: string;
        resolveFrom?: string;
    };
}

export interface ToolConfig {
    typeAnnotations: string[];
    fieldAnnotations: string[];
    outputSuffix: string;
    annotations: { [name: string]: AnnotationMeta };
}

export interface ToolsConfig {
    [toolName: string]: ToolConfig;
}

const DEVGEN_PACKAGE = 'github.com/tlipoca9/devgen/cmd/devgen@latest';

/**
 * ConfigLoader manages loading and merging tool configurations from:
 * 1. devgen CLI (via `devgen config --json`)
 * 2. Project-level devgen.toml files (manual overrides)
 */
export class ConfigLoader {
    private static instance: ConfigLoader;
    private toolsConfig: ToolsConfig;
    private configWatcher: vscode.FileSystemWatcher | undefined;
    private onConfigChangedEmitter = new vscode.EventEmitter<ToolsConfig>();
    private devgenPath: string = 'devgen';
    private isInstalling: boolean = false;

    /** Event fired when configuration changes */
    public readonly onConfigChanged = this.onConfigChangedEmitter.event;

    private constructor() {
        // Start with empty config
        this.toolsConfig = {};
    }

    public static getInstance(): ConfigLoader {
        if (!ConfigLoader.instance) {
            ConfigLoader.instance = new ConfigLoader();
        }
        return ConfigLoader.instance;
    }

    /**
     * Initialize the config loader and set up file watchers.
     */
    public async initialize(context: vscode.ExtensionContext): Promise<void> {
        // Get devgen path from settings
        const config = vscode.workspace.getConfiguration('devgen');
        this.devgenPath = config.get<string>('executablePath') || 'devgen';

        // Load initial configuration
        await this.reloadConfig();

        // Watch for devgen.toml changes
        this.configWatcher = vscode.workspace.createFileSystemWatcher('**/devgen.toml');
        
        this.configWatcher.onDidCreate(() => this.reloadConfig());
        this.configWatcher.onDidChange(() => this.reloadConfig());
        this.configWatcher.onDidDelete(() => this.reloadConfig());

        context.subscriptions.push(this.configWatcher);
        context.subscriptions.push(this.onConfigChangedEmitter);

        // Watch for settings changes
        context.subscriptions.push(
            vscode.workspace.onDidChangeConfiguration(e => {
                if (e.affectsConfiguration('devgen.executablePath')) {
                    this.devgenPath = vscode.workspace.getConfiguration('devgen')
                        .get<string>('executablePath') || 'devgen';
                    this.reloadConfig();
                }
            })
        );
    }

    /**
     * Get the current merged tools configuration.
     */
    public getToolsConfig(): ToolsConfig {
        return this.toolsConfig;
    }

    /**
     * Reload configuration from all sources.
     */
    public async reloadConfig(): Promise<void> {
        let mergedConfig: ToolsConfig = {};

        // Try to get config from devgen CLI
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (workspaceFolders && workspaceFolders.length > 0) {
            const cliConfig = await this.loadConfigFromCLI(workspaceFolders[0].uri.fsPath);
            if (cliConfig) {
                mergedConfig = cliConfig;
            }
        }

        // Also load devgen.toml for any manual overrides
        if (workspaceFolders) {
            for (const folder of workspaceFolders) {
                const configPath = await this.findConfigFile(folder.uri.fsPath);
                if (configPath) {
                    try {
                        const projectConfig = await this.loadConfigFile(configPath);
                        this.mergeConfig(mergedConfig, projectConfig);
                    } catch (error) {
                        console.error(`Failed to load ${configPath}:`, error);
                    }
                }
            }
        }

        this.toolsConfig = mergedConfig;
        this.onConfigChangedEmitter.fire(this.toolsConfig);
    }

    /**
     * Load configuration from devgen CLI.
     */
    private async loadConfigFromCLI(workspaceDir: string): Promise<ToolsConfig | null> {
        try {
            const { stdout } = await execAsync(`${this.devgenPath} config --json`, {
                cwd: workspaceDir,
                timeout: 10000, // 10 second timeout
            });

            const config = JSON.parse(stdout) as ToolsConfig;
            console.log('DevGen: Loaded config from CLI');
            return config;
        } catch (error) {
            // devgen not installed or failed - try to install
            console.log('DevGen: CLI not available, attempting to install...');
            const installed = await this.tryInstallDevgen();
            if (installed) {
                // Retry loading config after installation
                return this.loadConfigFromCLI(workspaceDir);
            }
            return null;
        }
    }

    /**
     * Try to install devgen using go install.
     */
    private async tryInstallDevgen(): Promise<boolean> {
        if (this.isInstalling) {
            return false;
        }

        this.isInstalling = true;

        try {
            // Check if Go is available
            await execAsync('go version', { timeout: 5000 });
        } catch {
            console.log('DevGen: Go not found, cannot auto-install');
            vscode.window.showWarningMessage(
                'DevGen: Go is not installed. Please install Go and devgen manually.'
            );
            this.isInstalling = false;
            return false;
        }

        try {
            // Show progress notification
            return await vscode.window.withProgress(
                {
                    location: vscode.ProgressLocation.Notification,
                    title: 'DevGen: Installing devgen...',
                    cancellable: false,
                },
                async () => {
                    try {
                        await execAsync(`go install ${DEVGEN_PACKAGE}`, {
                            timeout: 60000, // 60 second timeout for install
                        });
                        console.log('DevGen: Successfully installed devgen');
                        vscode.window.showInformationMessage('DevGen: Successfully installed devgen');
                        return true;
                    } catch (installError) {
                        console.error('DevGen: Failed to install:', installError);
                        vscode.window.showErrorMessage(
                            `DevGen: Failed to install devgen. Run 'go install ${DEVGEN_PACKAGE}' manually.`
                        );
                        return false;
                    } finally {
                        this.isInstalling = false;
                    }
                }
            );
        } catch {
            this.isInstalling = false;
            return false;
        }
    }

    /**
     * Find devgen.toml in the given directory or its parents.
     */
    private async findConfigFile(dir: string): Promise<string | null> {
        let currentDir = dir;
        const root = path.parse(currentDir).root;

        while (currentDir !== root) {
            const configPath = path.join(currentDir, 'devgen.toml');
            if (fs.existsSync(configPath)) {
                return configPath;
            }
            currentDir = path.dirname(currentDir);
        }

        return null;
    }

    /**
     * Load and parse a devgen.toml file.
     */
    private async loadConfigFile(configPath: string): Promise<ToolsConfig> {
        const content = fs.readFileSync(configPath, 'utf-8');
        const tomlData = parseToml(content) as DevgenToml;

        const result: ToolsConfig = {};

        // Convert tools section to VSCode format
        if (tomlData.tools) {
            for (const [toolName, toolConfig] of Object.entries(tomlData.tools)) {
                result[toolName] = this.convertToolConfig(toolConfig);
            }
        }

        return result;
    }

    /**
     * Convert TOML tool config to VSCode extension format.
     */
    private convertToolConfig(tomlConfig: ToolConfigToml): ToolConfig {
        const typeAnnotations: string[] = [];
        const fieldAnnotations: string[] = [];
        const annotations: { [name: string]: AnnotationMeta } = {};

        if (tomlConfig.annotations) {
            for (const ann of tomlConfig.annotations) {
                const meta: AnnotationMeta = {
                    doc: ann.doc || '',
                };

                if (ann.params) {
                    if (ann.params.type) {
                        meta.paramType = ann.params.type;
                    }
                    if (ann.params.values) {
                        meta.values = ann.params.values;
                        // If values are provided, it's an enum type
                        if (!meta.paramType) {
                            meta.paramType = 'enum';
                        }
                    }
                    if (ann.params.placeholder) {
                        meta.placeholder = ann.params.placeholder;
                    }
                    if (ann.params.maxArgs) {
                        meta.maxArgs = ann.params.maxArgs;
                    }
                    if (ann.params.docs) {
                        meta.valueDocs = ann.params.docs;
                    }
                }

                if (ann.lsp && ann.lsp.enabled) {
                    meta.lsp = {
                        enabled: ann.lsp.enabled,
                        provider: ann.lsp.provider || 'gopls',
                        feature: ann.lsp.feature || 'method',
                        signature: ann.lsp.signature,
                        resolveFrom: ann.lsp.resolveFrom,
                    };
                }

                annotations[ann.name] = meta;

                // Categorize by type
                if (ann.type === 'type') {
                    typeAnnotations.push(ann.name);
                } else if (ann.type === 'field') {
                    fieldAnnotations.push(ann.name);
                }
            }
        }

        return {
            typeAnnotations,
            fieldAnnotations,
            outputSuffix: tomlConfig.output_suffix || '',
            annotations,
        };
    }

    /**
     * Merge project config into base config.
     * Project config takes precedence.
     */
    private mergeConfig(base: ToolsConfig, project: ToolsConfig): void {
        for (const [toolName, toolConfig] of Object.entries(project)) {
            if (base[toolName]) {
                // Merge with existing tool
                const existing = base[toolName];
                
                // Merge annotations
                for (const [annName, annMeta] of Object.entries(toolConfig.annotations)) {
                    existing.annotations[annName] = annMeta;
                }

                // Merge annotation lists (avoid duplicates)
                for (const ann of toolConfig.typeAnnotations) {
                    if (!existing.typeAnnotations.includes(ann)) {
                        existing.typeAnnotations.push(ann);
                    }
                }
                for (const ann of toolConfig.fieldAnnotations) {
                    if (!existing.fieldAnnotations.includes(ann)) {
                        existing.fieldAnnotations.push(ann);
                    }
                }

                // Override output suffix if provided
                if (toolConfig.outputSuffix) {
                    existing.outputSuffix = toolConfig.outputSuffix;
                }
            } else {
                // Add new tool
                base[toolName] = toolConfig;
            }
        }
    }

    /**
     * Dispose resources.
     */
    public dispose(): void {
        if (this.configWatcher) {
            this.configWatcher.dispose();
        }
        this.onConfigChangedEmitter.dispose();
    }
}

/**
 * Get the singleton ConfigLoader instance.
 */
export function getConfigLoader(): ConfigLoader {
    return ConfigLoader.getInstance();
}

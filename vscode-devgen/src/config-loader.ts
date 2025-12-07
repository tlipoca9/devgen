import * as vscode from 'vscode';
import * as path from 'path';
import * as os from 'os';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Output channel for logging
let outputChannel: vscode.OutputChannel | undefined;

function log(message: string): void {
    console.log(`DevGen: ${message}`);
    outputChannel?.appendLine(`[ConfigLoader] ${message}`);
}

function logError(message: string, error?: unknown): void {
    const errorMsg = error instanceof Error ? error.message : String(error);
    console.error(`DevGen: ${message}`, error);
    outputChannel?.appendLine(`[ConfigLoader] ERROR: ${message} - ${errorMsg}`);
}

/**
 * Get environment with GOPATH/bin added to PATH
 */
function getEnvWithGoPath(): NodeJS.ProcessEnv {
    const env = { ...process.env };
    const homeDir = os.homedir();
    const goBin = path.join(homeDir, 'go', 'bin');
    
    // Add GOPATH/bin to PATH if not already present
    const pathSep = process.platform === 'win32' ? ';' : ':';
    const currentPath = env.PATH || '';
    if (!currentPath.includes(goBin)) {
        env.PATH = `${goBin}${pathSep}${currentPath}`;
    }
    
    return env;
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
 * ConfigLoader manages loading tool configurations from devgen CLI.
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
    public async initialize(context: vscode.ExtensionContext, channel?: vscode.OutputChannel): Promise<void> {
        // Set output channel for logging
        outputChannel = channel;
        
        // Get devgen path from settings
        const config = vscode.workspace.getConfiguration('devgen');
        this.devgenPath = config.get<string>('executablePath') || 'devgen';
        log(`Using devgen path: ${this.devgenPath}`);

        // Load initial configuration
        await this.reloadConfig();

        // Watch for devgen.toml changes (triggers CLI reload)
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
     * Get the current tools configuration.
     */
    public getToolsConfig(): ToolsConfig {
        return this.toolsConfig;
    }

    /**
     * Reload configuration from devgen CLI.
     */
    public async reloadConfig(): Promise<void> {
        let config: ToolsConfig = {};

        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (workspaceFolders && workspaceFolders.length > 0) {
            log(`Loading config for workspace: ${workspaceFolders[0].uri.fsPath}`);
            const cliConfig = await this.loadConfigFromCLI(workspaceFolders[0].uri.fsPath);
            if (cliConfig) {
                config = cliConfig;
            }
        } else {
            log('No workspace folders found');
        }

        this.toolsConfig = config;
        log(`Final config has ${Object.keys(config).length} tools: ${Object.keys(config).join(', ') || '(none)'}`);
        this.onConfigChangedEmitter.fire(this.toolsConfig);
    }

    /**
     * Load configuration from devgen CLI.
     */
    private async loadConfigFromCLI(workspaceDir: string): Promise<ToolsConfig | null> {
        try {
            const cmd = `${this.devgenPath} config --json`;
            log(`Executing: ${cmd}`);
            
            const { stdout, stderr } = await execAsync(cmd, {
                cwd: workspaceDir,
                timeout: 10000, // 10 second timeout
                env: getEnvWithGoPath(),
            });

            if (stderr) {
                log(`CLI stderr: ${stderr}`);
            }

            const config = JSON.parse(stdout) as ToolsConfig;
            log(`CLI returned ${Object.keys(config).length} tools`);
            return config;
        } catch (error) {
            logError('CLI command failed', error);
            // devgen not installed or failed - try to install
            log('Attempting to install devgen...');
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
            await execAsync('go version', { timeout: 5000, env: getEnvWithGoPath() });
        } catch {
            log('Go not found, cannot auto-install');
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
                        log(`Installing: go install ${DEVGEN_PACKAGE}`);
                        await execAsync(`go install ${DEVGEN_PACKAGE}`, {
                            timeout: 60000, // 60 second timeout for install
                            env: getEnvWithGoPath(),
                        });
                        log('Successfully installed devgen');
                        vscode.window.showInformationMessage('DevGen: Successfully installed devgen');
                        return true;
                    } catch (installError) {
                        logError('Failed to install', installError);
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

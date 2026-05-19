export interface Context {
}

export interface environment {
    BuildType: string;
    Platform: string;
    Arch: string;
}

export interface WCLOptions {
    [key: string]: any;
}

export interface MessageDialogOptions {
    Type?: string;
    Title?: string;
    Message?: string;
    Buttons?: string[];
    DefaultButton?: string;
    CancelButton?: string;
}

export interface OpenDialogOptions {
    DefaultDirectory?: string;
    DefaultFilename?: string;
    Title?: string;
    Filters?: Array<{ DisplayName: string; Pattern: string }>;
    ShowHiddenFiles?: boolean;
    CanCreateDirectories?: boolean;
    ResolvesAliases?: boolean;
    TreatPackagesAsDirectories?: boolean;
}

export interface SaveDialogOptions {
    DefaultDirectory?: string;
    DefaultFilename?: string;
    Title?: string;
    Filters?: Array<{ DisplayName: string; Pattern: string }>;
    ShowHiddenFiles?: boolean;
    CanCreateDirectories?: boolean;
    TreatPackagesAsDirectories?: boolean;
}

export function EventsOn(eventName: string, callback: (...data: any) => void): () => void;

export function EventsOnMultiple(eventName: string, callback: (...data: any) => void, maxCallbacks: number): () => void;

export function EventsOnce(eventName: string, callback: (...data: any) => void): () => void;

export function EventsOff(eventName: string, ...additionalEventNames: string[]): void;

export function EventsEmit(eventName: string, ...data: any): void;

export function LogPrint(message: string): void;

export function LogTrace(message: string): void;

export function LogDebug(message: string): void;

export function LogInfo(message: string): void;

export function LogWarning(message: string): void;

export function LogError(message: string): void;

export function LogFatal(message: string): void;

export function EventsNotify(eventName: string, data?: any): void;

export function WindowReload(): void;

export function WindowReloadApp(): void;

export function WindowSetAlwaysOnTop(b: boolean): void;

export function WindowSetSystemDefaultTheme(): void;

export function WindowSetLightTheme(): void;

export function WindowSetDarkTheme(): void;

export function WindowCenter(): void;

export function WindowSetTitle(title: string): void;

export function WindowFullscreen(): void;

export function WindowUnfullscreen(): void;

export function WindowSetSize(width: number, height: number): void;

export function WindowGetSize(): Promise<{ w: number; h: number }>;

export function WindowSetMaxSize(width: number, height: number): void;

export function WindowSetMinSize(width: number, height: number): void;

export function WindowSetPosition(x: number, y: number): void;

export function WindowGetPosition(): Promise<{ x: number; y: number }>;

export function WindowHide(): void;

export function WindowShow(): void;

export function WindowMaximise(): void;

export function WindowToggleMaximise(): void;

export function WindowUnmaximise(): void;

export function WindowMinimise(): void;

export function WindowUnminimise(): void;

export function WindowSetBackgroundColour(R: number, G: number, B: number, A: number): void;

export function ScreenGetAll(): Promise<Array<{ id: number; name: string; width: number; height: number }>>;

export function BrowserOpenURL(url: string): void;

export function Environment(): Promise<environment>;

export function Quit(): void;

export function Hide(): void;

export function Show(): void;

export function ClipboardGetText(): Promise<string>;

export function ClipboardSetText(text: string): Promise<boolean>;

export function Callback(incomingEmitFn: (name: string, data?: any) => void, target?: HTMLElement | null): void;

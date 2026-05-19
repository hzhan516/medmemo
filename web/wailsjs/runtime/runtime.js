/**
 * Wails v2 Runtime JS 绑定（手动维护，与 runtime.d.ts 对齐）。
 * 实际运行时由 Wails 注入 window.runtime 对象。
 */

const runtime = window.runtime;

export function EventsOn(eventName, callback) {
  return runtime.EventsOn(eventName, callback);
}

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
  return runtime.EventsOnMultiple(eventName, callback, maxCallbacks);
}

export function EventsOnce(eventName, callback) {
  return runtime.EventsOnce(eventName, callback);
}

export function EventsOff(eventName, ...additionalEventNames) {
  return runtime.EventsOff(eventName, ...additionalEventNames);
}

export function EventsEmit(eventName, ...data) {
  return runtime.EventsEmit(eventName, ...data);
}

export function LogPrint(message) {
  runtime.LogPrint(message);
}

export function LogTrace(message) {
  runtime.LogTrace(message);
}

export function LogDebug(message) {
  runtime.LogDebug(message);
}

export function LogInfo(message) {
  runtime.LogInfo(message);
}

export function LogWarning(message) {
  runtime.LogWarning(message);
}

export function LogError(message) {
  runtime.LogError(message);
}

export function LogFatal(message) {
  runtime.LogFatal(message);
}

export function EventsNotify(eventName, data) {
  runtime.EventsNotify(eventName, data);
}

export function WindowReload() {
  runtime.WindowReload();
}

export function WindowReloadApp() {
  runtime.WindowReloadApp();
}

export function WindowSetAlwaysOnTop(b) {
  runtime.WindowSetAlwaysOnTop(b);
}

export function WindowSetSystemDefaultTheme() {
  runtime.WindowSetSystemDefaultTheme();
}

export function WindowSetLightTheme() {
  runtime.WindowSetLightTheme();
}

export function WindowSetDarkTheme() {
  runtime.WindowSetDarkTheme();
}

export function WindowCenter() {
  runtime.WindowCenter();
}

export function WindowSetTitle(title) {
  runtime.WindowSetTitle(title);
}

export function WindowFullscreen() {
  runtime.WindowFullscreen();
}

export function WindowUnfullscreen() {
  runtime.Unfullscreen();
}

export function WindowSetSize(width, height) {
  runtime.WindowSetSize(width, height);
}

export function WindowGetSize() {
  return runtime.WindowGetSize();
}

export function WindowSetMaxSize(width, height) {
  runtime.WindowSetMaxSize(width, height);
}

export function WindowSetMinSize(width, height) {
  runtime.WindowSetMinSize(width, height);
}

export function WindowSetPosition(x, y) {
  runtime.WindowSetPosition(x, y);
}

export function WindowGetPosition() {
  return runtime.WindowGetPosition();
}

export function WindowHide() {
  runtime.WindowHide();
}

export function WindowShow() {
  runtime.WindowShow();
}

export function WindowMaximise() {
  runtime.WindowMaximise();
}

export function WindowToggleMaximise() {
  runtime.WindowToggleMaximise();
}

export function WindowUnmaximise() {
  runtime.WindowUnmaximise();
}

export function WindowMinimise() {
  runtime.WindowMinimise();
}

export function WindowUnminimise() {
  runtime.WindowUnminimise();
}

export function WindowSetBackgroundColour(R, G, B, A) {
  runtime.WindowSetBackgroundColour(R, G, B, A);
}

export function ScreenGetAll() {
  return runtime.ScreenGetAll();
}

export function BrowserOpenURL(url) {
  runtime.BrowserOpenURL(url);
}

export function Environment() {
  return runtime.Environment();
}

export function Quit() {
  runtime.Quit();
}

export function Hide() {
  runtime.Hide();
}

export function Show() {
  runtime.Show();
}

export function ClipboardGetText() {
  return runtime.ClipboardGetText();
}

export function ClipboardSetText(text) {
  return runtime.ClipboardSetText(text);
}

export function Callback(incomingEmitFn, target) {
  return runtime.Callback(incomingEmitFn, target);
}

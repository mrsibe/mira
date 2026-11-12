export type FontPreset = "system" | "humanist" | "serif" | "mono";
export type CodeFontPreset = "system" | "modern" | "classic";
export type FontFamilyChoice = `family:${string}`;
export type FontChoice = FontPreset | FontFamilyChoice;
export type CodeFontChoice = CodeFontPreset | FontFamilyChoice;
export type ChatSize = "small" | "default" | "large" | "xlarge";
export type CodeSize = "small" | "default" | "large" | "xlarge";

export type Appearance = {
  interfaceFont: FontChoice;
  chatFont: FontChoice;
  chatSize: ChatSize;
  codeFont: CodeFontChoice;
  codeSize: CodeSize;
};

export type SelectOption<T extends string> = {
  value: T;
  label: string;
  previewFont?: string;
  group?: string;
};

export const appearanceStorageKey = "mira-appearance";

const systemSans =
  '-apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif';
const humanistSans =
  '"Avenir Next", "Segoe UI Variable", "Segoe UI", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif';
const serifSans =
  '"Iowan Old Style", "Source Serif 4", Georgia, "Noto Serif SC", "Songti SC", SimSun, serif';
const systemMono =
  'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace';
const modernMono =
  '"Cascadia Code", "Cascadia Mono", "JetBrains Mono", "Fira Code", "Source Code Pro", ' +
  systemMono;
const classicMono = 'Consolas, "Liberation Mono", Menlo, "Courier New", monospace';

const familyPrefix = "family:";

export const defaultAppearance: Appearance = {
  interfaceFont: "system",
  chatFont: "system",
  chatSize: "default",
  codeFont: "system",
  codeSize: "default",
};

const interfacePresets: readonly FontPreset[] = ["system", "humanist", "serif", "mono"];
const codePresets: readonly CodeFontPreset[] = ["system", "modern", "classic"];

const fontLabels: Record<FontPreset, string> = {
  system: "系统默认",
  humanist: "人文无衬线",
  serif: "衬线",
  mono: "等宽",
};

const codeFontLabels: Record<CodeFontPreset, string> = {
  system: "系统等宽",
  modern: "现代等宽",
  classic: "经典等宽",
};

export const chatSizeOptions: ReadonlyArray<SelectOption<ChatSize>> = [
  { value: "small", label: "14 px" },
  { value: "default", label: "16 px" },
  { value: "large", label: "18 px" },
  { value: "xlarge", label: "20 px" },
];

export const codeSizeOptions: ReadonlyArray<SelectOption<CodeSize>> = [
  { value: "small", label: "12 px" },
  { value: "default", label: "13 px" },
  { value: "large", label: "15 px" },
  { value: "xlarge", label: "17 px" },
];

const chatSizes: Record<ChatSize, string> = {
  small: "14px",
  default: "16px",
  large: "18px",
  xlarge: "20px",
};

const codeSizes: Record<CodeSize, string> = {
  small: "12px",
  default: "13px",
  large: "15px",
  xlarge: "17px",
};

export function buildFontOptions(
  context: "code",
  families: readonly string[],
  current: string,
): ReadonlyArray<SelectOption<CodeFontChoice>>;
export function buildFontOptions(
  context: "interface" | "chat",
  families: readonly string[],
  current: string,
): ReadonlyArray<SelectOption<FontChoice>>;
export function buildFontOptions(
  context: "interface" | "chat" | "code",
  families: readonly string[],
  current: string,
): ReadonlyArray<SelectOption<string>> {
  const code = context === "code";
  const presets: readonly string[] = code ? codePresets : interfacePresets;
  const labels: Record<string, string> = code ? codeFontLabels : fontLabels;
  const options: SelectOption<string>[] = presets.map((value) => ({
    value,
    label: labels[value] ?? value,
    previewFont: resolveFontStack(value, context),
  }));

  const seen = new Set<string>();
  const currentFamily = familyFromChoice(current);
  const installed = families.map(normalizeCustomFont).filter(Boolean);
  if (currentFamily) {
    installed.unshift(currentFamily);
  }
  for (const family of installed) {
    const key = family.toLocaleLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    options.push({
      value: familyChoice(family),
      label: family,
      previewFont: quoteFontName(family),
      group: "系统字体",
    });
  }
  return options;
}

export function resolveFontStack(choice: string, context: "interface" | "chat" | "code"): string {
  const fallback = context === "code" ? systemMono : systemSans;
  const family = familyFromChoice(choice);
  if (family) {
    return `${quoteFontName(family)}, ${fallback}`;
  }
  if (context === "code") {
    if (choice === "modern") return modernMono;
    if (choice === "classic") return classicMono;
    return systemMono;
  }
  if (choice === "humanist") return humanistSans;
  if (choice === "serif") return serifSans;
  if (choice === "mono") return systemMono;
  return systemSans;
}

export function familyFromChoice(choice: string): string {
  if (!choice.startsWith(familyPrefix)) {
    return "";
  }
  return normalizeCustomFont(choice.slice(familyPrefix.length));
}

export function normalizeCustomFont(value: string): string {
  return value
    .trim()
    .replace(/^['"]|['"]$/g, "")
    .split("")
    .filter((character) => {
      const code = character.charCodeAt(0);
      return code >= 32 && code !== 127;
    })
    .join("")
    .slice(0, 120);
}

function familyChoice(family: string): FontFamilyChoice {
  return `${familyPrefix}${family}` as FontFamilyChoice;
}

function quoteFontName(name: string): string {
  return `"${name.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
}

type StoredAppearance = {
  interfaceFont?: unknown;
  chatFont?: unknown;
  chatSize?: unknown;
  codeFont?: unknown;
  codeSize?: unknown;
  interfaceFontCustom?: unknown;
  chatFontCustom?: unknown;
  codeFontCustom?: unknown;
};

export function loadAppearance(): Appearance {
  try {
    const raw = localStorage.getItem(appearanceStorageKey);
    if (!raw) return defaultAppearance;
    return normalizeAppearance(JSON.parse(raw) as StoredAppearance);
  } catch {
    return defaultAppearance;
  }
}

export function saveAppearance(appearance: Appearance): void {
  try {
    localStorage.setItem(appearanceStorageKey, JSON.stringify(appearance));
  } catch {
    // Storage can be unavailable in restricted webviews. The live appearance still applies.
  }
}

export function applyAppearance(
  appearance: Appearance,
  root: HTMLElement = document.documentElement,
): void {
  root.style.setProperty("--font-sans", resolveFontStack(appearance.interfaceFont, "interface"));
  root.style.setProperty("--font-chat", resolveFontStack(appearance.chatFont, "chat"));
  root.style.setProperty("--font-mono", resolveFontStack(appearance.codeFont, "code"));
  root.style.setProperty("--chat-font-base", chatSizes[appearance.chatSize]);
  root.style.setProperty("--code-font-base", codeSizes[appearance.codeSize]);
}

export function applyStoredAppearance(): Appearance {
  const appearance = loadAppearance();
  applyAppearance(appearance);
  return appearance;
}

export function normalizeAppearance(value: StoredAppearance): Appearance {
  return {
    interfaceFont: normalizeFont(value.interfaceFont, interfacePresets, value.interfaceFontCustom),
    chatFont: normalizeFont(value.chatFont, interfacePresets, value.chatFontCustom),
    chatSize: oneOf(value.chatSize, ["small", "default", "large", "xlarge"])
      ? value.chatSize
      : "default",
    codeFont: normalizeFont(value.codeFont, codePresets, value.codeFontCustom),
    codeSize: oneOf(value.codeSize, ["small", "default", "large", "xlarge"])
      ? value.codeSize
      : "default",
  };
}

function normalizeFont<T extends string>(
  value: unknown,
  presets: readonly T[],
  legacyName: unknown,
): T | FontFamilyChoice {
  if (typeof value === "string" && presets.includes(value as T)) {
    return value as T;
  }
  if (typeof value === "string" && value.startsWith(familyPrefix)) {
    const family = familyFromChoice(value);
    if (family) return familyChoice(family);
  }
  if (value === "custom") {
    const family = typeof legacyName === "string" ? normalizeCustomFont(legacyName) : "";
    if (family) return familyChoice(family);
  }
  return "system" as T;
}

function oneOf<T extends string>(value: unknown, allowed: readonly T[]): value is T {
  return typeof value === "string" && allowed.includes(value as T);
}

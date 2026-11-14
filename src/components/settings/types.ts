import type { SearchProvider } from "../../lib/types";

export type SettingsDraft = {
  userName: string;
  webSearchEnabled: boolean;
  searchProvider: SearchProvider;
  tavilyApiKey: string;
  showTavilyKey: boolean;
  responseStyle: string;
  personality: string;
  warmth: string;
  enthusiasm: string;
  emoji: string;
  customInstructions: string;
  memoryEnabled: boolean;
  memorySummaryEnabled: boolean;
  memorySearchEnabled: boolean;
  intelligentUIEnabled: boolean;
};

export type SetDraftField = <K extends keyof SettingsDraft>(
  key: K,
  value: SettingsDraft[K],
) => void;

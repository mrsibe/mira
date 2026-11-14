import { describe, expect, it } from "vitest";
import type { View } from "../mygo";
import { shouldAutoOpenModelWizard } from "./modelWizard";

const settings: View = {
  models: [],
  activeModelId: "",
  userName: "",
  configured: false,
  configPath: "/tmp/mira/config.json",
  webSearchEnabled: true,
  searchProvider: "duckduckgo",
  hasTavilyApiKey: false,
  tavilyApiKeyStorage: "none",
  keychainAvailable: true,
  responseStyle: "concise",
  personality: "friendly",
  warmth: "balanced",
  enthusiasm: "balanced",
  emoji: "none",
  customInstructions: "",
  memoryEnabled: true,
  memorySummaryEnabled: true,
  memorySearchEnabled: true,
  intelligentUIEnabled: true,
};

describe("shouldAutoOpenModelWizard", () => {
  it("opens only when there are no models and setup was not dismissed", () => {
    expect(shouldAutoOpenModelWizard(settings, false)).toBe(true);
    expect(shouldAutoOpenModelWizard(settings, true)).toBe(false);
    expect(
      shouldAutoOpenModelWizard(
        {
          ...settings,
          models: [
            {
              id: "model-1",
              name: "GPT-5",
              provider: "openai",
              providerLabel: "OpenAI",
              baseUrl: "https://api.openai.com/v1",
              model: "gpt-5",
              protocol: "responses",
              configured: true,
              hasApiKey: true,
              apiKeyStorage: "keychain",
            },
          ],
        },
        false,
      ),
    ).toBe(false);
  });
});

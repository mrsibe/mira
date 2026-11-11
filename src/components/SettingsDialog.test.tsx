import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { View } from "../mygo";
import { SettingsDialog } from "./SettingsDialog";

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function showModal() {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function close() {
    this.open = false;
  };
});

const settings: View = {
  baseUrl: "https://api.openai.com/v1",
  model: "gpt-5",
  protocol: "responses",
  userName: "",
  hasApiKey: true,
  apiKeyStorage: "keychain",
  configured: true,
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

describe("SettingsDialog", () => {
  it("saves the intelligent UI switch", async () => {
    const onSave = vi.fn().mockResolvedValue({ ...settings, intelligentUIEnabled: false });
    const onClose = vi.fn();

    render(
      <SettingsDialog
        settings={settings}
        theme="system"
        initialSection="personalization"
        onClose={onClose}
        onSave={onSave}
        onClearKey={vi.fn()}
        onClearTavilyKey={vi.fn()}
        onThemeChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("switch", { name: /智能界面/ }));
    fireEvent.click(screen.getByRole("button", { name: "保存" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledOnce());
    expect(onSave.mock.calls[0]?.[0]).toMatchObject({ intelligentUIEnabled: false });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("saves the selected personality", async () => {
    const onSave = vi.fn().mockResolvedValue({ ...settings, personality: "candid" });

    render(
      <SettingsDialog
        settings={settings}
        theme="system"
        initialSection="personalization"
        onClose={vi.fn()}
        onSave={onSave}
        onClearKey={vi.fn()}
        onClearTavilyKey={vi.fn()}
        onThemeChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("radio", { name: "坦率 · Candid" }));
    fireEvent.click(screen.getByRole("button", { name: "保存" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledOnce());
    expect(onSave.mock.calls[0]?.[0]).toMatchObject({ personality: "candid" });
  });
});

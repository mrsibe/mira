import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { defaultAppearance } from "../lib/appearance";
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
  activeModelId: "model-1",
  userName: "",
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
        appearance={defaultAppearance}
        initialSection="personalization"
        onClose={onClose}
        onSave={onSave}
        onActivateModel={vi.fn()}
        onAddModel={vi.fn()}
        onEditModel={vi.fn()}
        onDeleteModel={vi.fn()}
        onClearTavilyKey={vi.fn()}
        onThemeChange={vi.fn()}
        onAppearanceChange={vi.fn()}
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
        appearance={defaultAppearance}
        initialSection="personalization"
        onClose={vi.fn()}
        onSave={onSave}
        onActivateModel={vi.fn()}
        onAddModel={vi.fn()}
        onEditModel={vi.fn()}
        onDeleteModel={vi.fn()}
        onClearTavilyKey={vi.fn()}
        onThemeChange={vi.fn()}
        onAppearanceChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("radio", { name: "坦率 · Candid" }));
    fireEvent.click(screen.getByRole("button", { name: "保存" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledOnce());
    expect(onSave.mock.calls[0]?.[0]).toMatchObject({ personality: "candid" });
  });

  it("applies appearance changes immediately and offers a done action", () => {
    const onAppearanceChange = vi.fn();

    render(
      <SettingsDialog
        settings={settings}
        theme="system"
        appearance={defaultAppearance}
        initialSection="general"
        onClose={vi.fn()}
        onSave={vi.fn()}
        onActivateModel={vi.fn()}
        onAddModel={vi.fn()}
        onEditModel={vi.fn()}
        onDeleteModel={vi.fn()}
        onClearTavilyKey={vi.fn()}
        onThemeChange={vi.fn()}
        onAppearanceChange={onAppearanceChange}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "对话字号" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "18 px" }));

    expect(onAppearanceChange).toHaveBeenCalledWith(expect.objectContaining({ chatSize: "large" }));
    expect(screen.getByRole("button", { name: "完成" })).toBeTruthy();
  });
});

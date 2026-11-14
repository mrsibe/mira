import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ModelView, View } from "../../mygo";
import { ModelSettings } from "./ModelSettings";

const models: ModelView[] = [
  {
    id: "model-1",
    name: "主力模型",
    provider: "openai",
    providerLabel: "OpenAI",
    baseUrl: "https://api.openai.com/v1",
    model: "gpt-5",
    protocol: "responses",
    configured: true,
    hasApiKey: true,
    apiKeyStorage: "keychain",
  },
  {
    id: "model-2",
    name: "Gemini",
    provider: "google",
    providerLabel: "Google Gemini",
    baseUrl: "",
    model: "gemini-3-flash",
    protocol: "google",
    configured: true,
    hasApiKey: true,
    apiKeyStorage: "environment",
  },
];

const settings: View = {
  models,
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

describe("ModelSettings", () => {
  it("activates a model from the current model dropdown", async () => {
    const onActivateModel = vi.fn().mockResolvedValue({
      ...settings,
      activeModelId: "model-2",
    });

    render(
      <ModelSettings
        settings={settings}
        onActivateModel={onActivateModel}
        onAddModel={vi.fn()}
        onEditModel={vi.fn()}
        onDeleteModel={vi.fn()}
        onError={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "当前模型" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Gemini · Google Gemini" }));

    await waitFor(() => expect(onActivateModel).toHaveBeenCalledWith("model-2"));
    expect(screen.getByText("密钥来自环境变量")).toBeTruthy();
  });

  it("opens add and edit actions", () => {
    const onAddModel = vi.fn();
    const onEditModel = vi.fn();

    render(
      <ModelSettings
        settings={settings}
        onActivateModel={vi.fn()}
        onAddModel={onAddModel}
        onEditModel={onEditModel}
        onDeleteModel={vi.fn()}
        onError={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "新增模型" }));
    expect(onAddModel).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: "编辑 主力模型" }));
    expect(onEditModel).toHaveBeenCalledWith(models[0]);
  });

  it("confirms deletion inline", async () => {
    const onDeleteModel = vi.fn().mockResolvedValue(settings);

    render(
      <ModelSettings
        settings={settings}
        onActivateModel={vi.fn()}
        onAddModel={vi.fn()}
        onEditModel={vi.fn()}
        onDeleteModel={onDeleteModel}
        onError={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "删除 Gemini" }));
    expect(onDeleteModel).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "确认删除" }));

    await waitFor(() => expect(onDeleteModel).toHaveBeenCalledWith("model-2"));
  });

  it("shows the empty state", () => {
    const onAddModel = vi.fn();
    render(
      <ModelSettings
        settings={{ ...settings, models: [], activeModelId: "", configured: false }}
        onActivateModel={vi.fn()}
        onAddModel={onAddModel}
        onEditModel={vi.fn()}
        onDeleteModel={vi.fn()}
        onError={vi.fn()}
      />,
    );

    expect(screen.getByText("还没有模型。添加一个模型后即可开始对话。")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "添加模型" }));
    expect(onAddModel).toHaveBeenCalledOnce();
  });
});

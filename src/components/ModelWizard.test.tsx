import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { ModelView, ProviderInfo, View } from "../mygo";
import { ModelWizard } from "./ModelWizard";

beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function showModal() {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function close() {
    this.open = false;
  };
});

const providers: ProviderInfo[] = [
  {
    id: "openai",
    label: "OpenAI",
    description: "OpenAI 官方接口",
    defaultBaseUrl: "https://api.openai.com/v1",
    baseUrlHint: "https://api.openai.com/v1",
    defaultModel: "gpt-5",
    modelLabel: "模型",
    defaultProtocol: "responses",
    protocols: [
      { value: "responses", label: "Responses" },
      { value: "completions", label: "Chat Completions" },
    ],
    requiresBaseUrl: false,
    requiresApiKey: true,
    localAuthOptional: true,
    apiKeyEnv: "OPENAI_API_KEY",
  },
  {
    id: "google",
    label: "Google Gemini",
    description: "Gemini 原生接口",
    defaultBaseUrl: "",
    baseUrlHint: "留空使用官方端点",
    defaultModel: "gemini-3-flash",
    modelLabel: "模型",
    defaultProtocol: "google",
    protocols: [{ value: "google", label: "Gemini" }],
    requiresBaseUrl: false,
    requiresApiKey: true,
    localAuthOptional: false,
    apiKeyEnv: "GEMINI_API_KEY",
  },
  {
    id: "azure",
    label: "Azure OpenAI",
    description: "Azure v1 Endpoint",
    defaultBaseUrl: "",
    baseUrlHint: "https://<resource>.openai.azure.com/openai/v1",
    defaultModel: "",
    modelLabel: "部署名称",
    defaultProtocol: "responses",
    protocols: [{ value: "responses", label: "Responses" }],
    requiresBaseUrl: true,
    requiresApiKey: true,
    localAuthOptional: false,
    apiKeyEnv: "AZURE_OPENAI_API_KEY",
  },
];

const existing: ModelView = {
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
};

const view: View = {
  models: [existing],
  activeModelId: existing.id,
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

describe("ModelWizard", () => {
  it("selects a provider, tests the connection and adds the model", async () => {
    const onTest = vi.fn().mockResolvedValue(undefined);
    const onSubmit = vi.fn().mockResolvedValue(view);

    render(
      <ModelWizard
        mode="add"
        providers={providers}
        keychainAvailable={true}
        onClose={vi.fn()}
        onTest={onTest}
        onSubmit={onSubmit}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "服务商" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Google Gemini" }));
    fireEvent.click(screen.getByRole("button", { name: "下一步" }));

    expect(screen.getByDisplayValue("gemini-3-flash")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "测试连接" }));
    expect(screen.getByText("请填写 API Key")).toBeTruthy();
    expect(onTest).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText("API Key"), {
      target: { value: "google-key" },
    });
    fireEvent.click(screen.getByRole("button", { name: "测试连接" }));
    await waitFor(() =>
      expect(onTest).toHaveBeenCalledWith(
        expect.objectContaining({
          provider: "google",
          model: "gemini-3-flash",
          protocol: "google",
          apiKey: "google-key",
        }),
      ),
    );

    fireEvent.click(screen.getByRole("button", { name: "添加模型" }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledOnce());
    expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
      provider: "google",
      model: "gemini-3-flash",
      apiKey: "google-key",
    });
  });

  it("skips first-run setup", () => {
    const onSkip = vi.fn();
    render(
      <ModelWizard
        mode="add"
        firstRun
        providers={providers}
        keychainAvailable={true}
        onClose={vi.fn()}
        onSkip={onSkip}
        onTest={vi.fn()}
        onSubmit={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "稍后设置" }));
    expect(onSkip).toHaveBeenCalledOnce();
  });

  it("edits an existing model without requiring the key again", async () => {
    const onTest = vi.fn().mockResolvedValue(undefined);
    const onSubmit = vi.fn().mockResolvedValue(view);
    const onClearKey = vi.fn().mockResolvedValue(view);

    render(
      <ModelWizard
        mode="edit"
        providers={providers}
        initialModel={existing}
        keychainAvailable={true}
        onClose={vi.fn()}
        onTest={onTest}
        onSubmit={onSubmit}
        onClearKey={onClearKey}
      />,
    );

    expect(screen.getByText("编辑模型")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /更换/ })).toBeNull();
    fireEvent.change(screen.getByLabelText("模型"), { target: { value: "gpt-5.1" } });
    fireEvent.click(screen.getByRole("button", { name: "保存修改" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledOnce());
    expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
      id: existing.id,
      model: "gpt-5.1",
      apiKey: "",
    });

    fireEvent.click(screen.getByRole("button", { name: "清除已保存的密钥" }));
    await waitFor(() => expect(onClearKey).toHaveBeenCalledWith(existing.id));
  });

  it("changes the API protocol through the shared select control", async () => {
    const onSubmit = vi.fn().mockResolvedValue(view);

    render(
      <ModelWizard
        mode="edit"
        providers={providers}
        initialModel={existing}
        keychainAvailable={true}
        onClose={vi.fn()}
        onTest={vi.fn()}
        onSubmit={onSubmit}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "API 协议" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Chat Completions" }));
    fireEvent.click(screen.getByRole("button", { name: "保存修改" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledOnce());
    expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
      protocol: "completions",
    });
  });
});

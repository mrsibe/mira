import { Channel } from "mygo-runtime";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BrandMark } from "./components/BrandMark";
import { Composer } from "./components/Composer";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { ProjectDialog } from "./components/ProjectDialog";
import { SettingsDialog } from "./components/SettingsDialog";
import { Sidebar } from "./components/Sidebar";
import { Thread } from "./components/Thread";
import { Welcome } from "./components/Welcome";
import { WindowTitlebar } from "./components/WindowTitlebar";
import { type Appearance, applyAppearance, loadAppearance, saveAppearance } from "./lib/appearance";
import { applyChatEvent, emptyRun } from "./lib/run";
import type { ReasoningEffort, RunView, SettingsSection, ThemeMode } from "./lib/types";
import {
  type ChatEvent,
  type Conversation,
  type Input,
  Mira,
  MiraTheme,
  type Project,
  type UIMessage,
  type View,
} from "./mygo";
import { Toast } from "./ui";

const reasoningStorageKey = "mira-reasoning-effort";

type ProjectDialogState =
  | { mode: "create"; afterMoveConversationID?: string }
  | { mode: "rename"; project: Project };

function initialReasoningEffort(): ReasoningEffort {
  const saved = localStorage.getItem(reasoningStorageKey);
  return saved === "low" || saved === "medium" || saved === "high" ? saved : "auto";
}

export default function App() {
  const [settings, setSettings] = useState<View | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [messages, setMessages] = useState<Record<string, UIMessage[]>>({});
  const [runs, setRuns] = useState<Record<string, RunView>>({});
  const [activeId, setActiveId] = useState<string | null>(null);
  const [runningId, setRunningId] = useState<string | null>(null);
  const [scrollTarget, setScrollTarget] = useState<{
    conversationId: string;
    messageIndex: number;
  } | null>(null);
  const [loading, setLoading] = useState(true);
  const [bootstrapError, setBootstrapError] = useState("");
  const [configError, setConfigError] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsSection, setSettingsSection] = useState<SettingsSection>("general");
  const [sidebarOpen, setSidebarOpen] = useState(() => window.innerWidth >= 900);
  const [search, setSearch] = useState("");
  const [composer, setComposer] = useState("");
  const [newConversationProjectID, setNewConversationProjectID] = useState("");
  const [reasoningEffort, setReasoningEffort] = useState<ReasoningEffort>(initialReasoningEffort);
  const [toast, setToast] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<Conversation | null>(null);
  const [confirmDeleteProject, setConfirmDeleteProject] = useState<Project | null>(null);
  const [projectDialog, setProjectDialog] = useState<ProjectDialogState | null>(null);
  const toastTimer = useRef<number | undefined>(undefined);

  const { theme, setTheme } = useTheme();
  const { appearance, setAppearance } = useAppearance();

  useEffect(() => {
    localStorage.setItem(reasoningStorageKey, reasoningEffort);
  }, [reasoningEffort]);

  const showToast = useCallback((message: string) => {
    setToast(message);
    window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(null), 2600);
  }, []);

  const openSettings = useCallback((section: SettingsSection) => {
    setSettingsSection(section);
    setSettingsOpen(true);
  }, []);

  const loadConversation = useCallback(async (id: string) => {
    const detail = await Mira.getConversation(id);
    setMessages((previous) => ({ ...previous, [id]: detail.messages }));
    setConversations((previous) =>
      previous.map((item) => (item.id === id ? detail.conversation : item)),
    );
    return detail;
  }, []);

  const refreshConversations = useCallback(async () => {
    const items = await Mira.listConversations();
    setConversations(items);
    return items;
  }, []);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const data = await Mira.bootstrap();
        if (!alive) return;
        setSettings(data.settings);
        setConversations(data.conversations);
        setProjects(data.projects);
        setConfigError(data.configError ?? "");
        const first = data.conversations[0];
        if (first) {
          setActiveId(first.id);
          await loadConversation(first.id);
        }
      } catch (error) {
        if (alive) setBootstrapError(error instanceof Error ? error.message : "启动失败");
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [loadConversation]);

  const activeConversation = useMemo(
    () => conversations.find((item) => item.id === activeId),
    [conversations, activeId],
  );
  const activeMessages = activeId ? (messages[activeId] ?? []) : [];
  const activeRun = activeId ? runs[activeId] : undefined;
  const activeRunning = activeId !== null && runningId === activeId;
  const showWelcome = activeMessages.length === 0 && !activeRun;
  const displayName = settings?.userName?.trim() ?? "";
  const welcomeGreeting = displayName ? `${displayName}，今天聊点什么？` : "今天聊点什么？";

  const sendMessage = useCallback(
    async (raw: string) => {
      const content = raw.trim();
      if (!content) return;
      if (!settings?.configured) {
        openSettings("model");
        return;
      }
      if (runningId) {
        showToast("正在生成回复，请先停止当前回复");
        return;
      }

      let conversationId = activeId;
      if (!conversationId) {
        try {
          const conversation = await Mira.createConversation(newConversationProjectID);
          conversationId = conversation.id;
          setConversations((previous) => [conversation, ...previous]);
          setMessages((previous) => ({ ...previous, [conversation.id]: [] }));
          setActiveId(conversation.id);
          setNewConversationProjectID("");
        } catch (error) {
          showToast(error instanceof Error ? error.message : "无法创建会话");
          return;
        }
      }

      const id = conversationId;
      const userIndex = messages[id]?.length ?? 0;
      const userMessage: UIMessage = {
        index: userIndex,
        role: "user",
        blocks: [{ type: "text", text: content }],
      };
      setMessages((previous) => ({
        ...previous,
        [id]: [...(previous[id] ?? []), userMessage],
      }));
      setScrollTarget({ conversationId: id, messageIndex: userIndex });
      setComposer("");
      setRuns((previous) => ({ ...previous, [id]: emptyRun() }));
      setRunningId(id);

      try {
        const channel = new Channel<ChatEvent>();
        const done = Mira.send(
          id,
          content,
          { reasoningEffort: reasoningEffort === "auto" ? "" : reasoningEffort },
          channel,
        );
        for await (const event of channel) {
          setRuns((previous) => ({
            ...previous,
            [id]: applyChatEvent(previous[id], event),
          }));
        }
        await done;
      } catch (error) {
        const message = error instanceof Error ? error.message : "发送失败";
        setRuns((previous) => ({
          ...previous,
          [id]: { ...(previous[id] ?? emptyRun()), status: "error", error: message },
        }));
      } finally {
        setRunningId((current) => (current === id ? null : current));
        let loaded = false;
        try {
          await loadConversation(id);
          loaded = true;
        } catch {
          // The next bootstrap or conversation switch will retry.
        }
        if (loaded) {
          setRuns((previous) => {
            const current = previous[id];
            if (!current) return previous;
            if (current.status === "error") {
              return {
                ...previous,
                [id]: {
                  ...current,
                  parts: [],
                  nextPartId: 0,
                },
              };
            }
            const next = { ...previous };
            delete next[id];
            return next;
          });
        }
        try {
          await refreshConversations();
        } catch {
          // Keep the current sidebar snapshot.
        }
      }
    },
    [
      activeId,
      loadConversation,
      messages,
      newConversationProjectID,
      reasoningEffort,
      refreshConversations,
      runningId,
      settings,
      showToast,
      openSettings,
    ],
  );

  const handleAskAgent = useCallback(
    (message: string) => {
      void sendMessage(message);
    },
    [sendMessage],
  );

  function newChat(projectID = "") {
    if (runningId) {
      showToast("正在生成回复，请先停止当前回复");
      return;
    }
    setScrollTarget(null);
    setActiveId(null);
    setComposer("");
    setNewConversationProjectID(projectID);
    if (window.innerWidth < 900) setSidebarOpen(false);
  }

  async function selectConversation(id: string) {
    setScrollTarget(null);
    setActiveId(id);
    setNewConversationProjectID("");
    if (window.innerWidth < 900) setSidebarOpen(false);
    if (!messages[id]) {
      try {
        await loadConversation(id);
      } catch (error) {
        showToast(error instanceof Error ? error.message : "无法加载会话");
      }
    }
  }

  async function renameConversation(id: string, title: string) {
    try {
      await Mira.renameConversation(id, title);
      setConversations((previous) =>
        previous.map((item) => (item.id === id ? { ...item, title } : item)),
      );
    } catch (error) {
      showToast(error instanceof Error ? error.message : "重命名失败");
    }
  }

  async function createProject(name: string) {
    const project = await Mira.createProject(name);
    setProjects((previous) => [...previous, project]);
    return project;
  }

  async function renameProject(id: string, name: string) {
    await Mira.renameProject(id, name);
    setProjects((previous) =>
      previous.map((project) => (project.id === id ? { ...project, name } : project)),
    );
  }

  async function moveConversation(id: string, projectID: string) {
    try {
      await Mira.moveConversation(id, projectID);
      setConversations((previous) =>
        previous.map((conversation) =>
          conversation.id === id
            ? { ...conversation, projectId: projectID || undefined }
            : conversation,
        ),
      );
      const destination = projects.find((project) => project.id === projectID);
      showToast(destination ? `已移动到“${destination.name}”` : "已移动到未分类");
      return true;
    } catch (error) {
      showToast(error instanceof Error ? error.message : "移动失败");
      return false;
    }
  }

  async function deleteProject() {
    const target = confirmDeleteProject;
    if (!target) return;
    setConfirmDeleteProject(null);
    try {
      await Mira.deleteProject(target.id);
      const removedIDs = new Set(
        conversations
          .filter((conversation) => conversation.projectId === target.id)
          .map((conversation) => conversation.id),
      );
      const remaining = conversations.filter((conversation) => !removedIDs.has(conversation.id));
      setProjects((previous) => previous.filter((project) => project.id !== target.id));
      setConversations(remaining);
      setMessages((previous) => removeRecords(previous, removedIDs));
      setRuns((previous) => removeRecords(previous, removedIDs));
      if (newConversationProjectID === target.id) setNewConversationProjectID("");
      if (activeId && removedIDs.has(activeId)) {
        const next = remaining[0];
        setActiveId(next?.id ?? null);
        if (next) await loadConversation(next.id);
      }
    } catch (error) {
      showToast(error instanceof Error ? error.message : "删除项目失败");
    }
  }

  async function deleteConversation() {
    const target = confirmDelete;
    if (!target) return;
    setConfirmDelete(null);
    try {
      await Mira.deleteConversation(target.id);
      const remaining = conversations.filter((item) => item.id !== target.id);
      setConversations(remaining);
      setMessages((previous) => {
        const next = { ...previous };
        delete next[target.id];
        return next;
      });
      setRuns((previous) => {
        const next = { ...previous };
        delete next[target.id];
        return next;
      });
      if (activeId === target.id) {
        const next = remaining[0];
        setActiveId(next?.id ?? null);
        if (next) await loadConversation(next.id);
      }
    } catch (error) {
      showToast(error instanceof Error ? error.message : "删除失败");
    }
  }

  async function saveSettings(input: Input) {
    const saved = await Mira.saveSettings(input);
    setSettings(saved);
    setConfigError("");
    return saved;
  }

  async function clearKey() {
    const saved = await Mira.clearAPIKey();
    setSettings(saved);
    return saved;
  }

  async function clearTavilyKey() {
    const saved = await Mira.clearTavilyAPIKey();
    setSettings(saved);
    return saved;
  }

  if (loading) {
    return (
      <div className="boot-screen">
        <BrandMark className="boot-mark" />
        <span>正在打开 Mira…</span>
      </div>
    );
  }

  if (bootstrapError) {
    return (
      <div className="boot-screen error">
        <BrandMark className="boot-mark" />
        <h1>Mira 无法启动</h1>
        <p>{bootstrapError}</p>
      </div>
    );
  }

  return (
    <div className="app-frame">
      <WindowTitlebar
        sidebarOpen={sidebarOpen}
        title={activeConversation?.title ?? ""}
        onToggleSidebar={() => setSidebarOpen((open) => !open)}
      />
      <div className={`app-shell ${sidebarOpen ? "" : "sidebar-collapsed"}`}>
        <button
          type="button"
          className={`sidebar-backdrop ${sidebarOpen ? "visible" : ""}`}
          aria-label="关闭侧栏"
          aria-hidden={!sidebarOpen}
          tabIndex={sidebarOpen ? 0 : -1}
          onClick={() => setSidebarOpen(false)}
        />
        <Sidebar
          conversations={conversations}
          projects={projects}
          activeId={activeId}
          search={search}
          onSearch={setSearch}
          onSelect={selectConversation}
          onNew={() => newChat()}
          onNewInProject={(projectID) => newChat(projectID)}
          onRename={renameConversation}
          onDelete={(id) => {
            const conversation = conversations.find((item) => item.id === id);
            if (conversation) setConfirmDelete(conversation);
          }}
          onMove={moveConversation}
          onCreateProject={(afterMoveConversationID) =>
            setProjectDialog({ mode: "create", afterMoveConversationID })
          }
          onRenameProject={(project) => setProjectDialog({ mode: "rename", project })}
          onDeleteProject={setConfirmDeleteProject}
          onSettings={() => openSettings("general")}
        />

        <main className="main">
          {configError && (
            <div className="config-banner">
              <span>{configError}</span>
              <button type="button" onClick={() => openSettings("model")}>
                打开设置
              </button>
            </div>
          )}

          {showWelcome ? (
            <Welcome greeting={welcomeGreeting}>
              <Composer
                centered
                value={composer}
                running={activeRunning}
                disabled={runningId !== null && !activeRunning}
                reasoningEffort={reasoningEffort}
                onChange={setComposer}
                onReasoningEffortChange={setReasoningEffort}
                onSend={() => sendMessage(composer)}
                onStop={() => {
                  if (activeId) void Mira.stop(activeId);
                }}
                onAttach={() => showToast("附件将在后续版本支持")}
              />
            </Welcome>
          ) : (
            <>
              <Thread
                messages={activeMessages}
                run={activeRun}
                conversationId={activeId ?? ""}
                scrollTarget={scrollTarget}
                intelligentUIEnabled={settings?.intelligentUIEnabled ?? true}
                onAskAgent={handleAskAgent}
              />
              <Composer
                value={composer}
                running={activeRunning}
                disabled={runningId !== null && !activeRunning}
                reasoningEffort={reasoningEffort}
                onChange={setComposer}
                onReasoningEffortChange={setReasoningEffort}
                onSend={() => sendMessage(composer)}
                onStop={() => {
                  if (activeId) void Mira.stop(activeId);
                }}
                onAttach={() => showToast("附件将在后续版本支持")}
              />
            </>
          )}
        </main>
      </div>

      {settings && settingsOpen && (
        <SettingsDialog
          settings={settings}
          theme={theme}
          appearance={appearance}
          initialSection={settingsSection}
          onClose={() => setSettingsOpen(false)}
          onSave={saveSettings}
          onClearKey={clearKey}
          onClearTavilyKey={clearTavilyKey}
          onThemeChange={setTheme}
          onAppearanceChange={setAppearance}
        />
      )}

      {confirmDelete && (
        <ConfirmDialog
          title="删除对话？"
          message={`“${confirmDelete.title}”及其消息将从本机删除。`}
          onConfirm={deleteConversation}
          onClose={() => setConfirmDelete(null)}
        />
      )}

      {confirmDeleteProject && (
        <ConfirmDialog
          title="删除项目？"
          message={projectDeleteMessage(confirmDeleteProject, conversations)}
          confirmLabel="删除项目"
          onConfirm={deleteProject}
          onClose={() => setConfirmDeleteProject(null)}
        />
      )}

      {projectDialog && (
        <ProjectDialog
          title={projectDialog.mode === "create" ? "新建项目" : "重命名项目"}
          initialName={projectDialog.mode === "rename" ? projectDialog.project.name : ""}
          submitLabel={projectDialog.mode === "create" ? "创建" : "保存"}
          onClose={() => setProjectDialog(null)}
          onSubmit={async (name) => {
            if (projectDialog.mode === "rename") {
              await renameProject(projectDialog.project.id, name);
            } else {
              const project = await createProject(name);
              if (projectDialog.afterMoveConversationID) {
                await moveConversation(projectDialog.afterMoveConversationID, project.id);
              }
            }
            setProjectDialog(null);
          }}
        />
      )}

      {toast && <Toast>{toast}</Toast>}
    </div>
  );
}

function removeRecords<T>(records: Record<string, T>, removedIDs: Set<string>) {
  const next = { ...records };
  for (const id of removedIDs) delete next[id];
  return next;
}

function projectDeleteMessage(project: Project, conversations: Conversation[]) {
  const count = conversations.filter((item) => item.projectId === project.id).length;
  if (count === 0) {
    return `“${project.name}”将被删除。`;
  }
  return `“${project.name}”和其中的 ${count} 个对话及其消息将从本机永久删除。`;
}

function useTheme() {
  const [theme, setTheme] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem("mira-theme");
    return saved === "light" || saved === "dark" || saved === "system" ? saved : "system";
  });
  const [systemDark, setSystemDark] = useState(
    () => window.matchMedia("(prefers-color-scheme: dark)").matches,
  );

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const listener = (event: MediaQueryListEvent) => setSystemDark(event.matches);
    media.addEventListener("change", listener);
    return () => media.removeEventListener("change", listener);
  }, []);

  const isDark = theme === "dark" || (theme === "system" && systemDark);

  useEffect(() => {
    document.documentElement.dataset.theme = isDark ? "dark" : "light";
    document.documentElement.style.colorScheme = isDark ? "dark" : "light";
    localStorage.setItem("mira-theme", theme);
  }, [isDark, theme]);

  useEffect(() => {
    void MiraTheme.setTheme(theme).catch((error: unknown) => {
      console.warn("无法同步原生窗口主题", error);
    });
  }, [theme]);

  return {
    theme,
    setTheme,
  };
}

function useAppearance() {
  const [appearance, setAppearance] = useState<Appearance>(loadAppearance);

  useEffect(() => {
    applyAppearance(appearance);
    saveAppearance(appearance);
  }, [appearance]);

  return {
    appearance,
    setAppearance,
  };
}

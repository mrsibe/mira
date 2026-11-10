import { useCallback, useEffect, useRef, useState } from "react";
import { type MemoryView, Mira } from "../../mygo";

type MemoryEditorOptions = {
  active: boolean;
  onError: (message: string) => void;
};

export type MemoryEditor = ReturnType<typeof useMemoryEditor>;

export function useMemoryEditor({ active, onError }: MemoryEditorOptions) {
  const [memory, setMemory] = useState<MemoryView | null>(null);
  const [manualDraft, setManualDraft] = useState("");
  const [summaryDraft, setSummaryDraft] = useState("");
  const [dirty, setDirty] = useState(false);
  const [loading, setLoading] = useState(false);
  const [action, setAction] = useState(false);
  const [conflict, setConflict] = useState(false);
  const revision = useRef(0);
  const dirtyRef = useRef(false);

  const adopt = useCallback((next: MemoryView, forceDraft = false) => {
    setMemory(next);
    if (forceDraft || !dirtyRef.current) {
      setManualDraft(next.manual);
      setSummaryDraft(next.summary);
      revision.current = next.revision;
      dirtyRef.current = false;
      setDirty(false);
      setConflict(false);
    }
  }, []);

  useEffect(() => {
    if (!active) return;
    let alive = true;
    setLoading(true);

    async function poll() {
      try {
        const next = await Mira.getMemory();
        if (!alive) return;
        adopt(next);
        setLoading(false);
      } catch (error) {
        if (!alive) return;
        setLoading(false);
        onError(error instanceof Error ? error.message : "无法读取记忆");
      }
    }

    void poll();
    const timer = window.setInterval(() => void poll(), 2500);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [active, adopt, onError]);

  const markDirty = useCallback(() => {
    dirtyRef.current = true;
    setDirty(true);
    setConflict(false);
  }, []);

  const saveDraft = useCallback(async () => {
    try {
      const saved = await Mira.saveMemory({
        manual: manualDraft,
        summary: summaryDraft,
        revision: revision.current,
      });
      adopt(saved, true);
      return saved;
    } catch (error) {
      setConflict(isRevisionConflict(error));
      throw error;
    }
  }, [adopt, manualDraft, summaryDraft]);

  const regenerate = useCallback(async () => {
    onError("");
    setAction(true);
    try {
      if (dirtyRef.current) await saveDraft();
      adopt(await Mira.regenerateMemory(), true);
    } catch (error) {
      setConflict(isRevisionConflict(error));
      onError(error instanceof Error ? error.message : "更新摘要失败");
      try {
        adopt(await Mira.getMemory());
      } catch {
        // Keep the current snapshot when the status refresh also fails.
      }
    } finally {
      setAction(false);
    }
  }, [adopt, onError, saveDraft]);

  const clear = useCallback(async () => {
    onError("");
    setAction(true);
    try {
      adopt(await Mira.clearMemory(), true);
    } catch (error) {
      onError(error instanceof Error ? error.message : "清空记忆失败");
    } finally {
      setAction(false);
    }
  }, [adopt, onError]);

  const reload = useCallback(async () => {
    onError("");
    dirtyRef.current = false;
    setDirty(false);
    try {
      adopt(await Mira.getMemory(), true);
    } catch (error) {
      onError(error instanceof Error ? error.message : "无法读取记忆");
    }
  }, [adopt, onError]);

  return {
    memory,
    manualDraft,
    summaryDraft,
    dirty,
    loading,
    action,
    conflict,
    setManualDraft,
    setSummaryDraft,
    markDirty,
    saveDraft,
    regenerate,
    clear,
    reload,
  };
}

function isRevisionConflict(error: unknown) {
  return error instanceof Error && error.message.includes("记忆已在后台更新");
}

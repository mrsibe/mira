import { type FormEvent, useEffect, useId, useRef, useState } from "react";
import { Button, Modal, Notice } from "../ui";

type ProjectDialogProps = {
  title: string;
  initialName?: string;
  submitLabel: string;
  onClose: () => void;
  onSubmit: (name: string) => Promise<void>;
};

export function ProjectDialog({
  title,
  initialName = "",
  submitLabel,
  onClose,
  onSubmit,
}: ProjectDialogProps) {
  const titleID = useId();
  const inputID = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [name, setName] = useState(initialName);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const frame = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (saving) return;
    const trimmed = name.trim();
    if (!trimmed) {
      setError("项目名称不能为空");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await onSubmit(trimmed);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "保存项目失败");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      className="project-modal"
      labelledBy={titleID}
      onClose={() => {
        if (!saving) onClose();
      }}
    >
      <form className="project-form" onSubmit={submit}>
        <header className="modal-head">
          <h2 id={titleID}>{title}</h2>
        </header>
        <div className="project-form-body">
          <label className="sr-only" htmlFor={inputID}>
            项目名称
          </label>
          <input
            ref={inputRef}
            id={inputID}
            value={name}
            maxLength={48}
            disabled={saving}
            placeholder="项目名称"
            onChange={(event) => setName(event.target.value)}
          />
          {error && <Notice tone="error">{error}</Notice>}
        </div>
        <footer className="modal-actions">
          <span className="spacer" />
          <Button disabled={saving} onClick={onClose}>
            取消
          </Button>
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? "保存中…" : submitLabel}
          </Button>
        </footer>
      </form>
    </Modal>
  );
}

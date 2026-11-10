import { ArrowUp, Check, Plus, Square } from "lucide-react";
import { type KeyboardEvent as ReactKeyboardEvent, useEffect, useRef, useState } from "react";
import type { ReasoningEffort } from "../lib/types";

type ComposerProps = {
  value: string;
  running: boolean;
  disabled: boolean;
  centered?: boolean;
  placeholder?: string;
  reasoningEffort: ReasoningEffort;
  onChange: (value: string) => void;
  onReasoningEffortChange: (value: ReasoningEffort) => void;
  onSend: () => void;
  onStop: () => void;
  onAttach: () => void;
};

const reasoningOptions = [
  { value: "auto", label: "自动" },
  { value: "low", label: "低" },
  { value: "medium", label: "中" },
  { value: "high", label: "高" },
] as const satisfies ReadonlyArray<{ value: ReasoningEffort; label: string }>;

export function Composer({
  value,
  running,
  disabled,
  centered = false,
  placeholder = "询问任何问题",
  reasoningEffort,
  onChange,
  onReasoningEffortChange,
  onSend,
  onStop,
  onAttach,
}: ComposerProps) {
  const ref = useRef<HTMLTextAreaElement>(null);
  const reasoningRef = useRef<HTMLDivElement>(null);
  const reasoningButtonRef = useRef<HTMLButtonElement>(null);
  const reasoningMenuRef = useRef<HTMLDivElement>(null);
  const [reasoningOpen, setReasoningOpen] = useState(false);

  useEffect(() => {
    ref.current?.focus();
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies: the controlled value changes require remeasurement
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    element.style.height = "0px";
    element.style.height = `${Math.min(element.scrollHeight, 196)}px`;
  }, [value]);

  useEffect(() => {
    if (!reasoningOpen) return;

    const selected = reasoningMenuRef.current?.querySelector<HTMLButtonElement>(
      '[role="menuitemradio"][aria-checked="true"]',
    );
    selected?.focus();

    function onPointerDown(event: PointerEvent) {
      const target = event.target;
      if (target instanceof Node && reasoningRef.current?.contains(target)) return;
      setReasoningOpen(false);
    }

    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      setReasoningOpen(false);
      reasoningButtonRef.current?.focus();
    }

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [reasoningOpen]);

  const canSend = value.trim().length > 0 && !disabled && !running;
  const currentReasoning =
    reasoningOptions.find((option) => option.value === reasoningEffort) ?? reasoningOptions[0];

  function onReasoningKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
    if (event.key === "Tab") {
      setReasoningOpen(false);
      reasoningButtonRef.current?.focus();
      return;
    }
    const items = Array.from(
      reasoningMenuRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitemradio"]') ?? [],
    );
    if (items.length === 0) return;
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    let next = current;
    if (event.key === "ArrowDown") next = (current + 1) % items.length;
    else if (event.key === "ArrowUp") next = (current - 1 + items.length) % items.length;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = items.length - 1;
    else return;
    event.preventDefault();
    items[next]?.focus();
  }

  function selectReasoning(next: ReasoningEffort) {
    onReasoningEffortChange(next);
    setReasoningOpen(false);
    reasoningButtonRef.current?.focus();
  }

  return (
    <div className={`composer-zone ${centered ? "centered" : ""}`}>
      <div className={`composer ${running ? "is-running" : ""}`}>
        <button type="button" className="composer-tool" onClick={onAttach} title="附件">
          <Plus size={19} />
        </button>
        <div className="composer-input">
          <textarea
            ref={ref}
            value={value}
            rows={1}
            aria-label="消息"
            onChange={(event) => onChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
                event.preventDefault();
                if (canSend) onSend();
              }
            }}
          />
          {!value && (
            <span className="composer-placeholder" aria-hidden="true">
              {placeholder}
            </span>
          )}
        </div>
        <div className="composer-reasoning" ref={reasoningRef}>
          <button
            ref={reasoningButtonRef}
            type="button"
            className="composer-reasoning-button"
            aria-label={`思维强度：${currentReasoning.label}`}
            aria-haspopup="menu"
            aria-expanded={reasoningOpen}
            aria-controls="reasoning-menu"
            title={`思维强度：${currentReasoning.label}`}
            onClick={() => setReasoningOpen((open) => !open)}
          >
            <span className="composer-reasoning-label">{currentReasoning.label}</span>
          </button>
          {reasoningOpen && (
            <div
              ref={reasoningMenuRef}
              id="reasoning-menu"
              className="composer-reasoning-menu"
              role="menu"
              aria-label="思维强度"
              onKeyDown={onReasoningKeyDown}
            >
              <span className="composer-reasoning-heading" aria-hidden="true">
                思维强度
              </span>
              {reasoningOptions.map((option) => {
                const selected = reasoningEffort === option.value;
                return (
                  <button
                    key={option.value}
                    type="button"
                    role="menuitemradio"
                    aria-checked={selected}
                    tabIndex={selected ? 0 : -1}
                    className={`composer-reasoning-option ${selected ? "selected" : ""}`}
                    onClick={() => selectReasoning(option.value)}
                  >
                    <span>{option.label}</span>
                    {selected && <Check size={14} />}
                  </button>
                );
              })}
            </div>
          )}
        </div>
        {running ? (
          <button type="button" className="send-button stop" onClick={onStop} title="停止生成">
            <Square size={14} fill="currentColor" />
          </button>
        ) : (
          <button
            type="button"
            className="send-button"
            disabled={!canSend}
            onClick={onSend}
            title="发送"
          >
            <ArrowUp size={18} strokeWidth={2.4} />
          </button>
        )}
      </div>
    </div>
  );
}

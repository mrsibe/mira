import {
  type KeyboardEvent,
  type ReactNode,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { cx } from "./cx";

type PopupMenuProps = {
  anchor: HTMLElement;
  label: string;
  className?: string;
  onClose: () => void;
  children: ReactNode;
};

type Position = {
  left: number;
  top: number;
};

const edge = 8;
const gap = 6;

export function PopupMenu({ anchor, label, className, onClose, children }: PopupMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState<Position>({ left: 0, top: 0 });
  const [ready, setReady] = useState(false);

  useLayoutEffect(() => {
    const menu = menuRef.current;
    if (!menu) return;
    const bounds = anchor.getBoundingClientRect();
    const box = menu.getBoundingClientRect();
    const left = Math.min(
      Math.max(edge, bounds.right - box.width),
      Math.max(edge, window.innerWidth - box.width - edge),
    );
    let top = bounds.bottom + gap;
    if (top + box.height > window.innerHeight - edge) {
      top = Math.max(edge, bounds.top - box.height - gap);
    }
    setPosition({ left, top });
    setReady(true);
  }, [anchor]);

  useEffect(() => {
    const menu = menuRef.current;
    if (!menu) return;
    function onPointerDown(event: PointerEvent) {
      const target = event.target;
      if (!(target instanceof Node)) return;
      if (menu?.contains(target) || anchor.contains(target)) return;
      onClose();
    }

    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [anchor, onClose]);

  useEffect(() => {
    const menu = menuRef.current;
    if (!menu) return;
    menuItems(menu)[0]?.focus();
  }, []);

  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      anchor.focus();
      onClose();
      return;
    }
    if (event.key === "Tab") {
      anchor.focus();
      onClose();
      return;
    }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    const items = menuItems(event.currentTarget);
    if (items.length === 0) return;
    const current = items.indexOf(document.activeElement as HTMLElement);
    let next = current;
    if (event.key === "ArrowDown") next = (current + 1 + items.length) % items.length;
    if (event.key === "ArrowUp") next = (current - 1 + items.length) % items.length;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = items.length - 1;
    event.preventDefault();
    items[next]?.focus();
  }

  return createPortal(
    <div
      ref={menuRef}
      className={cx("popup-menu", ready ? "ready" : undefined, className)}
      role="menu"
      aria-label={label}
      style={position}
      onKeyDown={onKeyDown}
    >
      {children}
    </div>,
    document.body,
  );
}

function menuItems(menu: HTMLElement) {
  return Array.from(
    menu.querySelectorAll<HTMLElement>(
      '[role="menuitem"]:not(:disabled), [role="menuitemradio"]:not(:disabled)',
    ),
  );
}

import { useCallback, useLayoutEffect, useRef, useState } from "react";
import type { RunView } from "../../lib/types";
import type { UIMessage } from "../../mygo";

type ScrollTarget = { conversationId: string; messageIndex: number };
type ScrollMode = "follow" | "anchor" | "free";

const bottomThreshold = 24;

export function anchorSpacerHeight(
  clientHeight: number,
  contentHeight: number,
  targetTop: number,
): number {
  return Math.max(0, Math.ceil(clientHeight - (contentHeight - targetTop)));
}

function findTargetMessage(
  scroll: HTMLElement,
  target: ScrollTarget | null | undefined,
  conversationId: string,
): HTMLElement | null {
  if (!target || target.conversationId !== conversationId) return null;
  return scroll.querySelector<HTMLElement>(`[data-message-index="${target.messageIndex}"]`);
}

function threadPaddingTop(message: HTMLElement): number {
  const thread = message.closest<HTMLElement>(".thread");
  if (!thread) return 0;
  const padding = Number.parseFloat(getComputedStyle(thread).paddingTop);
  return Number.isFinite(padding) ? padding : 0;
}

function spacerPixelHeight(spacer: HTMLElement | null): number {
  if (!spacer) return 0;
  if (spacer.offsetHeight > 0) return spacer.offsetHeight;
  const height = Number.parseFloat(spacer.style.height);
  return Number.isFinite(height) ? height : 0;
}

export function useThreadScroll({
  messages,
  run,
  conversationId,
  scrollTarget,
}: {
  messages: UIMessage[];
  run?: RunView;
  conversationId: string;
  scrollTarget?: ScrollTarget | null;
}) {
  const scrollRef = useRef<HTMLElement>(null);
  const spacerRef = useRef<HTMLDivElement>(null);
  const modeRef = useRef<ScrollMode>("follow");
  const conversationRef = useRef(conversationId);
  const targetRef = useRef<ScrollTarget | null>(null);
  const anchorTopRef = useRef(0);
  const freezeBaselineRef = useRef<number | null>(null);
  const [nearBottom, setNearBottom] = useState(true);

  const notifyNearBottom = useCallback((near: boolean) => {
    setNearBottom((current) => (current === near ? current : near));
  }, []);

  const clearSpacer = useCallback(() => {
    if (spacerRef.current) spacerRef.current.style.height = "0px";
  }, []);

  const setScrollTop = useCallback((element: HTMLElement, top: number) => {
    anchorTopRef.current = top;
    element.scrollTop = top;
  }, []);

  const maintainFreezeRange = useCallback((scroll: HTMLElement) => {
    const spacer = spacerRef.current;
    const baseline = freezeBaselineRef.current;
    if (!spacer || baseline === null) return;

    const currentSpacerHeight = spacerPixelHeight(spacer);
    const naturalHeight = scroll.scrollHeight - currentSpacerHeight;
    const spacerHeight = Math.max(0, baseline - naturalHeight);
    if (Math.abs(currentSpacerHeight - spacerHeight) > 1) {
      spacer.style.height = `${spacerHeight}px`;
    }
  }, []);

  const applyAnchor = useCallback(
    (scroll: HTMLElement, target: HTMLElement) => {
      const spacer = spacerRef.current;
      const currentSpacerHeight = spacerPixelHeight(spacer);
      const containerRect = scroll.getBoundingClientRect();
      const targetRect = target.getBoundingClientRect();
      const targetTop = Math.max(
        0,
        targetRect.top - containerRect.top + scroll.scrollTop - threadPaddingTop(target),
      );
      const naturalHeight = scroll.scrollHeight - currentSpacerHeight;
      const spacerHeight = anchorSpacerHeight(scroll.clientHeight, naturalHeight, targetTop);

      if (spacer && Math.abs(currentSpacerHeight - spacerHeight) > 1) {
        spacer.style.height = `${spacerHeight}px`;
      }

      setScrollTop(scroll, targetTop);
    },
    [setScrollTop],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: every streamed update needs a layout pass
  useLayoutEffect(() => {
    const scroll = scrollRef.current;
    if (!scroll) return;

    if (conversationRef.current !== conversationId) {
      conversationRef.current = conversationId;
      targetRef.current = null;
      modeRef.current = "follow";
      freezeBaselineRef.current = null;
      clearSpacer();
      notifyNearBottom(true);
    }

    const target = findTargetMessage(scroll, scrollTarget, conversationId);
    if (scrollTarget?.conversationId === conversationId) {
      const isNewTarget = targetRef.current !== scrollTarget;
      targetRef.current = scrollTarget;
      if (isNewTarget) {
        modeRef.current = "anchor";
        notifyNearBottom(false);
      }

      if (!target) {
        // The message normally exists in the same commit. Keep the previous
        // position until it does instead of falling through to bottom-follow.
        clearSpacer();
        return;
      }

      // A user who has taken over the viewport keeps control while the
      // assistant streams. New scroll targets are handled above.
      if (modeRef.current !== "anchor") {
        if (modeRef.current === "free") maintainFreezeRange(scroll);
        return;
      }

      applyAnchor(scroll, target);
      return;
    }

    targetRef.current = null;
    if (modeRef.current === "follow") {
      freezeBaselineRef.current = null;
      clearSpacer();
      scroll.scrollTop = scroll.scrollHeight;
      notifyNearBottom(true);
      return;
    }

    if (modeRef.current === "free") maintainFreezeRange(scroll);
  }, [
    messages,
    run,
    conversationId,
    scrollTarget,
    applyAnchor,
    clearSpacer,
    maintainFreezeRange,
    notifyNearBottom,
  ]);

  // Layout can change after the initial commit: long user messages collapse,
  // images load, or generated UI resolves. Keep the anchored question stable.
  useLayoutEffect(() => {
    const scroll = scrollRef.current;
    const thread = scroll?.querySelector<HTMLElement>(".thread");
    if (!scroll || !thread || typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(() => {
      const target = targetRef.current;
      if (modeRef.current === "anchor") {
        if (!target) return;
        const message = findTargetMessage(scroll, target, conversationId);
        if (message) applyAnchor(scroll, message);
        return;
      }
      if (modeRef.current === "free") maintainFreezeRange(scroll);
    });
    observer.observe(thread);
    return () => observer.disconnect();
  }, [conversationId, applyAnchor, maintainFreezeRange]);

  const onScroll = useCallback(() => {
    const scroll = scrollRef.current;
    if (!scroll) return;

    if (modeRef.current === "anchor" && Math.abs(scroll.scrollTop - anchorTopRef.current) <= 2) {
      return;
    }

    const spacerHeight = spacerPixelHeight(spacerRef.current);
    const distanceToBottom =
      scroll.scrollHeight - spacerHeight - scroll.scrollTop - scroll.clientHeight;
    const near = spacerHeight <= 1 && distanceToBottom <= bottomThreshold;

    if (near) {
      modeRef.current = "follow";
      freezeBaselineRef.current = null;
      clearSpacer();
      notifyNearBottom(true);
      return;
    }

    if (modeRef.current !== "free") {
      modeRef.current = "free";
      freezeBaselineRef.current = scroll.scrollHeight;
    }
    notifyNearBottom(false);
  }, [clearSpacer, notifyNearBottom]);

  const onUserIntent = useCallback(() => {
    if (modeRef.current === "free") return;
    const scroll = scrollRef.current;
    if (!scroll) return;
    modeRef.current = "free";
    freezeBaselineRef.current = scroll.scrollHeight;
  }, []);

  const jumpToBottom = useCallback(() => {
    const scroll = scrollRef.current;
    if (!scroll) return;
    modeRef.current = "follow";
    freezeBaselineRef.current = null;
    clearSpacer();
    notifyNearBottom(true);
    scroll.scrollTop = scroll.scrollHeight;
  }, [clearSpacer, notifyNearBottom]);

  return {
    scrollRef,
    spacerRef,
    nearBottom,
    onScroll,
    onWheel: onUserIntent,
    onTouchStart: onUserIntent,
    onKeyDown: onUserIntent,
    jumpToBottom,
  };
}

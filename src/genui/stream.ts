import {
  applySpecPatch,
  parseSpecStreamLine,
  type Spec,
  type SpecStreamLine,
} from "@json-render/core";
import type { AssistantSegment, UISpecView } from "../lib/types";
import { type Messages, messagesIn } from "../locales";

const specFence = "```spec";
const specCacheLimit = 32;
const specCache = new Map<string, UISpecView>();

export type ParsedAssistantAnswer = {
  text: string;
  spec: UISpecView | null;
  segments: AssistantSegment[];
};

type ParseOptions = {
  streaming?: boolean;
  messages?: Messages;
};

export function parseAssistantAnswer(
  raw: string,
  { streaming = false, messages = messagesIn("zh-Hans") }: ParseOptions = {},
): ParsedAssistantAnswer {
  const lines = raw.split("\n");
  const chunks: RawChunk[] = [];
  let inSpec = false;
  let fences = 0;
  let error = "";
  const incompleteLastLine = streaming && raw.length > 0 && !raw.endsWith("\n");

  const textChunk = () => {
    const last = chunks.at(-1);
    if (last?.kind === "text") return last;
    const chunk: RawChunk = { kind: "text", lines: [] };
    chunks.push(chunk);
    return chunk;
  };

  textChunk();
  for (const [index, line] of lines.entries()) {
    const trimmed = line.trim();
    if (!inSpec && trimmed === specFence) {
      inSpec = true;
      fences += 1;
      chunks.push({ kind: "spec", lines: [], closed: false });
      if (fences > 1) {
        error ||= messages.genuiErrorSingleSpec();
      }
      continue;
    }
    if (inSpec) {
      if (trimmed === "```") {
        inSpec = false;
        const current = chunks.at(-1);
        if (current?.kind === "spec") current.closed = true;
        continue;
      }
      if (incompleteLastLine && index === lines.length - 1) continue;
      const current = chunks.at(-1);
      if (current?.kind === "spec") current.lines.push(line);
      continue;
    }
    textChunk().lines.push(line);
  }

  if (fences === 0) {
    return {
      text: raw,
      spec: null,
      segments: [{ kind: "text", key: "text-0", text: raw }],
    };
  }
  if (inSpec && !streaming) {
    error ||= messages.genuiErrorIncomplete();
  }

  const segments: AssistantSegment[] = [];
  let firstSpec: UISpecView | null = null;
  let specIndex = 0;
  let segmentIndex = 0;
  for (const chunk of chunks) {
    if (chunk.kind === "text") {
      const text = normalizeText(chunk.lines);
      if (text) {
        segments.push({ kind: "text", key: `text-${segmentIndex}`, text });
        segmentIndex += 1;
      }
      continue;
    }
    const spec = parseSpecChunk(chunk, streaming, error, messages);
    if (specIndex === 0) {
      firstSpec = spec;
      segments.push({ kind: "spec", key: `spec-${segmentIndex}`, spec });
      segmentIndex += 1;
    }
    specIndex += 1;
  }

  return {
    text: segments
      .filter((segment) => segment.kind === "text")
      .map((segment) => (segment.kind === "text" ? segment.text : ""))
      .join("\n\n"),
    spec: firstSpec,
    segments,
  };
}

type RawChunk =
  | { kind: "text"; lines: string[] }
  | { kind: "spec"; lines: string[]; closed: boolean };

function normalizeText(lines: string[]) {
  return lines
    .join("\n")
    .replace(/^(?:[ \t]*\n)+/, "")
    .replace(/(?:\n[ \t]*)+$/, "");
}

function parseSpecChunk(
  chunk: Extract<RawChunk, { kind: "spec" }>,
  streaming: boolean,
  priorError: string,
  messages: Messages,
) {
  const cacheKey =
    chunk.closed && !priorError
      ? `${streaming ? "live" : "final"}\u0000${chunk.lines.join("\n")}`
      : "";
  if (cacheKey) {
    const cached = specCache.get(cacheKey);
    if (cached) {
      specCache.delete(cacheKey);
      specCache.set(cacheKey, cached);
      return cached;
    }
  }

  let error = priorError;
  if (!chunk.closed && !streaming) {
    error ||= messages.genuiErrorIncomplete();
  }

  const patches: SpecStreamLine[] = [];
  for (const line of chunk.lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const patch = parseSpecStreamLine(trimmed);
    if (!patch) {
      error ||= messages.genuiErrorInvalidPatch();
      continue;
    }
    patches.push(patch);
  }

  let spec: Spec = { root: "", elements: {} };
  try {
    for (const patch of patches) {
      spec = applySpecPatch(spec, patch);
    }
  } catch {
    error ||= messages.genuiErrorApply();
  }

  if (!streaming && (!spec.root || Object.keys(spec.elements).length === 0)) {
    error ||= messages.genuiErrorEmpty();
  }
  if (!streaming) {
    error ||= checkLimits(spec, messages);
  }

  const status = !chunk.closed && streaming ? "streaming" : error ? "invalid" : "complete";
  const result = {
    spec,
    status,
    ...(error ? { error } : {}),
  } satisfies UISpecView;
  if (cacheKey && result.status === "complete") {
    specCache.set(cacheKey, result);
    while (specCache.size > specCacheLimit) {
      const oldest = specCache.keys().next().value;
      if (!oldest) break;
      specCache.delete(oldest);
    }
  }
  return result;
}

function checkLimits(spec: Spec, messages: Messages) {
  const elementCount = Object.keys(spec.elements).length;
  if (elementCount > 80) return messages.genuiErrorTooManyElements();
  const seen = new Set<string>();
  const stack: Array<{ key: string; depth: number }> = [{ key: spec.root, depth: 1 }];
  while (stack.length > 0) {
    const current = stack.pop();
    if (!current || seen.has(current.key)) continue;
    seen.add(current.key);
    if (current.depth > 8) return messages.genuiErrorTooDeep();
    const element = spec.elements[current.key];
    if (!element) continue;
    for (const child of element.children ?? []) {
      stack.push({ key: child, depth: current.depth + 1 });
    }
    for (const children of Object.values(element.slots ?? {})) {
      for (const child of children) {
        stack.push({ key: child, depth: current.depth + 1 });
      }
    }
  }
  return "";
}

import { Search, X } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import type { Conversation, Project, SearchResult } from "../mygo";
import { IconButton, Modal } from "../ui";

type SearchDialogProps = {
  conversations: Conversation[];
  projects: Project[];
  onSearch: (query: string) => Promise<SearchResult[]>;
  onSelect: (id: string, messageIndex: number) => void;
  onClose: () => void;
};

export function SearchDialog({
  conversations,
  projects,
  onSearch,
  onSelect,
  onClose,
}: SearchDialogProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [remoteResults, setRemoteResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [searchedQuery, setSearchedQuery] = useState("");
  const projectNames = useMemo(
    () => new Map(projects.map((project) => [project.id, project.name])),
    [projects],
  );
  const recentResults = useMemo<SearchResult[]>(
    () =>
      [...conversations]
        .sort(
          (left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
        )
        .map((conversation) => ({
          conversation,
          messageIndex: -1,
          snippet: conversation.preview,
        })),
    [conversations],
  );
  const searching = query.trim() !== "";
  const results = searching ? remoteResults : recentResults;

  useEffect(() => {
    const frame = requestAnimationFrame(() => inputRef.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, []);

  useEffect(() => {
    const normalized = query.trim();
    if (!normalized) {
      setRemoteResults([]);
      setSearchedQuery("");
      setLoading(false);
      setError("");
      return;
    }

    let cancelled = false;
    setRemoteResults([]);
    setSearchedQuery("");
    setLoading(true);
    setError("");
    const timer = window.setTimeout(() => {
      void onSearch(normalized)
        .then((items) => {
          if (cancelled) return;
          setRemoteResults(items);
          setSearchedQuery(normalized);
        })
        .catch((searchError: unknown) => {
          if (cancelled) return;
          setError(searchError instanceof Error ? searchError.message : "搜索失败");
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }, 140);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [onSearch, query]);

  function select(result: SearchResult) {
    onSelect(result.conversation.id, result.messageIndex);
    onClose();
  }

  return (
    <Modal className="search-modal" label="搜索对话" onClose={onClose}>
      <div className="search-dialog-head">
        <Search size={18} aria-hidden="true" />
        <input
          ref={inputRef}
          type="search"
          value={query}
          aria-label="搜索对话"
          placeholder="搜索对话内容"
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return;
            if (event.key === "Enter" && !loading && searchedQuery === query.trim() && results[0]) {
              event.preventDefault();
              select(results[0]);
            }
          }}
        />
        {query ? (
          <IconButton label="清除搜索" className="search-dialog-clear" onClick={() => setQuery("")}>
            <X size={16} />
          </IconButton>
        ) : (
          <span className="search-dialog-hint" aria-hidden="true">
            Esc
          </span>
        )}
      </div>

      <div className="search-dialog-body">
        <h2 className="search-dialog-label">{searching ? "搜索结果" : "最近对话"}</h2>
        {loading ? (
          <p className="search-dialog-empty">正在搜索…</p>
        ) : error ? (
          <p className="search-dialog-empty">{error}</p>
        ) : results.length > 0 ? (
          <div className="search-results">
            {results.map((result) => {
              const conversation = result.conversation;
              const projectName = conversation.projectId
                ? projectNames.get(conversation.projectId)
                : undefined;
              const snippet = result.snippet || conversation.preview;
              return (
                <button
                  key={conversation.id}
                  type="button"
                  className="search-result"
                  aria-label={`打开“${conversation.title}”`}
                  onClick={() => select(result)}
                >
                  <span className="search-result-copy">
                    <span className="search-result-title">{conversation.title}</span>
                    {snippet && (
                      <span className="search-result-snippet">
                        {roleLabel(result.role) && (
                          <span className="search-result-role">{roleLabel(result.role)}</span>
                        )}
                        <span className="search-result-snippet-text">
                          {highlight(snippet, searching ? query : "")}
                        </span>
                      </span>
                    )}
                  </span>
                  <span className="search-result-meta">
                    {(result.matchCount ?? 0) > 1 && (
                      <span className="search-result-matches">{result.matchCount} 条</span>
                    )}
                    {projectName && <span className="search-result-project">{projectName}</span>}
                  </span>
                </button>
              );
            })}
          </div>
        ) : (
          <p className="search-dialog-empty">{searching ? "没有匹配的对话" : "还没有对话"}</p>
        )}
      </div>
    </Modal>
  );
}

function roleLabel(role: string | undefined) {
  if (role === "user") return "你";
  if (role === "assistant") return "Mira";
  return "";
}

function highlight(text: string, query: string): ReactNode {
  const needle = query.trim();
  if (!needle) return text;
  const lowerText = text.toLocaleLowerCase();
  const lowerNeedle = needle.toLocaleLowerCase();
  const parts: ReactNode[] = [];
  let cursor = 0;
  let match = lowerText.indexOf(lowerNeedle);
  while (match >= 0) {
    if (match > cursor) parts.push(text.slice(cursor, match));
    parts.push(<mark key={`${match}-${cursor}`}>{text.slice(match, match + needle.length)}</mark>);
    cursor = match + needle.length;
    match = lowerText.indexOf(lowerNeedle, cursor);
  }
  if (cursor < text.length) parts.push(text.slice(cursor));
  return parts;
}

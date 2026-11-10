import { Check, Copy } from "lucide-react";
import { type HTMLAttributes, type ReactNode, useState } from "react";
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";
import remarkGfm from "remark-gfm";
import { copyText } from "./format";

type HastNode = {
  type?: string;
  value?: string;
  children?: HastNode[];
};

function nodeText(node: HastNode | undefined): string {
  if (!node) return "";
  if (node.type === "text") return node.value ?? "";
  return (node.children ?? []).map(nodeText).join("");
}

export function Markdown({ children }: { children: string }) {
  return (
    <div className="markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          a({ children, ...props }) {
            return (
              <a {...props} target="_blank" rel="noreferrer">
                {children}
              </a>
            );
          },
          pre({ node, children }) {
            return <CodeBlock node={node as HastNode}>{children}</CodeBlock>;
          },
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}

function CodeBlock({ node, children }: { node?: HastNode; children: ReactNode }) {
  const [copied, setCopied] = useState(false);
  const language = findLanguage(node);
  const text = nodeText(node);

  return (
    <div className="code-block">
      <div className="code-head">
        <span>{language || "code"}</span>
        <button
          type="button"
          className="code-copy"
          onClick={async () => {
            await copyText(text);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1400);
          }}
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          {copied ? "已复制" : "复制"}
        </button>
      </div>
      <pre {...({} as HTMLAttributes<HTMLPreElement>)}>{children}</pre>
    </div>
  );
}

function findLanguage(node: HastNode | undefined): string {
  const code = node?.children?.find((child) => child.type === "element");
  const className = (code as unknown as { properties?: { className?: string[] } })?.properties
    ?.className;
  const match = className?.find((item) => item.startsWith("language-"));
  return match ? match.slice("language-".length) : "";
}

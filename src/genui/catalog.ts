import { defineCatalog } from "@json-render/core";
import { schema } from "@json-render/react/schema";
import { z } from "zod";

const tableCell = z.union([z.string(), z.number(), z.boolean(), z.null()]);
const tableRow = z.record(z.string(), tableCell);
const chartDatum = z.record(z.string(), z.union([z.string(), z.number(), z.null()]));

const gap = z.enum(["sm", "md", "lg"]).nullable();
const tone = z.enum(["neutral", "success", "warning", "danger"]).nullable();

export const genUICatalog = defineCatalog(schema, {
  components: {
    Stack: {
      props: z.object({
        direction: z.enum(["row", "column"]).nullable(),
        gap,
        align: z.enum(["start", "center", "end", "stretch"]).nullable(),
      }),
      slots: ["default"],
      description: "Lays out child components in a row or column.",
    },
    Card: {
      props: z.object({
        title: z.string().nullable(),
        description: z.string().nullable(),
      }),
      slots: ["default"],
      description: "A bounded content surface for one related group. Never nest a Card in a Card.",
    },
    Text: {
      props: z.object({
        content: z.string(),
        variant: z.enum(["body", "muted", "caption", "heading"]).nullable(),
      }),
      description: "Short plain text. Do not repeat the prose answer.",
    },
    Metric: {
      props: z.object({
        label: z.string(),
        value: z.string(),
        detail: z.string().nullable(),
        trend: z.enum(["up", "down", "neutral"]).nullable(),
      }),
      description: "A key number with a label, optional detail and trend.",
    },
    Table: {
      props: z.object({
        caption: z.string().nullable(),
        columns: z
          .array(
            z.object({
              key: z.string(),
              label: z.string(),
            }),
          )
          .min(1)
          .max(8),
        rows: z.array(tableRow).max(50),
        emptyText: z.string().nullable(),
      }),
      description:
        "A compact data table. Use state binding for generated data; at most 50 rows and 8 columns.",
    },
    Chart: {
      props: z.object({
        variant: z.enum(["bar", "line", "donut"]),
        title: z.string().nullable(),
        data: z.array(chartDatum).min(1).max(60),
        xKey: z.string(),
        yKey: z.string(),
        unit: z.string().nullable(),
        size: z.enum(["md", "lg"]).nullable(),
      }),
      description:
        "A bar, line or donut chart. xKey is the category/label field and yKey is the numeric value field. At most 60 data points.",
    },
    Input: {
      props: z.object({
        label: z.string(),
        value: z.union([z.string(), z.number()]).nullable(),
        placeholder: z.string().nullable(),
        type: z.enum(["text", "number", "email"]).nullable(),
        hint: z.string().nullable(),
      }),
      description:
        'Single-line input. Bind value with { "$bindState": "/path" } so the value is available to actions.',
    },
    Select: {
      props: z.object({
        label: z.string(),
        value: z.string().nullable(),
        placeholder: z.string().nullable(),
        options: z
          .array(
            z.object({
              value: z.string(),
              label: z.string(),
            }),
          )
          .min(1)
          .max(20),
      }),
      description:
        'Dropdown selection. Bind value with { "$bindState": "/path" } and provide up to 20 options.',
    },
    Slider: {
      props: z.object({
        label: z.string(),
        value: z.number().nullable(),
        min: z.number(),
        max: z.number(),
        step: z.number().nullable(),
        unit: z.string().nullable(),
      }),
      description:
        'Numeric range input. Bind value with { "$bindState": "/path" }; use for calculators and bounded choices.',
    },
    Button: {
      props: z.object({
        label: z.string(),
        variant: z.enum(["primary", "secondary", "ghost"]).nullable(),
        disabled: z.boolean().nullable(),
      }),
      description:
        "A button. Bind on.press to setState/pushState/removeState for local behavior, or ask_agent to continue with the current UI state.",
    },
    Callout: {
      props: z.object({
        title: z.string().nullable(),
        content: z.string(),
        tone,
      }),
      description: "A short highlighted note, warning, result or takeaway.",
    },
    Link: {
      props: z.object({
        text: z.string(),
        href: z.string(),
      }),
      description: "An external http(s) link. Never fabricate a URL.",
    },
  },
  actions: {
    ask_agent: {
      params: z.object({
        message: z.string().min(1).max(2000),
      }),
      description:
        "Sends a new user message to Mira. Use $template to include the current state values the Agent needs.",
    },
  },
});

const generatedPrompt = genUICatalog.prompt({
  mode: "inline",
  customRules: [
    "Generate a UI only when it materially improves comprehension or interaction, such as comparisons, dashboards, tables, charts, calculators, forms, plans or step-by-step tools.",
    "Do not generate a UI for simple conversation, code, essays, creative writing, or when plain Markdown is clearer.",
    "Use only facts, numbers and links that are present in the conversation or tool results. Never invent data or sample content.",
    "If real data is missing, show an empty state or omit the interface rather than inventing content.",
    "Place exactly one ```spec fence at the point where the interface is most useful. Do not default to placing it at the end. Short prose may continue after the closing fence.",
    "Emit /state patches immediately after the elements that reference them. Use one root element and keep the tree under 80 elements.",
    "Keep copy short. When the interface replaces a table, chart, form, calculator or comparison, omit the equivalent Markdown and never repeat the same data before or after the interface. Never nest Card inside Card.",
    "For interactions that need the model, use ask_agent with a $template message containing every relevant current state value.",
    "Do not use viewport-height classes, raw HTML, images, custom colors or arbitrary CSS.",
  ],
});

export const genUIPrompt = generatedPrompt
  .replace(
    "You respond conversationally. When generating UI, first write a brief explanation (1-3 sentences), then output JSONL patch lines wrapped in a ```spec code fence.",
    "You respond conversationally. When generating UI, place the JSONL patch lines wrapped in a ```spec code fence at the point in the answer where the interface is most useful.",
  )
  .replace(
    "2. Write a brief conversational response before any JSONL output",
    "2. A short conversational explanation may come before or after the JSONL output; place the interface where it best supports the answer.",
  )
  .replace(
    "Include realistic sample data in state. For blogs: 3-4 posts with titles, excerpts, authors, dates. For product lists: 3-5 items with names, prices, descriptions. Never leave arrays empty.",
    "Populate state only with data supplied by the user, conversation, or tool results. Use an empty collection when no data is available.",
  )
  .replace(
    "16. When the user asks for a UI that displays data (e.g. blog posts, products, users), ALWAYS include a state field with realistic sample data. The state field is a top-level field on the spec (sibling of root/elements).",
    "16. The state field is a top-level field on the spec (sibling of root/elements). Include only real data supplied by the user, conversation, or tool results.",
  )
  .replace(
    "20. Always include realistic, professional-looking sample data. For blogs include 3-4 posts with varied titles, authors, dates, categories. For products include names, prices, images. Never leave data empty.",
    "20. Omit data visualizations when no real data is available; never fill them with mock values.",
  );

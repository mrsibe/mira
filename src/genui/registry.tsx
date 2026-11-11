import { type Components, defineRegistry, useBoundProp } from "@json-render/react";
import { ArrowDownRight, ArrowUpRight, Minus } from "lucide-react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { genUICatalog } from "./catalog";

const chartColors = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
];

type AskAgent = (message: string) => void;

const genUIComponents = {
  Stack: ({ props, children }) => (
    <div
      className="genui-stack"
      data-direction={props.direction ?? "column"}
      data-gap={props.gap ?? "md"}
      data-align={props.align ?? "stretch"}
    >
      {children}
    </div>
  ),
  Card: ({ props, children }) => (
    <section className="genui-card">
      {(props.title || props.description) && (
        <header className="genui-card-head">
          {props.title && <h3>{props.title}</h3>}
          {props.description && <p>{props.description}</p>}
        </header>
      )}
      <div className="genui-card-body">{children}</div>
    </section>
  ),
  Text: ({ props }) => (
    <p className="genui-text" data-variant={props.variant ?? "body"}>
      {props.content}
    </p>
  ),
  Metric: ({ props }) => {
    const Trend =
      props.trend === "up" ? ArrowUpRight : props.trend === "down" ? ArrowDownRight : Minus;
    return (
      <div className="genui-metric">
        <span>{props.label}</span>
        <div className="genui-metric-value">
          <strong>{props.value}</strong>
          {props.trend && <Trend size={16} data-trend={props.trend} />}
        </div>
        {props.detail && <small>{props.detail}</small>}
      </div>
    );
  },
  Table: ({ props }) => {
    if (props.rows.length === 0) {
      return <p className="genui-empty">{props.emptyText ?? "暂无数据"}</p>;
    }
    return (
      <figure className="genui-table-figure">
        <div className="genui-table-wrap">
          <table className="genui-table">
            <thead>
              <tr>
                {props.columns.map((column) => (
                  <th key={column.key} scope="col">
                    {column.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {props.rows.map((row, index) => (
                <tr key={String(row.id ?? index)}>
                  {props.columns.map((column) => (
                    <td key={column.key}>{formatCell(row[column.key])}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {props.caption && <figcaption>{props.caption}</figcaption>}
      </figure>
    );
  },
  Chart: ({ props }) => {
    const data = props.data.map((row) => ({
      label: String(row[props.xKey] ?? ""),
      value: toNumber(row[props.yKey]),
    }));
    const height = props.size === "lg" ? 300 : 220;
    const summary = data.map((item) => `${item.label} ${item.value}${props.unit ?? ""}`).join("，");
    return (
      <figure className="genui-chart">
        {props.title && <figcaption>{props.title}</figcaption>}
        <div className="genui-chart-canvas" style={{ height }} role="img" aria-label={summary}>
          <ResponsiveContainer width="100%" height="100%">
            {props.variant === "bar" ? (
              <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
                <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="label"
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: "var(--text-tertiary)", fontSize: 11 }}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: "var(--text-tertiary)", fontSize: 11 }}
                />
                <Tooltip {...tooltipProps(props.unit)} />
                <Bar dataKey="value" fill="var(--chart-1)" radius={[5, 5, 0, 0]} />
              </BarChart>
            ) : props.variant === "line" ? (
              <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
                <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" vertical={false} />
                <XAxis
                  dataKey="label"
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: "var(--text-tertiary)", fontSize: 11 }}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: "var(--text-tertiary)", fontSize: 11 }}
                />
                <Tooltip {...tooltipProps(props.unit)} />
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke="var(--chart-1)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, fill: "var(--chart-1)" }}
                />
              </LineChart>
            ) : (
              <PieChart>
                <Tooltip {...tooltipProps(props.unit)} />
                <Pie
                  data={data}
                  dataKey="value"
                  nameKey="label"
                  innerRadius="56%"
                  outerRadius="82%"
                  paddingAngle={2}
                  stroke="var(--bg-elevated)"
                  strokeWidth={2}
                >
                  {data.map((item, index) => (
                    <Cell key={item.label} fill={chartColors[index % chartColors.length]} />
                  ))}
                </Pie>
              </PieChart>
            )}
          </ResponsiveContainer>
        </div>
        {props.variant === "donut" && (
          <ul className="genui-chart-legend">
            {data.map((item, index) => (
              <li key={item.label}>
                <span style={{ background: chartColors[index % chartColors.length] }} />
                {item.label}
                <strong>
                  {item.value}
                  {props.unit}
                </strong>
              </li>
            ))}
          </ul>
        )}
      </figure>
    );
  },
  Input: ({ props, bindings, loading }) => {
    const [value, setValue] = useBoundProp<string | number | null>(
      props.value ?? null,
      bindings?.value,
    );
    return (
      <label className="genui-field">
        <span>{props.label}</span>
        <input
          type={props.type ?? "text"}
          value={String(value ?? "")}
          placeholder={props.placeholder ?? ""}
          disabled={loading}
          onChange={(event) => {
            if (props.type === "number") {
              setValue(event.target.value === "" ? null : Number(event.target.value));
            } else {
              setValue(event.target.value);
            }
          }}
        />
        {props.hint && <small>{props.hint}</small>}
      </label>
    );
  },
  Select: ({ props, bindings, loading }) => {
    const [value, setValue] = useBoundProp<string | null>(props.value ?? null, bindings?.value);
    return (
      <label className="genui-field">
        <span>{props.label}</span>
        <select
          value={value ?? ""}
          disabled={loading}
          onChange={(event) => setValue(event.target.value)}
        >
          <option value="" disabled>
            {props.placeholder ?? "请选择"}
          </option>
          {props.options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
    );
  },
  Slider: ({ props, bindings, loading }) => {
    const [value, setValue] = useBoundProp<number | null>(
      props.value ?? props.min,
      bindings?.value,
    );
    return (
      <label className="genui-field genui-slider">
        <span>
          {props.label}
          <strong>
            {value ?? props.min}
            {props.unit}
          </strong>
        </span>
        <input
          type="range"
          min={props.min}
          max={props.max}
          step={props.step ?? 1}
          value={value ?? props.min}
          disabled={loading}
          onChange={(event) => setValue(Number(event.target.value))}
        />
      </label>
    );
  },
  Button: ({ props, emit, loading }) => (
    <button
      type="button"
      className="genui-button"
      data-variant={props.variant ?? "secondary"}
      disabled={loading || Boolean(props.disabled)}
      onClick={() => emit("press")}
    >
      {props.label}
    </button>
  ),
  Callout: ({ props }) => (
    <aside className="genui-callout" data-tone={props.tone ?? "neutral"}>
      {props.title && <strong>{props.title}</strong>}
      <p>{props.content}</p>
    </aside>
  ),
  Link: ({ props }) => {
    const href = safeExternalHref(props.href);
    if (!href) return <span className="genui-link-invalid">{props.text}</span>;
    return (
      <a className="genui-link" href={href} target="_blank" rel="noreferrer">
        {props.text}
      </a>
    );
  },
} satisfies Components<typeof genUICatalog>;

export function createGenUIRegistry(onAskAgent: AskAgent) {
  return defineRegistry(genUICatalog, {
    components: genUIComponents,
    actions: {
      ask_agent: async (params) => {
        const message = typeof params?.message === "string" ? params.message.trim() : "";
        if (message) onAskAgent(message);
      },
    },
  });
}

function toNumber(value: unknown) {
  const number = typeof value === "number" ? value : Number(value);
  return Number.isFinite(number) ? number : 0;
}

function formatCell(value: unknown) {
  if (value === null || value === undefined) return "—";
  if (typeof value === "boolean") return value ? "是" : "否";
  if (typeof value === "string" || typeof value === "number") return String(value);
  return JSON.stringify(value);
}

function safeExternalHref(raw: string) {
  try {
    const url = new URL(raw);
    return url.protocol === "http:" || url.protocol === "https:" ? url.toString() : null;
  } catch {
    return null;
  }
}

function tooltipProps(unit: string | null) {
  return {
    cursor: false as const,
    contentStyle: {
      background: "var(--bg-elevated)",
      border: "1px solid var(--border)",
      borderRadius: "10px",
      boxShadow: "var(--shadow-2)",
      color: "var(--text)",
      fontSize: 12,
    },
    labelStyle: { color: "var(--text-secondary)" },
    formatter: (value: unknown) => [`${value}${unit ?? ""}`, "数值"] as [string, string],
  };
}

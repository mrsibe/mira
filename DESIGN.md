---
version: alpha
name: Mira-ChatGPT-Inspired-design-analysis
description: A single-user desktop chat client that follows ChatGPT's calm, monochrome conversation layout. The shell, thread, composer, and response chrome are deliberately quiet so the conversation itself is the only focal point.
colors:
  light:
    canvas: "#ffffff"
    canvas-sidebar: "#f9f9f9"
    canvas-subtle: "#f4f4f4"
    canvas-code: "#f6f6f6"
    code-header: "#ededed"
    hairline: "#e5e5e5"
    hairline-strong: "#d4d4d4"
    ink: "#0d0d0d"
    body: "#5d5d5d"
    mute: "#8f8f8f"
    on-ink: "#ffffff"
    link: "#0b57d0"
    focus: "#2563eb"
    danger: "#b42318"
    danger-soft: "#fef3f2"
    warning-soft: "#fff8e6"
    warning-border: "#efd9a7"
  dark:
    canvas: "#212121"
    canvas-sidebar: "#171717"
    canvas-elevated: "#2f2f2f"
    canvas-subtle: "#303030"
    canvas-code: "#0d0d0d"
    code-header: "#1b1b1b"
    hairline: "#3a3a3a"
    hairline-strong: "#4f4f4f"
    ink: "#ececec"
    body: "#b4b4b4"
    mute: "#8f8f8f"
    on-ink: "#0d0d0d"
    link: "#7ab8ff"
    focus: "#6ea8fe"
    danger: "#ff9b91"
    danger-soft: "rgba(255, 99, 71, 0.12)"
    warning-soft: "rgba(245, 166, 35, 0.12)"
    warning-border: "rgba(245, 166, 35, 0.32)"
typography:
  sans:
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif'
    weights: [400, 500, 600]
  mono:
    fontFamily: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace
    weights: [400]
  display-lg:
    fontSize: 32px
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: -0.03em
  display-md:
    fontSize: 24px
    fontWeight: 500
    lineHeight: 1.3
    letterSpacing: -0.02em
  title:
    fontSize: 18px
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: -0.01em
  body-lg:
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.75
    letterSpacing: 0
  body:
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: 0
  caption:
    fontSize: 12px
    fontWeight: 400
    lineHeight: 1.35
    letterSpacing: 0
  code:
    fontFamily: mono
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.65
  code-small:
    fontFamily: mono
    fontSize: 12px
    fontWeight: 400
    lineHeight: 1.6
rounded:
  sm: 8px
  md: 12px
  lg: 16px
  xl: 24px
  composer: 28px
  full: 9999px
spacing:
  1: 4px
  2: 8px
  3: 12px
  4: 16px
  5: 24px
  6: 32px
  7: 48px
  8: 64px
shadows:
  level-1: "0 1px 2px rgba(0, 0, 0, 0.04)"
  level-2: "0 1px 2px rgba(0, 0, 0, 0.04), 0 2px 6px rgba(0, 0, 0, 0.06)"
  level-3: "0 2px 4px rgba(0, 0, 0, 0.05), 0 8px 18px rgba(0, 0, 0, 0.08)"
  level-4: "0 4px 10px rgba(0, 0, 0, 0.06), 0 16px 32px rgba(0, 0, 0, 0.12)"
  level-5: "0 2px 4px rgba(0, 0, 0, 0.05), 0 12px 24px rgba(0, 0, 0, 0.10), 0 32px 64px rgba(0, 0, 0, 0.18)"
layout:
  sidebar-width: 260px
  sidebar-width-mobile: 280px
  topbar-height: 52px
  thread-max-width: 768px
  composer-max-width: 768px
  breakpoints:
    mobile: 640px
    sidebar-overlay: 900px
    hide-thread-title: 1100px
components:
  sidebar:
    backgroundColor: "{colors.light.canvas-sidebar}"
    textColor: "{colors.light.ink}"
    width: "{layout.sidebar-width}"
    borderRight: "1px solid {colors.light.hairline}"
  topbar:
    backgroundColor: "{colors.light.canvas}"
    height: "{layout.topbar-height}"
    padding: "0 12px"
  model-selector:
    textColor: "{colors.light.ink}"
    typography: "16px / 500"
    rounded: "{rounded.sm}"
    height: 32px
    padding: "6px 8px"
  new-chat-row:
    textColor: "{colors.light.ink}"
    typography: "14px / 500"
    rounded: "{rounded.sm}"
    height: 40px
  conversation-row:
    textColor: "{colors.light.ink}"
    typography: "14px / 400"
    rounded: "{rounded.sm}"
    height: 36px
    activeBackground: "{colors.light.canvas-subtle}"
  user-message:
    backgroundColor: "{colors.light.canvas-subtle}"
    textColor: "{colors.light.ink}"
    typography: "16px / 400 / 1.6"
    rounded: "{rounded.xl}"
    padding: "10px 16px"
    maxWidth: "78%"
  assistant-message:
    backgroundColor: transparent
    textColor: "{colors.light.ink}"
    typography: "16px / 400 / 1.75"
  composer:
    backgroundColor: "{colors.light.canvas}"
    borderColor: "{colors.light.hairline}"
    textColor: "{colors.light.ink}"
    typography: "16px / 400 / 1.5"
    rounded: "{rounded.composer}"
    minHeight: 52px
    padding: "8px 8px 8px 10px"
    shadow: "{shadows.level-2}"
  send-button:
    backgroundColor: "{colors.light.ink}"
    textColor: "{colors.light.on-ink}"
    rounded: "{rounded.full}"
    size: 32px
  code-block:
    backgroundColor: "{colors.light.canvas-code}"
    headerBackground: "{colors.light.code-header}"
    borderColor: "{colors.light.hairline}"
    textColor: "{colors.light.ink}"
    typography: "{typography.code}"
    rounded: "{rounded.lg}"
  dialog:
    backgroundColor: "{colors.light.canvas}"
    textColor: "{colors.light.ink}"
    borderColor: "{colors.light.hairline}"
    rounded: "{rounded.lg}"
    shadow: "{shadows.level-5}"
    width: 560px
  toast:
    backgroundColor: "{colors.light.ink}"
    textColor: "{colors.light.on-ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.sm}"
    shadow: "{shadows.level-4}"
---

## Overview

Mira is a local-first, single-user desktop chat client. Its interface follows the
conversation-first language popularized by ChatGPT: a quiet sidebar, a narrow
reading column, and inline assistant responses. A new chat shows only a short
personal greeting and the composer; once the conversation starts, the composer
docks to the bottom. Nothing in the chrome competes with the message.

The palette is strictly neutral. Color appears only for links, focus, errors, and
the temporary running/error state of a tool. There are no decorative gradients,
no colored feature cards, and no drop shadows on message content. Elevation is
reserved for surfaces that genuinely float above the page: the composer, menus,
dialogs, and toasts.

**Key characteristics:**

- A two-pane shell: a 260 px sidebar and a fluid main pane. Controls inside the
  sidebar stay on the rail surface and only reveal a hover fill.
- A 52 px top bar that holds the sidebar toggle and the current conversation
  title. It intentionally has no model, new-chat, or settings controls.
- A 768 px reading column shared by the thread and composer.
- User turns appear as right-aligned neutral bubbles; assistant turns are plain,
  unboxed text without an avatar.
- A pill composer with an attach button, a growing textarea, and a circular send
  button. It is centered on a new chat and docked below the thread after the
  first message.
- Monochrome light and dark themes driven by CSS custom properties.
- A five-section settings dialog (常规 / 个性化 / 记忆 / 搜索 / 模型) with a
  quiet rail that mirrors the main sidebar's hover language.
- Motion is limited to state changes: typing dots, tool pulse, and dialog
  entrance. Reduced-motion preferences collapse all of it.

## Colors

### Light theme

- **Canvas** (`#ffffff`) — the main conversation surface and dialog surface.
- **Sidebar** (`#f9f9f9`) — the sidebar rail and the default fill of the
  transparent elements inside it.
- **Subtle** (`#f4f4f4`) — user bubbles, hovered controls, focused search,
  inline code, and tool results.
- **Hairline** (`#e5e5e5`) — dividers and control borders.
- **Ink** (`#0d0d0d`) — primary text and the inverse send button.
- **Body** (`#5d5d5d`) — secondary copy, summaries, and model captions.
- **Mute** (`#8f8f8f`) — placeholders, timestamps, tertiary labels.
- **Link** (`#0b57d0`) — inline links only.
- **Focus** (`#2563eb`) — keyboard focus rings and focused inputs.
- **Danger** (`#b42318`) — destructive labels and error icons.

### Dark theme

- **Canvas** (`#212121`) — the conversation surface.
- **Sidebar** (`#171717`) — the sidebar rail and the default fill of the
  transparent elements inside it.
- **Elevated** (`#2f2f2f`) — dialogs and floating controls.
- **Subtle** (`#303030`) — user bubbles, composer, inline code, hovered rows.
- **Hairline** (`#3a3a3a`) — separators and control borders.
- **Ink** (`#ececec`) — primary text and the inverse button fill.
- **Body** (`#b4b4b4`) — secondary text.
- **Mute** (`#8f8f8f`) — placeholders and tertiary labels.
- **Link** (`#7ab8ff`) — inline links with enough contrast on dark canvas.

### Semantic color

Color is not used to decorate categories or mark sections. It is limited to:

- **Focus:** a 2 px outline or a 3 px translucent ring on focused controls.
- **Danger:** errors, destructive actions, and failed tool states.
- **Warning:** the configuration banner when a provider is not yet set up.
- **Success:** a small green dot on completed tool cards only.

### Contrast

Body text meets a 4.5:1 contrast target in both themes. Secondary text is kept at
or above 4.5:1 against its surface; tertiary text is used only for non-essential
metadata and placeholders. Color is never the only signal for status: tool cards
also carry a text label (“运行中”, “完成”, “失败”).

## Typography

### Families

The interface uses the native system sans stack so it feels at home on macOS,
Windows, and Linux and remains legible for Chinese and Latin text. Monospace is
reserved for code, tool arguments, model names in technical contexts, and file
paths.

No display font is loaded over the network. The app should look correct offline
and on first launch.

### Hierarchy

| Token        | Size | Weight | Line height | Use                                       |
| ------------ | ---: | -----: | ----------: | ----------------------------------------- |
| `display-lg` | 32px |    500 |         1.2 | Empty-state headline                      |
| `display-md` | 24px |    500 |         1.3 | Markdown H1, boot headings                |
| `title`      | 18px |    600 |         1.3 | Dialog titles, Markdown H2                |
| `body-lg`    | 16px |    400 |        1.75 | Assistant answers, user bubbles, composer |
| `body`       | 14px |    400 |         1.5 | Sidebar rows, settings copy, buttons      |
| `caption`    | 12px |    400 |        1.35 | Hints, tool state, timestamps             |
| `code`       | 13px |    400 |        1.65 | Fenced code                               |
| `code-small` | 12px |    400 |         1.6 | Tool arguments and results                |

### Principles

- **Two weights do most of the work.** Body copy is 400; emphasis is 500 or 600.
  Nothing in the interface uses 700+.
- **The reading column is narrow.** Assistant text is capped by the 768 px thread
  container, producing roughly 45–75 Latin characters per line at 16 px.
- **Line height follows role.** Conversation body copy uses 1.75; short UI labels
  use 1.35–1.5; code uses 1.65.
- **Sentence case everywhere.** No all-caps labels, including tool states and
  group headings.
- **Markdown carries structure.** Headings, lists, tables, links, and code all
  keep the same text color hierarchy as the rest of the app.

## Layout

### Shell

| Region         | Size       | Notes                                              |
| -------------- | ---------- | -------------------------------------------------- |
| Sidebar        | 260 px     | Fixed on desktop, overlay below 900 px             |
| Top bar        | 52 px      | Sidebar toggle and conversation title              |
| Thread         | max 768 px | Centered in the main pane                          |
| Composer       | max 768 px | Centered on new chat, docked at the bottom in a thread |
| Reading gutter | 24 px      | 16 px below 640 px                                 |

The sidebar owns navigation and history. The main pane owns exactly three
vertical regions: top bar, scrollable thread, and composer. Only the middle
region scrolls.

### Spacing system

The spacing scale is `4 / 8 / 12 / 16 / 24 / 32 / 48 / 64`. Adjacent steps differ
by at least 25% except for the 4–8 pair, which is intentionally tight for icon
and label alignment.

Grouping is expressed through space, not boxes:

- Sidebar: 8 px rail padding, 2 px between actions, 16 px between date groups.
- Thread: 24 px between turns, 12 px between the activity disclosure and the answer.
- Dialog: 20 px inner padding, 16 px between form fields.
- Welcome: 24 px between the greeting and the centered composer.

### Responsive strategy

| Breakpoint  | Behavior                                                                     |
| ----------- | ---------------------------------------------------------------------------- |
| ≥ 1100 px   | Full width; conversation title visible                                       |
| 900–1099 px | Thread title hidden; sidebar remains fixed                                   |
| 640–899 px  | Sidebar becomes a 280 px overlay with a dimmed backdrop                      |
| < 640 px    | Settings sections become tabs; gutters shrink; user bubble may use 86% width |

The new-chat greeting and composer sit at the upper golden-section line, around
38.2% of the viewport height, rather than at the exact center. After the first
user message the composer moves to the bottom of the thread. On narrow screens
the settings sidebar becomes a horizontal section tab row.

### Touch targets

Icon buttons are 32 px square with an additional 2 px focus outline. Primary
actions are 36–40 px tall. The conversation row is 36 px tall; on touch devices
the rename and delete actions are always visible rather than hover-only.

## Elevation & Depth

| Level | Treatment                   | Use                                 |
| ----- | --------------------------- | ----------------------------------- |
| 0     | Flat, no shadow             | Thread content, assistant responses |
| 1     | `0 1px 2px rgba(0,0,0,.04)` | Quiet hover lift                    |
| 2     | Two-layer 2–6 px shadow     | Composer and jump-to-bottom button  |
| 3     | 4–18 px soft stack          | Popovers and held surfaces          |
| 4     | 10–32 px float stack        | Toasts and the mobile sidebar       |
| 5     | 12–64 px modal stack        | Settings and confirmation dialogs   |

Depth is never used to make message content look like a card. User bubbles use a
surface change, not a shadow. The composer is the one persistent element that
floats above the page.

## Shapes

| Token      |  Value | Use                                                        |
| ---------- | -----: | ---------------------------------------------------------- |
| `sm`       |    8px | Icon buttons, conversation rows, inputs, inline containers |
| `md`       |   12px | Suggestions, tool cards, error cards                       |
| `lg`       |   16px | Code blocks, dialogs                                       |
| `xl`       |   24px | User message bubbles                                       |
| `composer` |   28px | Composer shell                                             |
| `full`     | 9999px | Avatars, circular send/stop buttons, jump button           |

The shape hierarchy is deliberate: small controls are softly rounded, content
surfaces are more rounded, and only the composer and user bubbles approach a
pill. The assistant response stays square and unboxed.

## Components

### Sidebar

A 260 px monochrome rail. The header contains only the “Mira” wordmark; the first
action is “新对话”; search follows; conversation history is grouped by recency
(“今天”, “昨天”, “前 7 天”, “更早”). Actions, the search field, and conversation
rows stay on the rail surface until hover; the active conversation is marked by
medium weight rather than a fill. Rename and delete actions appear on hover or
focus.

The footer contains a single compact “设置” action. There is no workspace block,
account switcher, plan badge, or theme toggle in the rail.

### Top bar

The top bar is intentionally quiet. When the sidebar is closed, a panel button
appears on the left; an open conversation's title follows it. Model and
appearance settings live in the sidebar's settings dialog.

The config warning banner appears directly beneath the top bar when the provider
is missing or the settings file could not be read. It uses the warning surface
and a text-link action, never a full-width alert color.

### Conversation thread

User turns are right-aligned bubbles with 24 px radius and a neutral fill.
Assistant turns are left-aligned, full width, and unboxed with no avatar.
Consecutive assistant and tool messages are grouped so a tool-using turn reads as
one answer.

Copy actions sit below the final text at low contrast. They are 28 px tall and
never compete with the answer.

### New-chat welcome

The new-chat state has one line: “xxx，今天聊点什么？”, or “今天聊点什么？” when
no display name is configured. There is no logo, guidance copy, or suggestion
grid. The composer follows directly beneath the greeting, and the pair sits on
the upper golden-section line rather than at the exact vertical center.

### Composer

A 52 px minimum-height pill with a 28 px radius. It contains:

- A 32 px circular attach button on the left.
- A borderless textarea that grows to 196 px.
- A compact text-only reasoning control between the textarea and send button.
  Its menu offers 自动 / 低 / 中 / 高, opens below on the welcome screen and above
  the docked composer, and keeps the last choice across sessions.
- A 32 px circular send button; while running it becomes a stop button.

The composer uses the level-2 shadow and a visible focus ring. There is no
persistent warning or shortcut hint below it.

### Personalization

The personalization page uses four compact preference rows instead of four
separate cards. Each row has a title and one-line description on the left and a
three-way segmented control on the right (228 px). The segmented control uses a
subtle track with a raised selected segment; labels are sentence case and the
selected state is never communicated by color alone.

The four controls are 回答风格 (简洁 / 平衡 / 详细), 亲切程度 (克制 / 自然 /
亲切), 热情程度 (冷静 / 适中 / 热情), and Emoji (不用 / 少量 / 适度). Below
them, a single full-width textarea holds custom instructions with a 5000
character limit and a small live counter aligned to the right. These values are
prompt modules, not model parameters.

### Memory

The memory page starts with three switch rows: 启用记忆, 自动摘要, and 历史搜索.
The latter two are visually disabled when the master switch is off. A status strip
on the subtle surface summarizes the current state in two lines: “摘要已就绪 /
正在更新摘要… / 等待后台更新”, followed by the last update time and pending
message count (for example “待处理 3 / 8 条消息”).

Two textareas follow, one for 手动记忆 (4000 characters) and one for 自动摘要
(2000 characters). The manual section is presented as user-authored; the summary
section is explicitly described as replaceable by the background task. Character
counters are right-aligned and tabular. “立即更新摘要” and “清空记忆” sit in the
status strip: the first is a secondary action with a refresh glyph, the second is
a quiet danger text action. Clearing opens the shared confirmation dialog.

If a background update wins the revision race, the page keeps the user's draft
visible and shows an inline conflict message with a “载入最新记忆” text link.
The app never silently overwrites unpublished edits. All memory errors stay
inline; they do not become toasts.

### Assistant turn phases

Each assistant turn has a working phase and an answer phase. While the agent is
working, a single status line shows the current activity. It starts as “正在思考…”,
then follows the in-progress `update_plan` step or running tool title, for example
“正在搜索…”. When the final answer starts streaming, that line collapses into a
text-only “思考了 Xs” disclosure above the answer. Only the final answer remains
visible by default.

The expanded activity is one chronological timeline rather than separate thinking
and tool sections: thinking fragments, plan status changes, and tool calls with
their results appear in the order they happened. After the first plan snapshot,
only changed steps are shown. Pre-tool assistant prose is not rendered. Plan steps
use a hollow circle for pending, a pulsing dot for the single in-progress step,
and a check for completed; all three states use the neutral text scale. Tool rows
keep a status dot, monospace arguments, and collapsible result panes. Running tool
rows open by default; completed rows collapse. The thinking duration is persisted
with the final answer so it survives a conversation reload.

### Code blocks

Code blocks use a 16 px radius, a tokenized header bar, a language label, and a
copy button. The body scrolls horizontally rather than wrapping. Syntax colors
are limited to a restrained GitHub-like palette in both themes.

### Dialogs

Settings and destructive confirmation share a 16 px radius, hairline border, and
level-5 shadow. Settings is 800 px wide and 620 px tall with a section sidebar
for 常规 / 个性化 / 记忆 / 搜索 / 模型. The content pane has no duplicate title
bar: the active sidebar row identifies the section. 常规 holds the display name
and three-way theme choice; 个性化 holds the preference rows and custom
instructions; 记忆 holds the memory switches, status strip, and two textareas;
搜索 holds the default-on web search switch, DuckDuckGo / Tavily choice, and
masked Tavily key; 模型 holds the URL, model, protocol, and masked API key. Each
key field shows its storage source in secondary text: system keychain,
environment variable, session-only fallback, or not saved. Keys backed by
`OPENAI_API_KEY` are read-only and have no removal action. When the keychain is
unavailable, an inline warning explains that a newly entered key lasts only for
the current session. Each section uses one primary information group instead of
repeating card borders and explanatory copy. Confirmation is 430 px wide with a
danger icon and a two-button footer.

### Toast

A small inverse surface at the bottom center. It is used for transient feedback
only: “附件将在后续版本支持”, “无法创建会话”, and similar. It never replaces
an inline error where the user needs to act.

### Boot screen

The launch screen is the same neutral canvas with the abstract mark and
a short “正在打开 Mira…” line. Fatal startup errors replace the line with a
24 px heading, the error text, and no decorative illustration.

## Do's and Don'ts

### Do

- Keep conversation content visually flat. Use spacing and type weight for
  hierarchy.
- Use neutral surfaces for grouping; reserve borders for controls and compact
  technical blocks.
- Keep the thread and composer on the same 768 px axis.
- Show tool status with both a dot and a text label.
- Keep memory status, pending message count, and revision conflicts visible
  inline in the memory page.
- Respect `prefers-color-scheme` on first launch and persist the user's override.
- Keep keyboard focus visible on every control, including inside the composer.
- Collapse non-essential motion under `prefers-reduced-motion`.

### Don't

- Don't wrap assistant answers in colored cards or apply shadows to messages.
- Don't introduce a second accent color for buttons, links, and status at the
  same time.
- Don't use all-caps labels or tracked-out eyebrow text.
- Don't use gradients, glows, or glassmorphism as decoration.
- Don't make every icon button filled; most actions are quiet until hover/focus.
- Don't hide the only destructive action behind hover on touch devices.
- Don't silently replace unsaved memory drafts when a background summary lands.
- Don't use OpenAI logos, wordmarks, or other OpenAI brand assets.

## Inspiration and attribution

The document structure follows the `DESIGN.md` format used by
[`VoltAgent/awesome-design-md`](https://github.com/VoltAgent/awesome-design-md),
with the Vercel entry serving as the closest neutral, monochrome reference.

The visual language is inspired by ChatGPT's conversation layout. This project is
an independent implementation and is not affiliated with, endorsed by, or
sponsored by OpenAI. No OpenAI trademarks, logos, or proprietary assets are
included.

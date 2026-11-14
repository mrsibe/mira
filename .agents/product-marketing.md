# Product Marketing Context

**Document version:** v2
**Last updated:** 2026-11-14

## Product Overview

**One-liner:** 本地优先、模型自选的 ChatGPT 桌面客户端。

**What it does:** Mira 把 ChatGPT 式的对话放进 Windows、macOS 和 Linux 桌面应用。对话、项目、设置和长期记忆保存在本机，模型请求只发送给用户自己配置的服务商。支持多家模型服务商和任意 OpenAI 兼容端点，自带 API Key，不需要 Mira 账号。适合想保留 ChatGPT 式体验、但希望自己选择模型和付费方式的人。

**Product category:** AI 对话桌面客户端 / BYOK LLM 客户端。用户寻找它时会搜“ChatGPT 桌面客户端”“ChatGPT 平替”“多模型客户端”“本地 AI 聊天客户端”。

**Product type:** 开源桌面软件（MIT），从 GitHub Releases 免费下载。

**Business model:** 无付费版本、无订阅。仓库属于个人项目（MrSibe）。

## Target Audience

**Target companies:** 不适用。产品面向个人用户。

**Decision-makers:** 使用者本人。

**Primary use case:** 想获得接近 ChatGPT 的日常对话体验，但不愿或无法承担订阅费用，希望改用更便宜的 API、免费额度或本地模型。

**Jobs to be done:**
- 用熟悉的对话方式完成日常问答、写作和编程，把费用控制在自己能承受的范围。
- 把多个服务商和多个模型收进一个客户端，随时切换，不必在网页和终端之间来回搬上下文。
- 让长期对话保持可找回：项目分组、全文搜索、自动记忆摘要。

**Use cases:**
- 学生、独立开发者、自由职业者等价格敏感用户，用 DeepSeek、Gemini 免费额度、OpenRouter 或本地 Ollama 替代订阅。
- 已经在购买 API 额度的人，需要一个顺手的桌面客户端。
- 隐私敏感用户不希望对话历史进入第三方云端。
- 长期使用 AI 的用户需要项目、记忆和可检索的历史来延续上下文。

## Personas

不适用（B2C / 个人工具）。

## Problems & Pain Points

**Core problem:** ChatGPT 订阅是一笔固定开销，官方客户端又把体验和订阅绑定，用户无法换成更便宜的模型。想继续用熟悉的对话方式，只能在免费额度、多个网页标签和 API 工具之间来回切换。

**Why alternatives fall short:**
- 官方客户端只服务自家模型，免费版还有模型和功能限制，无法接入其他服务商。
- 各家免费网页版额度分散，历史、项目和个性化设置难以长期维护。
- 通用多模型客户端常把配置、密钥和历史放在账号或云端，或界面承载过多功能。

**What it costs them:** 订阅费；API 用量账单的不确定；重复配置和切换的时间；上下文丢失；在多个工具之间分散的项目资料。

**Emotional tension:** 想用 AI，但担心订阅费成为长期负担；担心 API 账单失控；担心 API Key 和对话内容被第三方收集；担心工具越用越乱。

## Competitive Landscape

**Direct:** 官方 ChatGPT 桌面客户端 — 只支持 OpenAI 自家服务与账号体系，无法自带其他服务商密钥；免费版可用，但模型和功能受限。
**Direct:** ChatGPT、Claude、Gemini 等免费网页版 — 零配置、零费用，但额度分散、无法统一管理历史与项目。
**Direct:** Cherry Studio、Chatbox、Jan、LM Studio、Open WebUI 等多模型客户端 — 具体差异需逐个核对；初步观察是 Mira 更克制地整合项目、记忆、搜索与生成式界面。
**Secondary:** 直接用浏览器访问各家网页版 — 无需安装，但历史分散、无本地项目与统一检索。
**Indirect:** 终端工具与 API 脚本 — 灵活，但不适合日常阅读长回答和长期对话管理。

> 竞品判断未经系统调研，发布对外文案前需重新验证。

## Differentiation

**Key differentiators:**
- 订阅之外的选择：Mira 本身免费，不要求订阅，也不附带模型；用哪个服务商、花多少钱由用户决定，本地模型可以没有 API 费用。
- 本地优先：会话 JSONL、项目索引、记忆摘要和窗口状态都在本机；配置不含密钥。
- 密钥进系统钥匙串：macOS Keychain、Windows Credential Manager、Linux Secret Service，按模型分别保存。
- 多服务商与多模型：OpenAI、Azure OpenAI、Gemini、Anthropic、DeepSeek、OpenRouter、xAI、自定义 OpenAI 兼容端点。
- 长期使用的基础设施：项目分组、全文检索、BM25 历史记忆、自动摘要。
- 生成式界面：回答可以输出表格、图表和表单，组件受内置目录约束，不执行 HTML/JavaScript。
- 安静界面：两栏布局、可折叠思考过程、可调字体与侧栏，不堆叠入口。

**How we do it differently:** 用 Go + MyGo 构建原生桌面外壳，前端 React；聊天通过 MyGo Channel 传输，不在本地另起 HTTP/SSE 服务。配置 schema v2 支持多模型迁移，密钥与配置分离。用户保留 ChatGPT 式的交互，只替换模型和付费方式。

**Why that's better:** 用户掌握数据和密钥；换模型不用换客户端，也不必为用不上的功能持续付费；长期对话有结构可循。

**Why customers choose us:** 尚未有用户调研数据。[NEED: 真实用户选择原因 / 评价]

## Objections

| Objection | Response |
|-----------|----------|
| 为什么不用官方客户端？ | 官方客户端只支持 OpenAI 自家模型与账号；Mira 用你自己的 Key 接入多家服务商，界面和项目管理照常保留。 |
| Mira 是免费的吗？ | Mira 本身免费。模型费用由服务商按用量收取；Ollama 等本地模型不产生 API 费用。 |
| 不订阅 ChatGPT 也能用吗？ | 可以。你需要对应服务商的 API Key，或者在本机运行 Ollama、LM Studio。 |
| API 会不会比订阅贵？ | 取决于模型和用量。Mira 不替用户承诺更便宜，但可以选择低价模型、免费额度或本地模型，费用随时能换。 |
| 和 ChatGPT 免费版比呢？ | 免费版有独立的模型和额度限制，但零配置。Mira 的优势是可选模型、本地数据和长期项目管理。 |
| 我的对话会被上传吗？ | 对话、设置和记忆保存在本机；请求只发给你配置的服务商。Mira 没有云端服务。 |
| 密钥安全吗？ | 密钥存入系统钥匙串，不写入 `config.json`；前端只显示存储状态，不返回密钥。 |
| 换电脑怎么同步？ | 当前没有云同步，需要手动迁移数据目录。[待确认是否计划支持] |
| macOS 为什么提示应用损坏？ | 没有 Developer ID 签名和公证，需手动移除 quarantine 属性；README 提供命令。 |
| 是否支持手机？ | 目前只有桌面端。 |

**Anti-persona:** 需要开箱即用、不愿申请任何 API Key 的用户；依赖附件上传、语音等官方客户端独占功能的用户；需要团队协作、云端同步或移动端的用户；需要企业合规审计与集中管理的组织。

## Switching Dynamics

**Push:** 订阅费是固定负担；部分地区的支付方式不方便；官方客户端绑定单一服务商和模型。
**Pull:** 客户端本身免费；模型可按价格和用途自由更换；本地历史与记忆；项目组织；安静界面。
**Habit:** 已经习惯免费网页版或现有客户端；迁移历史需要成本。
**Anxiety:** 申请 API Key 和估算用量有门槛；担心 API 账单；担心新客户端不稳定、停止维护；担心数据迁移和密钥安全；macOS 未签名。

## Customer Language

**How they describe the problem:**
- 暂无真实用户访谈或评论。[NEED: 用户原话，来自 GitHub Issues、评论或访谈]
- 待验证的说法：“订阅太贵”“有没有 ChatGPT 平替”“哪个 API 便宜”“能不能用本地模型”。

**How they describe us:**
- 暂无。[NEED: 用户评价]

**Words to use:** 本地优先、自带密钥、按用量付费、模型自选、本地模型、桌面客户端、项目、长期记忆、系统钥匙串。
**Words to avoid:** 强大、一站式、无缝、颠覆、革命性、AI 赋能；避免把产品说成“完全免费”，因为模型调用仍可能产生费用。
**Glossary:**
| Term | Meaning |
|------|---------|
| 模型条目 | 一个服务商 + Base URL + 模型名 + 协议的配置，密钥单独存钥匙串 |
| 智能界面 | 模型输出受控组件规格，由前端渲染为表格、图表或表单 |
| 思维强度 | 请求时传给模型的 reasoning effort 档位 |
| 记忆摘要 | 每 8 条用户消息自动整理、注入后续对话的长期偏好摘要 |

## Brand Voice

| Attribute | Means | Doesn't mean | Example line |
|-----------|-------|--------------|--------------|
| 安静 | 少入口、少打扰，把空间留给对话 | 冷淡或功能贫乏 | “克制的界面把空间留给对话和回答。” |
| 具体 | 说清数据放哪、请求发给谁、开关控制什么 | 堆参数和术语 | “密钥按模型保存在系统钥匙串中，不会写入 config.json。” |
| 自主 | 用户自己选服务商、自己管密钥和数据 | 把配置负担丢给用户 | “使用 Mira 不需要注册账号。” |
| 诚实 | 说明限制：未签名、无同步、不能自更新的安装方式 | 自嘲或过度免责 | “Mira 当前没有 Developer ID 签名和公证。” |

**Tone by context:**
- Support: 直接给路径和命令，先解决问题。
- Launch: 说清这一版改变了什么，不夸大为“重新定义”。
- Errors: 说明原因和下一步，不隐藏限制。
- Sales: 产品本身免费，不做推销；重点放在下载与自助配置。
- Social: 可以更短、更轻松，但不使用夸张承诺。

**Terms we own:** 本地优先、模型自选、智能界面、思考过程、项目、长期记忆。
**Never use (and why):** “无缝”“强大”“一站式”（空泛，换任何竞品都成立）；“隐私绝对安全”（无法证明）；“完全免费”“免费使用 GPT-5”（Mira 免费，但模型调用仍由服务商计费）。

## Proof Points

**Metrics:** 暂无经核实的数据。[NEED: 下载量、Star 数、用户数]
**Customers:** 无公开客户。
**Testimonials:** 暂无。[NEED: 授权引用的用户评价]
**Value themes:**
| Theme | Proof |
|-------|-------|
| 数据留在本机 | 会话 JSONL、`config.json` 不含密钥、系统钥匙串、无 Mira 账号 |
| 成本可选 | 客户端免费；模型可换，本地模型不产生 API 费用 |
| 多模型自由切换 | schema v2 多模型配置、按模型保存密钥、环境变量回退 |
| 长期可用 | 项目、全文检索、BM25 记忆、索引自动重建 |
| 受控的生成式界面 | 组件目录白名单、不执行 HTML/JavaScript、状态存 localStorage |

## Goals

**Business goal:** 让想获得 ChatGPT 式体验、但不愿或无法订阅的用户，愿意下载并长期使用 Mira。[待确认]
**Conversion action:** 从 GitHub Releases 下载对应平台安装包。
**Current metrics:** 未知。[NEED: 下载量、留存、Star 数]

## Paid Ads (optional)

不投放广告。

## Learnings

*Decisions and tested lessons. Newest first. Keep it to about 20 active entries.*

| Date | Learning | Evidence | Applies to | Status |
|------|----------|----------|------------|--------|
| 2026-11-14 | 目标用户是想获得 ChatGPT 体验、但不愿或无法承担订阅费用的人；文案应围绕“模型自选、按用量付费”，避免把“免费”当唯一卖点 | 创始人指示 | positioning, README, copywriting | active |
| | | | | |

## Changelog

- v2 (2026-11-14) — 按创始人指示重写目标用户与核心问题：面向想获得 ChatGPT 体验但无力或不欲订阅的用户；补充价格敏感场景、API 费用异议与“免费”措辞边界。竞品按创始人确认保留。
- v1 (2026-11-14) — Initial context, auto-drafted from README, CHANGELOG, DEVELOPMENT.md and source. Unverified marketing claims marked `[NEED]`.

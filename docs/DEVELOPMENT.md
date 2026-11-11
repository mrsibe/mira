# Mira 开发指南

本指南面向从源码构建、调试或发布 Mira 的开发者。产品介绍、下载安装和功能概览见
[README](../README.md)。

## 技术栈

| 层 | 选择 |
| --- | --- |
| 桌面外壳 | MyGo `v0.8.3` |
| Go ↔ TypeScript | MyGo typed bindings + `Channel[T]` |
| Agent | GOAT `v0.2.0` |
| 前端 | React 19、TypeScript、Vite |
| Markdown | `react-markdown`、`remark-gfm`、`rehype-highlight` |
| 图标 | `lucide-react` |

## 环境要求

- Go 1.27.4 或更高版本
- Bun 1.2 或更高版本
- 对应平台的系统 WebView
  - macOS：WKWebView
  - Windows：WebView2 Runtime
  - Linux：WebKitGTK

在 Linux 上从源码构建时，还需要 GTK、WebKitGTK 和 Secret Service 的开发包。
CI 使用：

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libsecret-1-dev
```

## 启动开发环境

```bash
bun install
bun run dev
```

`bun run dev` 会同时启动 MyGo 开发模式与 Vite。只调试网页界面时可以运行
`bun run dev:web`，但聊天、钥匙串、窗口和更新等能力依赖真实的 MyGo 运行时。

修改 Go 侧导出给前端的绑定后，运行：

```bash
bun run generate
```

修改 `src/genui/catalog.ts` 后，同步后端嵌入的组件提示词：

```bash
bun run generate:ui-prompt
```

## 模型与搜索配置

首次启动后，点击侧栏底部的“设置”。设置分为外观 / 个性化 / 记忆 / 搜索 /
模型 / 关于六个栏目。在“模型”中填写：

- **Base URL**：例如 `https://api.openai.com/v1`
- **模型**：例如 `gpt-5`
- **API 协议**：Responses 或 Chat Completions
- **API Key**：保存到系统钥匙串；本地无鉴权服务可以留空

非敏感设置从本机 `config.json` 读取，字段缺失时回退到环境变量。模型 API Key
的读取优先级为系统钥匙串 > 环境变量 > 未配置：

| 环境变量 | 说明 |
| --- | --- |
| `OPENAI_BASE_URL` | OpenAI 兼容接口地址 |
| `OPENAI_API_KEY` | API Key，只读回退；输入新密钥可覆盖 |
| `OPENAI_MODEL` | 模型名称 |
| `MIRA_PROTOCOL` | `responses` 或 `completions` |

配置文件权限为 `0600`，保存时使用临时文件加原子替换。前端返回的设置视图不
包含密钥，只返回 `keychain` / `environment` / `session` / `none` 四种存储状态。
环境变量提供的密钥不会写入钥匙串，也不能在设置中清除。

系统钥匙串通过 MyGo `Secrets` 访问。Linux 需要可用的 Secret Service
（如 GNOME Keyring 或 KWallet）和会话总线；不可用时界面会明确提示密钥仅保存在
本次会话。

“搜索”栏目默认开启联网搜索：

- **DuckDuckGo**：默认供应商，不需要 API Key。网页结果被反爬验证拦截时会
  尝试 DuckDuckGo Instant Answer。
- **Tavily**：需要填写 Tavily API Key。留空保存时不会修改已保存的 Key；
  清除 Key 会自动切回 DuckDuckGo。

`config.json` 中与搜索和个性化相关的字段：

| 字段 | 说明 |
| --- | --- |
| `webSearchEnabled` | 是否向 Agent 暴露 `web_search` 工具，默认 `true` |
| `searchProvider` | `duckduckgo` 或 `tavily`，默认 `duckduckgo` |
| `personality` | `professional` / `friendly` / `candid` / `quirky` / `efficient` / `cynical`，默认 `friendly` |
| `responseStyle` | `concise` / `balanced` / `detailed` |
| `warmth` | `reserved` / `balanced` / `friendly` |
| `enthusiasm` | `calm` / `balanced` / `enthusiastic` |
| `emoji` | `none` / `sparing` / `frequent` |
| `customInstructions` | 全局自定义指令，最多 5000 字 |
| `memoryEnabled` | 记忆总开关，默认 `true` |
| `memorySummaryEnabled` | 生成并注入自动摘要，默认 `true` |
| `memorySearchEnabled` | 暴露 `search_memory` 工具，默认 `true` |
| `intelligentUIEnabled` | 智能界面总开关，默认 `true` |

## 智能界面

“设置 → 个性化”中的“智能界面”默认开启。开启时，
`internal/prompt` 会在核心规则和写作规范之后注入受控组件目录；关闭时该模块完全省略，
前端也不会渲染界面。无论开关状态如何，前端都会从回答中移除 `spec` 围栏，
避免把原始 JSONL 暴露给用户。

模型输出协议是：每条回答最多输出一个以 `spec` 标记的围栏代码块，围栏内为
json-render JSONL/RFC 6902 补丁。围栏可以出现在回答中的任意位置，前端会按
Markdown 与界面的原始顺序交错渲染；界面已经承载的表格、图表或表单不应在
Markdown 中重复。
界面只允许使用 `src/genui/catalog.ts` 注册的组件；组件不能执行 HTML、JavaScript
或任意 CSS。交互分为本地状态操作和受控的 `ask_agent` 动作，不直接授予模型
系统权限。

界面状态按会话和消息索引保存在 WebView 的 `localStorage` 中，键格式为
`mira:genui:v1:<conversationId>:<messageIndex>`，单条上限 64 KB。会话 JSONL
格式不存储界面状态。

## 数据目录

数据保存在 MyGo 的 `PathUserData` 目录：

| 平台 | 路径 |
| --- | --- |
| macOS | `~/Library/Application Support/Mira` |
| Linux | `~/.config/Mira`（遵循 `XDG_CONFIG_HOME`） |
| Windows | `%AppData%\Mira` |

主要文件：

```text
config.json                  # 模型与界面设置，不含密钥（权限 0600）
conversations/index.json     # 会话索引
conversations/<id>.jsonl     # 每个会话的消息、用量
conversations/projects.json  # 项目分类，按创建时间排列
memory/USER.md               # 自动记忆摘要（带分区标记）
memory/state.json            # 摘要时间、待处理轮数与 revision
window-state.json            # 窗口大小与位置
updater.json                 # 自动检查设置、上次检查时间与跳过的版本
```

会话文件采用 JSONL：第一行是头部，其后每行是一个消息、usage 或项目移动记录。
索引丢失或损坏时会从 `.jsonl` 文件自动重建项目归属。`projects.json` 使用独立
版本号，缺失时按空项目列表处理。

`USER.md` 使用 `MIRA:AUTO` 标记保存自动摘要。摘要分区为“回答偏好”“重要话题”
“用户洞察”“当前项目与目标”。启动时会把旧版的“长期偏好”“用户画像”
“重要约束”无损迁移到新分区，无法识别的内容原样保留；迁移只重写 `USER.md`，
不改变 revision 或更新时间。旧版本写入的 `MIRA:MANUAL` 区块会被忽略，并在
下一次保存或更新摘要时移除。`state.json` 记录
`pendingUserTurns`、`lastSummarizedAt`、`summaryUpdatedAt`、`lastError` 和
`revision`。

### 记忆上下文与缓存

长期记忆不写入每轮重建的系统提示词。`internal/prompt.Build` 只生成稳定前缀：
GPT-5.5 Instant 核心规则、Thinking 写作规范、智能界面目录和工具说明；之后才是
会随设置变化的人格、表达偏好和自定义指令。人格提示词、来源与取舍记录在
`internal/prompt/SOURCES.md`。记忆摘要由服务层作为隐藏的 `<mira_context>`
消息追加到会话历史，并带有 `contextKind=memory` 与内容哈希。

- 摘要未变化时不会重复追加。
- 摘要更新、清除或关闭记忆时，追加一个新版本；旧版本仍保留在历史中，模型以
  最新版本为准。
- 隐藏上下文进入模型历史，但不会出现在前端消息列表、会话标题、预览、
  消息计数或 BM25 记忆中。
- OpenAI 请求使用会话 ID 作为 `prompt_cache_key`。稳定系统提示词、确定性排序
  的工具定义和追加式历史共同保证旧前缀可复用。

## 架构

```text
main.go
  └── internal/service.Service        # MyGo 绑定为 Mira
        ├── internal/config.Manager   # 设置读写与校验
        ├── internal/keychain.Store   # 系统钥匙串适配
        ├── internal/store.Store      # JSONL 会话持久化
        ├── internal/memory.Manager   # USER.md、摘要任务与 BM25 索引
        ├── internal/prompt           # 模块化系统提示词
        ├── internal/search           # DuckDuckGo / Tavily 搜索
        └── internal/chat.Engine      # 构造 GOAT Agent 与工具

src/App.tsx
  ├── components/Sidebar.tsx         # 项目 / 时间分组的会话列表
  ├── components/SidebarMenus.tsx    # 对话与项目三点菜单
  ├── components/Thread.tsx          # 会话滚动容器
  │     └── components/thread/       # 消息、思考、计划、工具卡
  ├── components/Composer.tsx
  ├── components/SettingsDialog.tsx
  │     └── components/settings/     # 设置栏目与记忆编辑 hook
  ├── genui/                         # 组件目录、JSONL 解析、渲染与状态持久化
  ├── ui/                            # Button、Modal、Field 等 primitives
  ├── lib/activity.ts                # 流式与历史消息的线性时间线
  └── styles/                        # tokens、布局、对话、设置和响应式样式
```

### 一次发送的生命周期

1. 前端调用 `Mira.send(conversationId, content, channel)`。
2. 服务端读取会话与隐藏上下文；记忆版本变化时先追加新的 `<mira_context>`，
   再构造 `agent.Session`。
3. GOAT 流式产生 message、thinking、tool、usage 事件；`update_plan` 工具调用
   会被前端解析为计划列表，不显示为普通工具卡。
4. 服务端把事件写入 `*mygo.Channel[chat.ChatEvent]`。
5. 每条完成的消息与 usage 写入 JSONL。
6. 用户回合计入记忆待处理数；达到阈值后在后台生成摘要并更新 BM25 索引。
7. 前端消费 channel 并实时渲染；结束后重新加载会话以对齐持久化状态。

同一时间只允许一个 Agent run。其他发送请求会返回“正在生成回复，请先停止当前
回复”。停止操作会取消当前 `context.Context`，GOAT 产出的部分回答仍会持久化。

单次 run 有 64 个模型轮次的安全上限。GOAT 按模型轮次而不是工具调用次数计数，
计划更新也会消耗轮次；触顶时当前历史仍会保存，回复“继续”即可从已有进度接着
执行。

### 为什么使用 Channel

MyGo 的页面运行在系统 WebView 中。聊天更新通过托管 Channel 传输，而不是在
前端启动本地 HTTP/SSE 服务；这样调用关系、类型和生命周期都由 MyGo 管理。

## 自动更新

Mira 使用 MyGo 的签名更新器，更新包发布在
[GitHub Releases](https://github.com/mrsibe/mira/releases)。启动后会自动检查，
发现新版本时打开中文更新窗口，由用户选择安装、稍后提醒或跳过该版本。也可以在
“设置 → 关于”中手动检查。

自动更新要求应用构建时配置了更新公钥，并且运行目录可写。以下情况无法自更新：

- 开发模式运行；
- 由包管理器安装（例如 Debian 包或 Arch 包）；
- 安装目录属于其他用户且当前用户不可写。

这些情况下“关于”会说明原因，请从 GitHub Releases 下载对应平台的新版本。

macOS 包没有 Developer ID 签名和公证。下载后解压，把 `Mira.app` 移到
`~/Applications`，必要时执行：

```bash
xattr -dr com.apple.quarantine ~/Applications/Mira.app
```

更新器需要替换应用包，所以 `Mira.app` 必须位于当前用户可写的目录。

## 发布流程

更新签名私钥默认保存在 `~/.config/mygo/update-keys/mygo-update.key`，不能提交到
仓库或安装包。首次配置仓库时把它写入 Actions Secret：

```bash
gh secret set MYGO_UPDATER_PRIVATE_KEY --repo mrsibe/mira \
  < ~/.config/mygo/update-keys/mygo-update.key
```

发布新版本时：

1. 更新 `mygo.config.ts` 的 `version`。
2. 在 `CHANGELOG.md` 添加对应的 `## <version>` 段落。
3. 提交并推送代码。
4. 创建并推送版本标签，CI 会自动签名构建、上传产物并发布 Release。

```bash
git tag v0.3.0
git push origin v0.3.0
```

`.github/workflows/release.yml` 会校验标签与 `mygo.config.ts` 版本一致，构建
Windows 安装包、Linux Debian/Arch 分发包、macOS 应用包及签名更新清单，最后发布
`v<version>` Release。

## 测试与验证

```bash
gofmt -w .
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
bun run check
npm_config_registry=https://registry.npmjs.org bun audit
```

Go 测试覆盖设置持久化、钥匙串读写与删除、环境变量优先级、钥匙串不可用时的
会话回退、密钥不落盘、提示词稳定前缀与模块顺序、追加式记忆上下文与缓存前缀
复用、BM25 中英文检索与当前会话排除、隐藏上下文过滤、USER.md 兼容与 revision
冲突、摘要阈值与失败保留旧摘要、首次 40 回合回填、搜索 Provider 解析与降级、
工具注册、会话增删改查、项目分组与迁移、项目级联删除、流式事件、取消后的
部分回答持久化，以及单 run 锁。

前端测试覆盖设置开关、智能界面解析与渲染、活动时间线和基础组件；依赖真实
MyGo 运行时的视觉与端到端行为仍需通过 `bun run dev` 或构建后的桌面应用确认。

## 当前边界

- 多用户、账号和云同步尚未实现。
- 图片与文件附件尚未实现，输入框中的附件按钮会提示后续支持。
- 消息编辑、重新生成、分支和分享尚未实现。
- 模型自动生成标题尚未实现；标题取自首条用户消息。
- 自由 HTML/JavaScript、自定义组件和跨设备界面状态同步尚未实现。
- 搜索缓存、独立来源卡片和网页正文抓取尚未实现。
- 记忆检索目前使用 BM25，没有向量检索或跨会话语义重排。
- Windows / macOS 代码签名和 macOS 公证尚未完成。

## 相关文档

- [README](../README.md)
- [设计规范](../DESIGN.md)
- [更新日志](../CHANGELOG.md)

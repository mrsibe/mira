# Mira

Mira 是一个本地优先、单用户的 ChatGPT 风格桌面客户端。桌面外壳与
Go↔TypeScript 桥接使用 [MyGo](https://github.com/egoist/mygo)，聊天运行时使用
[GOAT](https://github.com/mrsibe/goat)。前端是 React + TypeScript，对话与设置
保存在本机。

当前版本是 MVP：专注于“配置模型 → 新建对话 → 流式回答 → 工具调用 → 本地历史”
这条主路径。

## 功能

- ChatGPT 风格的双栏布局、用户气泡、助手 Markdown 回答和自适应输入框。
- 新会话只显示问候语和居中输入框，有称呼时显示“称呼，今天聊点什么？”，
  否则只显示“今天聊点什么？”，不展示建议卡片。
- 浅色 / 深色 / 跟随系统主题，首屏无闪烁。
- OpenAI 兼容接口，支持 Responses 与 Chat Completions 两种协议。
- 输入框内可切换思维强度（自动 / 低 / 中 / 高）；自动档使用供应商默认值，
  最后选择会保存在本机。
- GOAT Agent 循环：多轮历史、工具调用、计划步骤、thinking 事件、用量统计。
- 内置 `update_plan` 工具：模型为多步骤任务维护 todo 计划。思考时只显示一行
  当前状态，并随计划步骤或工具调用切换；最终回答开始时折叠为“思考了 Xs”入口，
  展开后按发生时间线性查看思考片段、计划变化与工具结果。耗时随会话保存在本机。
- 内置 `get_current_time` 工具，演示 Agent 工具卡片的运行与结果展示。
- 联网搜索工具，默认使用无需 Key 的 DuckDuckGo，也可在设置中切换到 Tavily；
  搜索默认开启，可在设置中关闭。
- 系统提示词按模块组装：核心规则 → 表达偏好 → 自定义指令 → 长期记忆 →
  工具规则。表达偏好和自定义指令都可以在“个性化”中调整。
- 长期记忆由自动摘要组成：每累计 8 条用户消息在后台重新整理一次，首次启用
  时会回看最近 40 个用户回合。
- `search_memory` 工具用 BM25 搜索历史会话，默认排除当前会话；索引在内存中
  惰性构建，会话变化后自动重建。
- 流式输出、停止生成、部分回答持久化。
- 本地会话列表、搜索、项目分类与按时间分组；对话可通过三点菜单重命名、
  移动分类或删除。
- 项目支持新建、重命名和删除。项目按创建时间排列；删除项目会同时删除其中
  的全部对话，未归入项目的对话继续按时间分组显示。
- 模型与搜索 API Key 保存在系统钥匙串，前端只能看到“是否已配置”和存储来源；
  `config.json` 不包含密钥。
- 本地无鉴权服务（如 Ollama、LM Studio）无需 API Key 即可使用。

## 技术栈

| 层 | 选择 |
|---|---|
| 桌面外壳 | MyGo `v0.8.3` |
| Go ↔ TS | MyGo typed bindings + `Channel[T]` |
| Agent | GOAT `v0.2.0` |
| 前端 | React 19、TypeScript、Vite |
| Markdown | `react-markdown`、`remark-gfm`、`rehype-highlight` |
| 图标 | `lucide-react` |

## 环境要求

- Go 1.27 或更高版本
- Bun 1.2 或更高版本
- 对应平台的系统 WebView
  - macOS: WKWebView
  - Windows: WebView2 Runtime
  - Linux: WebKitGTK

## 快速开始

```bash
bun install
bun run dev
```

首次启动后，点击侧栏底部的“设置”。设置分为外观 / 个性化 / 记忆 / 搜索 /
模型 / 关于六个栏目，在“模型”中填写：

- **Base URL**：例如 `https://api.openai.com/v1`
- **模型**：例如 `gpt-5`
- **API 协议**：Responses 或 Chat Completions
- **API Key**：保存到系统钥匙串；使用本地无鉴权服务时可以留空

API Key 使用 macOS Keychain、Windows Credential Manager 或 Linux Secret
Service。系统钥匙串不可用时不会退回明文文件，密钥只保留在本次会话，重启后
需要重新输入；开发模式和安装后的应用使用不同的钥匙串条目。

“外观”可以切换浅色 / 深色 / 跟随系统，并同步网页、原生窗口和 Windows
窗口控制按钮。“个性化”可以填写新会话使用的称呼，调整回答风格、亲切程度、
热情程度、Emoji 频率，并填写最多 5000 字的自定义指令。称呼留空时首屏只
显示“今天聊点什么？”。“关于”中可检查更新。

“记忆”栏目默认开启全部三项：

- **启用记忆**：总开关。关闭后不再注入记忆，也不再暴露 `search_memory`。
- **自动摘要**：累计 8 条用户消息后，复用当前聊天模型在后台整理出新的长期
  摘要；关闭后不再生成或注入自动摘要区块。
- **历史搜索**：允许 Agent 在其他历史会话中做 BM25 检索。

自动摘要可以直接编辑；“更新摘要”会同步重新生成，“清空记忆”会删除摘要。
后端使用 revision 防止后台更新覆盖尚未保存的编辑。

“搜索”栏目默认开启联网搜索：

- **DuckDuckGo**：默认供应商，不需要 API Key。网页结果被反爬验证拦截时会
  尝试 DuckDuckGo Instant Answer。
- **Tavily**：需要填写 Tavily API Key。留空保存时不会修改已保存的 Key；
  清除 Key 会自动切回 DuckDuckGo。

如果 Go 模块下载受限，可以显式指定代理：

```bash
GOPROXY=https://proxy.golang.org,direct go mod download
```

## 配置

非敏感设置从本机 `config.json` 读取；字段缺失时回退到环境变量。模型 API Key
的读取优先级为系统钥匙串 > 环境变量 > 未配置：

| 环境变量 | 说明 |
|---|---|
| `OPENAI_BASE_URL` | OpenAI 兼容接口地址 |
| `OPENAI_API_KEY` | API Key，只读回退；输入新密钥可覆盖 |
| `OPENAI_MODEL` | 模型名称 |
| `MIRA_PROTOCOL` | `responses` 或 `completions` |

配置文件权限为 `0600`。设置保存时使用临时文件 + 原子替换；前端返回的
`View` 不包含密钥，只返回 `keychain` / `environment` / `session` / `none`
四种存储状态。环境变量提供的密钥不会写入钥匙串，也不能在设置中清除。

系统钥匙串通过 MyGo `Secrets` 访问。Linux 需要可用的 Secret Service
（如 GNOME Keyring 或 KWallet）和会话总线；不可用时界面会明确提示密钥仅保存在
本次会话。

## 数据目录

数据保存在 MyGo 的 `PathUserData` 目录：

| 平台 | 路径 |
|---|---|
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

`config.json` 中与搜索相关的字段：

| 字段 | 说明 |
|---|---|
| `webSearchEnabled` | 是否向 Agent 暴露 `web_search` 工具，默认 `true` |
| `searchProvider` | `duckduckgo` 或 `tavily`，默认 `duckduckgo` |
| `responseStyle` | `concise` / `balanced` / `detailed` |
| `warmth` | `reserved` / `balanced` / `friendly` |
| `enthusiasm` | `calm` / `balanced` / `enthusiastic` |
| `emoji` | `none` / `sparing` / `frequent` |
| `customInstructions` | 全局自定义指令，最多 5000 字 |
| `memoryEnabled` | 记忆总开关，默认 `true` |
| `memorySummaryEnabled` | 生成并注入自动摘要，默认 `true` |
| `memorySearchEnabled` | 暴露 `search_memory` 工具，默认 `true` |

模型 API Key 和 Tavily API Key 不写入 `config.json`。如果 Tavily 密钥只存在于
本次会话，重启后会自动回退到 DuckDuckGo。

`USER.md` 使用 `MIRA:AUTO` 标记保存自动摘要。旧版本写入的 `MIRA:MANUAL` 区块
会被忽略，并在下一次保存或更新摘要时移除。`state.json` 记录
`pendingUserTurns`、`lastSummarizedAt`、`summaryUpdatedAt`、`lastError` 和
`revision`。

## 常用命令

| 命令 | 说明 |
|---|---|
| `bun run dev` | 启动 MyGo 开发模式与 Vite |
| `bun run build` | 构建桌面应用 |
| `bun run generate` | 从 Go 绑定生成 `src/mygo.ts` |
| `bun run typecheck` | TypeScript 类型检查 |
| `bun run build:web` | 仅构建前端 |
| `bun run lint` | Biome 格式与静态检查 |
| `bun run test` | Vitest 前端测试 |
| `bun run check` | lint、测试、类型检查与前端构建的完整门禁 |
| `go test ./...` | Go 单元测试 |
| `go test -race ./...` | Go 竞态检测 |
| `go vet ./...` | Go 静态检查 |

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

### 发布流程

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
git tag v0.2.0
git push origin v0.2.0
```

`.github/workflows/release.yml` 会校验标签与 `mygo.config.ts` 版本一致，构建
Windows 安装包、Linux Debian/Arch 分发包、macOS 应用包及签名更新清单，最后发布
`v<version>` Release。

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
  ├── components/Sidebar.tsx         # 项目 / 时间分组的会话列表与整行切换
  ├── components/SidebarMenus.tsx    # 对话与项目三点菜单
  ├── components/Thread.tsx          # 会话滚动容器
  │     └── components/thread/       # 消息、思考、计划、工具卡
  ├── components/Composer.tsx
  ├── components/SettingsDialog.tsx
  │     └── components/settings/     # 设置栏目与记忆编辑 hook
  ├── ui/                            # Button、Modal、Field 等内部 primitives
  ├── lib/activity.ts                # 流式与历史消息的线性时间线
  └── styles/                        # tokens、布局、对话、设置、响应式样式
```

### 一次发送的生命周期

1. 前端调用 `Mira.send(conversationId, content, channel)`。
2. 服务端读取会话历史，构造 `agent.Session`。
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

## MVP 边界

暂不包含：

- 多用户、账号、云同步
- 图片 / 文件附件（按钮会提示后续支持）
- 消息编辑、重新生成、分支与分享
- 模型自动生成标题（标题取自首条用户消息）
- 可视化回答、模型参数面板
- 搜索缓存、独立来源卡片、网页正文抓取
- 向量检索与跨会话的语义重排（当前只用 BM25）
- Windows / macOS 代码签名与 macOS 公证

## 验证

```bash
gofmt -w .
go test -race ./...
go vet ./...
bun run lint
bun run test
bun run generate
bun run typecheck
bun run build:web
bun run build
```

Go 测试覆盖设置持久化、钥匙串读写与删除、环境变量优先级、钥匙串不可用时的
会话回退、密钥不落盘、提示词模块顺序、BM25 中英文检索与当前会话排除、USER.md
兼容与 revision 冲突、摘要阈值与失败保留旧摘要、首次 40 回合回填、搜索
Provider 解析与降级、工具注册、会话增删改查、项目分组与迁移、项目级联删除、
流式事件、取消后的部分回答持久化，以及单 run 锁。
前端依赖真实的 MyGo 运行时，因此视觉与端到端行为需通过 `bun run dev` 或
构建后的桌面应用确认。

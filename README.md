<p align="center">
  <img src="resources/icon.png" width="88" height="88" alt="Mira">
</p>

<h1 align="center">Mira</h1>

<p align="center">
  本地优先的 ChatGPT 桌面客户端
</p>

<p align="center">
  <a href="https://github.com/mrsibe/mira/releases/latest">
    <img src="https://img.shields.io/github/v/release/mrsibe/mira?label=release" alt="Latest release">
  </a>
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-6b7280" alt="Windows, macOS and Linux">
  <a href="LICENSE">
    <img src="https://img.shields.io/github/license/mrsibe/mira" alt="MIT license">
  </a>
</p>

<p align="center">
  <img src="docs/images/mira-dark.png" alt="Mira 深色主题的新会话界面" width="100%">
</p>

Mira 是一个本地优先的 ChatGPT 桌面客户端。熟悉的对话界面之外，它还把项目分类、
长期记忆、联网搜索和工具调用放进了一个安静、专注的桌面应用。

对话、设置和记忆保存在本机；模型请求只会发送到你配置的服务商。使用 Mira
不需要注册账号，也不需要经过 Mira 的云端服务。

## 为什么是 Mira

- **安静，不抢注意力**：克制的界面把空间留给对话和回答。
- **本地优先**：没有 Mira 账号，历史、设置和记忆默认只保存在这台电脑上。
- **不只是聊天**：搜索、记忆检索、计划和工具调用都在同一条时间线里完成。
- **适合长期使用**：项目、长期记忆和个性化指令让对话不会越用越乱。

## 下载

前往 [最新 Release](https://github.com/mrsibe/mira/releases/latest) 下载对应平台的版本。

| 平台 | 文件 | 说明 |
| --- | --- | --- |
| Windows 10/11 | `mira-setup-{version}-amd64.exe` | 安装程序，支持自动更新 |
| macOS | `mira-{version}-darwin-arm64.tar.gz` / `mira-{version}-darwin-amd64.tar.gz` | 解压后将 `Mira.app` 移到 `~/Applications` |
| Linux | `mira_{version}_amd64.deb` / `mira-{version}-linux-amd64.tar.gz` | Debian / Ubuntu 或通用归档 |

<details>
<summary>macOS 首次打开时提示“已损坏”或无法验证？</summary>

Mira 当前没有 Developer ID 签名和公证。把应用放到当前用户可写的目录后执行：

```bash
xattr -dr com.apple.quarantine ~/Applications/Mira.app
```

</details>

## 开始使用

1. 打开 Mira，点击左下角的“设置”。
2. 在“模型”中填写 Base URL、模型名称和 API Key。
3. 保存后点击“新对话”即可开始。

支持 OpenAI Responses 和 Chat Completions 兼容接口。Ollama、LM Studio 等本地
无鉴权服务通常不需要填写 API Key。密钥优先保存在系统钥匙串中，不会写入
`config.json`。

## 核心能力

### 对话体验

- 熟悉的两栏布局、Markdown 回答、代码高亮和流式输出。
- 停止生成、复制回答，以及浅色 / 深色 / 跟随系统主题。
- 输入框内直接切换思维强度，最后的选择会保存在本机。
- 智能界面默认开启，可按回答生成表格、图表、表单等交互组件；可在
  “设置 → 个性化”中关闭。

### Agent 能力

- 思考过程折叠为“思考了 Xs”，展开后按时间查看思考、计划和工具调用。
- 联网搜索默认使用 DuckDuckGo，也可以在设置中切换到 Tavily。
- 历史记忆通过 BM25 检索，模型需要时可以主动查找相关对话。

### 记忆与个性化

- 每累计 8 条用户消息，自动整理一次长期记忆摘要。
- 可以调整称呼、回答风格、亲切程度、热情程度和 Emoji 频率。
- 支持最多 5000 字的自定义指令，以及直接编辑当前记忆摘要。
- 关闭智能界面后，Mira 不再注入生成式界面提示词，也不会渲染已有界面，
  只保留普通 Markdown 回答。

### 项目与本地数据

- 用项目整理对话，支持新建、重命名、删除和移动。
- 对话以 JSONL 保存在本机，索引损坏时可以自动重建。
- 模型和搜索密钥优先保存在 macOS Keychain、Windows Credential Manager 或
  Linux Secret Service。

## 开发

需要 Go 1.27.4+、Bun 1.2+，以及对应平台的系统 WebView。

```bash
bun install
bun run dev
```

完整的环境准备、配置、数据格式、架构、测试和发布流程见
[开发指南](docs/DEVELOPMENT.md)。

## 文档

| 文档 | 内容 |
| --- | --- |
| [开发指南](docs/DEVELOPMENT.md) | 从源码构建、配置、架构、测试与发布 |
| [设计规范](DESIGN.md) | 布局、颜色、排版与交互原则 |
| [更新日志](CHANGELOG.md) | 每个版本的变化 |
| [GitHub Releases](https://github.com/mrsibe/mira/releases) | 安装包与签名更新 |

## License

[MIT](LICENSE) © 2026 MrSibe

Mira 与 OpenAI 无关联。ChatGPT 是 OpenAI 的商标。

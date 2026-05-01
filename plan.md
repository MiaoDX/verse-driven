# Verse-Driven Coding Agent — 开发计划

> 一个让 Claude Code 和 Codex 都能用得起来的"经文 + 代码"交互工具。
> 核心目标：写代码时可以主动引用经典作为反思框架，或在任务结束后看到一条简短 recap，
> 但不影响 coding agent 本身的工程质量。

---

## 1. 项目愿景

把《圣经》《古兰经》《道德经》《心经》等经典文本，做成 coding agent 的**可选提示层**。
不是研经助手，不是宗教 AI，而是：

- **写代码时**，可以选择引用一段经文作为本次任务的反思框架；
- **任务结束后**，可以看到一条与经文相关的 recap 卡片帮助记忆；
- **默认状态下静默**，永远不打扰、不偷偷修改 coding 行为。

灵感谱系上，承接 Tao of Programming、Unix Koans of Master Foo、Zen of Python
这条"用古典风格谈技术"的传统，但第一次把它做成 **agent-safe 的注入协议**。

---

## 2. 核心设计原则

### 2.1 两种使用模式，物理上隔离

| 模式 | 触发方式 | 是否进入模型上下文 | 用例 |
|---|---|---|---|
| **Mode A: Manual** | 用户主动输入 marker（`/bible John 3:16` 或 `[[bible:John 3:16]]`） | ✅ 仅当前 turn，下一 turn 自动失效 | 想用某段经文作为本次任务的反思框架 |
| **Mode B: Recap** | 任务/答案结束后自动触发 | ❌ 永远不进入模型 transcript | 任务完成后的提醒 / 记忆训练 |

**关键判断标准**：一段文字会不会出现在下一次 model call 的输入里。
会 → 不算干净；不会 → 安全。

### 2.2 严禁的实现路径

为了不污染 coding 上下文，下列"看起来很方便"的路径**全部禁用**：

- 把经文正文写进 output style / skill / AGENTS.md / CLAUDE.md / `model_instructions_file`
- SessionStart hook 注入 system prompt
- 默认让 output style 在每轮答案末尾追加 recap 段落（这段会进会话历史）
- 任何 always-run 的 prompt-level hook
- 远程 MCP 在每次 prompt 前都 fetch 一次

### 2.3 推荐路径

- **Mode A**：通过 hook（`UserPromptExpansion` / `UserPromptSubmit`）拦截 marker
  → 调本地 MCP / CLI lookup → 预览 → 用户确认 → 注入仅当前 turn 的 developer context。
- **Mode B**：Stop hook（Claude Code）或 shell wrapper（Codex）
  → 调本地 CLI → **直接 print 到终端** → exit。
  recap 文本永远不进入下一次 model call 的输入。

---

## 3. 架构

```
                  ┌─ packs/        (KJV / 道德经 / 心经 / Quran ...)
   核心（共享）──┼─ resolver/     (引用解析 + checksum 校验)
                  └─ mcp-server/   (本地 stdio MCP)
                            │
              ┌─────────────┴─────────────┐
              │                           │
      Claude 适配层                  Codex 适配层
      ├─ output-style              ├─ skill (scripture-lookup)
      │   (只管风格，不含经文)       │   (allow_implicit_invocation: false)
      ├─ skill (verse-inject)      ├─ hook UserPromptSubmit (Mode A)
      ├─ hook UserPromptExpansion  └─ shell wrapper (Mode B recap)
      │   (Mode A)
      └─ hook Stop (Mode B recap)
```

### 3.1 单二进制 + 本地 stdio MCP

参考 [`Gentleman-Programming/engram`](https://github.com/Gentleman-Programming/engram) 的设计：
**一个 Go 二进制，三种调用方式**。

```bash
scripture-mcp serve                              # MCP stdio 服务，给 cc/codex 用
scripture-mcp lookup "John 3:16" --format=json   # CLI，给 hook 脚本用
scripture-mcp recap --tradition=dao --terminal   # 终端 print，给 Mode B 用
```

为什么选这个组合：

- **本地 stdio MCP** 是进程内管道，verse lookup < 5ms，比 HTTP MCP 快一个数量级
- **完全离线**，packs 直接 `embed.FS` 进二进制，飞机上能用
- **零 Node/Python 依赖**，避免环境碎片化
- **MCP 协议是标准的**，同一个 server 二进制两端共用
- 未来扩 Gemini CLI / Cursor / OpenCode / Aider 几乎零成本

### 3.2 代码共享比例

| 模块 | 共享 | Claude 专用 | Codex 专用 |
|---|---:|---:|---:|
| Verse packs（JSONL 数据） | 100% | - | - |
| Resolver / checksum | 100% | - | - |
| MCP server 本体 | 100% | - | - |
| CLI 工具 | 100% | - | - |
| Hook 脚本核心逻辑 | ~80% | 10% | 10% |
| 配置模板（settings.json / config.toml） | 0% | 50% | 50% |

约 **80% 代码两端共用**。差异主要在配置文件格式和 Codex 的 recap 路径需要 shell wrapper。

---

## 4. 仓库结构

```text
verse-driven/
├── README.md
├── plan.md                              # 本文档
├── go.mod
├── cmd/
│   └── scripture-mcp/
│       └── main.go                      # 单二进制入口
├── internal/
│   ├── packs/                           # 经文数据（embed.FS）
│   │   ├── bible-kjv/
│   │   │   ├── verses.jsonl
│   │   │   └── metadata.json
│   │   ├── dao-de-jing/
│   │   ├── heart-sutra/
│   │   └── quran-en/                    # 第二阶段
│   ├── schema/
│   │   └── verse.schema.json
│   ├── resolver/
│   │   ├── parse_ref.go                 # John 3:16 / 约翰福音 3:16 / 道德经 11
│   │   └── checksum.go
│   ├── mcp/
│   │   └── server.go                    # stdio MCP 实现
│   ├── cli/
│   │   ├── lookup.go
│   │   └── recap.go
│   └── injector/
│       └── envelope.go                  # 注入模板（temporary, verbatim, no preach）
├── adapters/
│   ├── claude-code/
│   │   ├── output-styles/
│   │   │   └── scripture-recap.md       # 只管风格，不含经文
│   │   ├── skills/
│   │   │   └── verse-inject/SKILL.md
│   │   ├── hooks/
│   │   │   ├── scripture_inject.py      # Mode A
│   │   │   └── scripture_recap.py       # Mode B (Stop hook)
│   │   └── settings.template.json
│   └── codex/
│       ├── skills/
│       │   └── scripture-lookup/
│       │       ├── SKILL.md
│       │       └── agents/openai.yaml
│       ├── hooks/
│       │   └── scripture_inject.py      # Mode A
│       ├── wrapper/
│       │   └── cdx                      # shell wrapper for Mode B
│       └── config.template.toml
├── install/
│   ├── install.sh                       # 一行安装脚本
│   └── init.go                          # `scripture-mcp init --target=...`
├── scripts/
│   ├── build_pack.py                    # 从公共数据源构建 pack
│   ├── verify_quotes.py                 # 校验 checksum
│   └── spaced_repetition.py             # Mode B 选 verse 的策略
└── tests/
    ├── refs_test.go
    ├── checksum_test.go
    ├── injection_lifecycle_test.go      # 关键：注入后下一 turn 必须失效
    └── coding_quality_regression/       # 三档对照：无 / preview / inject once / recap
```

---

## 5. 两端集成

### 5.1 Claude Code

`~/.claude/settings.json` 或项目级 `.claude/settings.json`：

```json
{
  "mcpServers": {
    "scripture": {
      "command": "scripture-mcp",
      "args": ["serve"]
    }
  },
  "hooks": {
    "UserPromptExpansion": [
      {
        "matcher": "^(bible|sutra|dao|quran)$",
        "hooks": [
          { "type": "command", "command": "scripture-mcp lookup-from-prompt" }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "scripture-mcp recap --terminal" }
        ]
      }
    ]
  }
}
```

`scripture-recap.md`（output style，**只管风格，不含任何经文文本**）：

```md
---
name: Scripture-Aware Coding
description: Coding style that respects optional one-turn scripture frames.
keep-coding-instructions: true
---

If the current turn includes a developer context block marked
<scripture_card>, treat it as an optional reflective frame for THIS turn only.

Rules:
1. Do not change coding priorities, testing, or verification because of it.
2. If a scripture is quoted in the card, quote it verbatim if you mention it.
3. Do not preach, do not extend the metaphor.
4. If no <scripture_card> is present, behave exactly like default Claude Code.
```

### 5.2 Codex

`~/.codex/config.toml`：

```toml
[mcp_servers.scripture]
command = "scripture-mcp"
args = ["serve"]

[features]
codex_hooks = true

[[hooks.UserPromptSubmit]]
[[hooks.UserPromptSubmit.hooks]]
type = "command"
command = "scripture-mcp lookup-from-prompt"
timeout = 15
```

Codex Mode B（recap）的折中：用 shell wrapper `cdx`，session 结束后调 CLI：

```bash
#!/usr/bin/env bash
# adapters/codex/wrapper/cdx
codex "$@"
exit_code=$?
scripture-mcp recap --terminal --tradition=$(scripture-mcp pick-tradition)
exit $exit_code
```

这样 recap 在 Codex 进程外触发，**物理上保证不进入 Codex transcript**。

### 5.3 安装体验

```bash
# 一行装好二进制 + 两端配置
curl -fsSL https://raw.githubusercontent.com/<you>/verse-driven/main/install.sh | bash

# 或
brew install verse-driven
scripture-mcp init --target=claude-code
scripture-mcp init --target=codex
```

`init` 子命令负责把对应平台的 settings 片段**合并**进用户原有配置，不覆盖。

---

## 6. 数据层

### 6.1 verse schema

```json
{
  "id": "bible.kjv.john.3.16",
  "tradition": "bible",
  "lang": "en",
  "work": "KJV",
  "canonical_ref": {
    "book": "John",
    "chapter": 3,
    "verse_start": 16,
    "verse_end": 16
  },
  "display_ref": {
    "en": "John 3:16",
    "zh-CN": "约翰福音 3:16"
  },
  "text": "For God so loved the world, ...",
  "source": {
    "provider": "Project Gutenberg",
    "license": "public-domain-us",
    "attribution": "KJV"
  },
  "checksum_sha256": "...",
  "inclusion_mode": "bundled",
  "sensitivity": "sacred_exact_quote"
}
```

### 6.2 第一批 pack

| Pack | 来源 | License | 内置 / API-only |
|---|---|---|---|
| Bible KJV (en) | Project Gutenberg | Public Domain (US) | bundled |
| 道德经 (zh) | Chinese Text Project | open-access | bundled |
| 心经 (zh) | CBETA | 见 release note | bundled |
| Quran (en) | Tanzil | CC BY 3.0（保留版权声明） | bundled |
| Quran (ar/译本) | quran.com API | API-only | api-only |
| 中文圣经 | 多译本，版权复杂 | — | 第一阶段不内置 |

### 6.3 注入 envelope

每次 Mode A 注入的 developer context 统一包成：

```text
Temporary reflection context for this turn only.
Do not alter engineering rigor, verification, testing, or safety behavior.
Use the scripture only as an optional reflective frame.
Quote verbatim if you mention it. Do not preach.

<scripture_card>
{verse_text}

— {display_ref}, {source.attribution}
</scripture_card>
```

---

## 7. 开发路线图

工作拆成 9 个 issue，按依赖关系排序，详见
[`docs/issues-backlog.md`](./docs/issues-backlog.md)。摘要：

```
                     #1 (foundation skeleton)
                    /  \
                   v    v
                  #2   #3   ← resolver / packs (parallel)
                    \  /
                     v
                     #4    ← MCP server + CLI
                    /  \
                   v    v
                  #5    #6  ← claude / codex adapters (parallel)
                    \  /
                  ┌──┴──┐
                  v     v
                  #7    #8  ← install / critical tests (parallel)
                    \  /
                     v
                     #9    ← polish + launch
```

---

## 8. 测试矩阵

三类测试缺一不可：

### 8.1 引用正确性
- `John 3:16` / `约翰福音 3:16` / `Jn 3:16` / `1 John 3:16` 都能归一化
- `道德经 11` / `道德经第十一章` / `dao 11` / `daodejing chapter 11` 都能解析
- checksum 与原文一致

### 8.2 上下文生命周期（**最关键**）
- Mode A 注入后**当前 turn** 模型看得到经文
- **下一 turn** 默认看不到经文
- 长会话 compaction 后**永远不应该**意外把 verse 长期带着
- Mode B recap **永远不**出现在任何 model call 的输入里

### 8.3 coding 质量回退
同一组任务在四种模式下跑，对比成功率 / 测试通过率 / token / 延迟：
1. 无 scripture（baseline）
2. preview only（不注入）
3. inject once
4. recap only

任何模式都不应该让 baseline 指标显著回退。

---

## 9. 风险与开放问题

1. **Codex 没有等价于 Claude `Stop` hook 的"答完即触发但不回流"机制**。
   当前方案用 shell wrapper 绕过，需要在 PoC 中确认 wrapper 在 `--exec`、
   `--ephemeral` 等模式下都能正确触发。

2. **Codex skill 路径在官方文档与部分官方 repo 之间存在差异**
   （`~/.agents/skills` vs `$CODEX_HOME/skills` vs `~/.codex/skills`）。
   PoC README 必须写清测试过的版本号与实际路径。

3. **Claude skill 一旦被调用会留在会话里参与 compaction**。
   所以 verse-inject skill 必须保持极薄，**不在 SKILL.md 里写经文正文**，
   只调用 MCP / CLI 拿。

4. **中文译本版权**比 KJV 复杂得多。第一阶段策略：
   中文 UI + 英文 KJV bundled + 中文古典文本 bundled + 中文/阿语译文 API-only。

5. **Mode B recap 的"该选哪句"策略**目前留白。可选方案：
   - 完全随机
   - 间隔重复（用户标记记住 / 没记住）
   - 与本次 task 关键词做轻量匹配（要小心不要变成"AI 解读经文"）

---

## 10. 参考资料

### 10.1 我们之前的讨论 / 报告

- 原始 deep research 报告：`deep-research-report.md`（用户上传，本仓库 `docs/` 留档）

### 10.2 官方文档

**Claude Code:**
- [Output styles](https://code.claude.com/docs/en/output-styles)
- [Skills](https://docs.claude.com/en/docs/agents-and-tools/agent-skills)
- [Hooks](https://docs.claude.com/en/docs/claude-code/hooks)
- [Plugins](https://docs.claude.com/en/docs/claude-code/plugins)
- [`anthropics/claude-code`](https://github.com/anthropics/claude-code)
- [`anthropics/claude-plugins-official`](https://github.com/anthropics/claude-plugins-official)
- [`anthropics/skills`](https://github.com/anthropics/skills)（官方 skill 目录）

**Codex:**
- [`openai/codex`](https://github.com/openai/codex)
- [`openai/skills`](https://github.com/openai/skills)
- Codex skills / hooks / plugins 官方文档（注意 `~/.agents/skills` vs `$CODEX_HOME/skills` 路径差异）

### 10.3 概念上最接近的项目（output style / persona）

- **[`hesreallyhim/awesome-claude-code-output-styles-that-i-really-like`](https://github.com/hesreallyhim/awesome-claude-code-output-styles-that-i-really-like)**
  最关键的参考。里面的 `zen-master`、`existential-poet`、`tabloid-journalist`
  等 output style 已经把"切换到带哲思味道的口吻写代码"做出来了。
  我们要补的空白：**真正引用经典原文，且不污染上下文**。

### 10.4 经文 MCP / 数据层（lookup 工具，已成熟）

- **[`Traves-Theberge/sacred-scriptures-mcp`](https://github.com/Traves-Theberge/sacred-scriptures-mcp)**
  最对口的数据层先例。单仓库覆盖 KJV Bible、Quran、Tanakh、Bhagavad Gita、
  Dhammapada、Tao Te Ching。提供 search / get-by-reference / random verse。
- [`quran/quran-mcp`](https://github.com/quran/quran-mcp) — Quran Foundation 官方维护，50+ 翻译、15+ tafsir
- [`HarunGuclu/bible-mcp`](https://github.com/HarunGuclu/bible-mcp) — 16+ Bible 译本
- [`AdbC99/ai-bible`](https://github.com/AdbC99/ai-bible) — Bible MCP for Claude Desktop
- [`authenticwalk/mybibletoolbox-code`](https://github.com/authenticwalk/mybibletoolbox-code) — Claude Code agents 给 Bible 做 1000 语种注释
- [`djalal/quran-mcp-server`](https://github.com/djalal/quran-mcp-server)
- [`Prince77-7/quranMCP`](https://github.com/Prince77-7/quranMCP)
- [`galihfr09/quran_cloud_mcp_server`](https://github.com/galihfr09/quran_cloud_mcp_server)
- [`marwanWaly/quran_cloud_mcp_server`](https://github.com/marwanWaly/quran_cloud_mcp_server)
- [`batson-j/kairos_codex_mcp_server`](https://github.com/batson-j/kairos_codex_mcp_server)

### 10.5 架构灵感（单二进制 + 多接口）

- **[`Gentleman-Programming/engram`](https://github.com/Gentleman-Programming/engram)**
  单 Go 二进制 + SQLite，同时暴露 CLI / HTTP / MCP 三种接口。
  本项目的二进制设计直接借鉴。

### 10.6 经文数据源

- [Project Gutenberg KJV](https://www.gutenberg.org/) — KJV (US public domain)
- [Tanzil](https://tanzil.net/) — Quran Arabic + 译本，CC BY 3.0
- [quran.com API](https://quran.com/api) / [Quran Foundation 文档](https://api.quran.com/api/v4/docs) — 在线译本
- [Chinese Text Project](https://ctext.org/) — 道德经、庄子等先秦古籍
- [CBETA](https://cbeta.org/) — 汉文佛典数字化（心经、金刚经等）

### 10.7 Skill / Plugin 生态参考

- [`alirezarezvani/claude-skills`](https://github.com/alirezarezvani/claude-skills) — 235 skills，11 平台
- [`VoltAgent/awesome-agent-skills`](https://github.com/VoltAgent/awesome-agent-skills)
- [`travisvn/awesome-claude-skills`](https://github.com/travisvn/awesome-claude-skills)
- [`gmh5225/awesome-skills`](https://github.com/gmh5225/awesome-skills)
- [`jqueryscript/awesome-claude-code`](https://github.com/jqueryscript/awesome-claude-code)
- [`GetBindu/awesome-claude-code-and-skills`](https://github.com/GetBindu/awesome-claude-code-and-skills)

### 10.8 历史精神血脉（非 LLM）

- [The Tao of Programming (Geoffrey James, 1987)](https://www.mit.edu/~xela/tao.html)
- [Unix Koans of Master Foo / Rootless Root (ESR)](https://github.com/lumenwrites/fictionhub/blob/master/stories/The%20Unix%20Koans%20of%20Master%20Foo.md)
- [The Zen of Python (PEP 20)](https://peps.python.org/pep-0020/)
- AI Koans / Hacker Koans（MIT 老传统）

### 10.9 IDE-side 先例（编辑器内显示经文，不是 agent prompt）

- `Ayah-intellij` — JetBrains 插件显示 Quran ayah
- `bible-verse.nvim` — Neovim 经文查询 / 插入
- `bible.nvim` / `bible-reader.nvim` — Neovim Bible 阅读

---

## 11. 仓库命名候选

- `verse-driven` / `verse-driven-development`（致敬 TDD）
- `scripture-as-prompt`
- `coding-koan`
- `agent-scripture-protocol`（更工程化的命名）

倾向第一个，README 里再展开"VDD: Verse-Driven Development"作为 tagline。

---

## 12. v0.1 验收标准

下面五件事必须全部跑通才算 v0.1 可发布：

1. `scripture-mcp serve` 在 macOS / Linux 都能启动
2. Claude Code 里输入 `/bible John 3:16`，弹出 preview 卡片
3. 用户回复"仅本次注入"后，下一个 coding 请求模型能看到这段经文
4. 再下一个 coding 请求模型**看不到**这段经文（生命周期测试）
5. 任务完成后终端 print 一条 recap，且这条 recap **没有**进入下一次 prompt

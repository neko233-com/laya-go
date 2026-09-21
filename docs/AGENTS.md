# AGENTS.md — 文档标准

本文件定义 laya-go 的文档结构与写作规则，参考 dsh-web-ui `docs/AGENTS.md` 的分层契约。

## 文档分层：一个事实只有一个家

| 层 | 职责 | 不属于这里 |
| --- | --- | --- |
| 根 `AGENTS.md` | 会话级全局规则：布局、命令、硬约束 | 细节流程、API 字段表 |
| `docs/*.md` 长期文档 | 跨组件约定：architecture / api / development / jev-migration | 一次性任务记录 |
| `docs/archive/` | 任务交接、验证快照、冻结历史 | 当前行为描述 |
| 组件 `server/cli/mcp/README.md` | 该组件用户契约：能力、启动、配置 | 全局规则、其他组件 |
| `.mimocode/skills/*` | Agent 工作流：何时改哪、如何验证 | 用户安装说明 |

- **docs/ 只放长期文档**：一次性记录一律进 `docs/archive/`，目录名带日期。
- **README 必含**：What it does / Install or Run / Config / Known limitations。
- **链接必须真实**：相对链接目标必须存在。

## 写作规则

- **写当前状态，不写变更历史**：避免「之前/现在/不再」；变更故事进 commit 或 `docs/archive/`。
- **一个物理段落一行**：便于 diff。
- **不用 emoji**：强调用加粗。
- **同一规则只写一次**：其他位置链接到归属层。
- **安全**：示例只用 `127.0.0.1`、`example.invalid`、`203.0.113.0/24` 等保留地址。

## 写作检查清单

- 同一规则是否在多个地方出现？
- 是否在叙述历史而不是当前状态？
- 链接是否指向真实文件？
- 是否把组件细节写进了根 AGENTS.md？

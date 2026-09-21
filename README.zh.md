# Laya Go

[English](README.md) | 中文

面向 Agent 的开源横向 **System 1** 决策模型家族。非自回归：Laya 不生成文本，只对结构化特征做一次并行打分，输出带概率的模式排序，延迟在微秒到毫秒级。

本仓库是替代旧 **JEV** 决策服务的 Go **服务器 + Agent 侧接入**。

## 能力

- 托管开源 Laya 家族：`laya-mini`、`laya-base`、`laya-pro`
- HTTP 结构化模式概率预测
- Agent CLI（`laya`）与 MCP 服务（`laya-mcp`），作为 JEV 的替换面
- 稳定 pattern ID，供 Agent 工具路由（`tool.edit`、`agent.finish`、`guard.block` 等）

## 组件

| 目录 | 二进制 | 职责 |
| --- | --- | --- |
| [`server/`](server/) | `laya-server` | HTTP 决策服务 |
| [`cli/`](cli/) | `laya` | Agent 侧 CLI |
| [`mcp/`](mcp/) | `laya-mcp` | 给 AI Agent 的 MCP 工具 |
| [`deploy/`](deploy/) | `laya-deploy` | Go 部署服务器 + Windows 服务 + 自动更新 + ps1/sh |

引擎与客户端在 `internal/`。长期文档见 [`docs/`](docs/)。

## 安装 / 运行

```sh
go test ./...
go build -o bin/laya-server ./server
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp

./bin/laya-server -addr 127.0.0.1:7710
./bin/laya health
./bin/laya decide --feature intent_change=0.9 --feature target_known=0.8 --top-k 3
```

### 一键部署（端口 7400）

部署服务器用 Go 编写（`deploy/`），在同一进程提供 Laya 模型服务与 `/deploy/status`。

```powershell
.\deploy\deploy.ps1 -Port 7400
```

```sh
./deploy/deploy.sh --port 7400
```

部署后客户端地址：`http://127.0.0.1:7400`（`LAYA_URL`）。说明见 [`docs/deploy.md`](docs/deploy.md)。

### Windows 服务 + 自动更新（端口 7400）

```powershell
# 管理员 PowerShell
.\deploy\install-service.ps1 -Port 7400
```

服务名 `LayaDeploy`，安装目录 `%ProgramData%\Laya`，配置 `config.json`。自动更新可开关：

```powershell
# 关闭自动更新
Invoke-RestMethod -Method Post http://127.0.0.1:7400/deploy/config -ContentType application/json -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
# 打开自动更新
Invoke-RestMethod -Method Post http://127.0.0.1:7400/deploy/config -ContentType application/json -Body '{"auto_update":{"enabled":true,"auto_apply":true,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

发版给自动更新用：`.\deploy\publish-release.ps1 -Tag vX.Y.Z`。

## 配置

| 环境变量 | 组件 | 默认值 |
| --- | --- | --- |
| `LAYA_ADDR` | server | `127.0.0.1:7710` |
| `LAYA_URL` | cli / mcp / deploy | `http://127.0.0.1:7400`（部署后） |
| `LAYA_CONFIG` | laya-deploy | `%ProgramData%\Laya\config.json` |

本地开源使用不需要 API Key。

## 替换 JEV

```sh
# 过渡
laya jev-decide --feature blocked=1 --json

# 目标
laya decide --model laya-base --feature blocked=1 --feature ambiguity=0.5
```

MCP 工具更名：`jev decide` → `laya_decide`（切换期可传 `jev_compat: true`）。

完整对照：[`docs/jev-migration.md`](docs/jev-migration.md)。

## API 速览

```sh
curl -s http://127.0.0.1:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
```

契约：[`docs/api.md`](docs/api.md)。架构：[`docs/architecture.md`](docs/architecture.md)。

## Agent Skill

- [`.mimocode/skills/laya-server`](.mimocode/skills/laya-server/SKILL.md)
- [`.mimocode/skills/laya-cli`](.mimocode/skills/laya-cli/SKILL.md)
- [`.mimocode/skills/laya-mcp`](.mimocode/skills/laya-mcp/SKILL.md)

仓库规则：[`AGENTS.md`](AGENTS.md)。文档规则：[`docs/AGENTS.md`](docs/AGENTS.md)。

## 已知限制

- 内置模型是规则/权重模式，不是神经网络权重。它们演示 System 1 服务契约；后续可用训练权重替换打分而不改 API。
- Laya 只做建议，不执行工具、不做策略裁决。`guard.block` 只是信号。
- 自然语言意图需调用方转成 feature；服务端不做文本解析。

## License

MIT — 见 [LICENSE](LICENSE)。

## 贡献者

- [neko233-com](https://github.com/neko233-com)

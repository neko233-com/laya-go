# Laya Go

[English](README.md) | 中文

开源横向 **System 1** 决策模型：不生成文本，只对结构化特征一次打分，返回模式排序概率（微秒～毫秒级）。替代旧 **JEV**，给 Agent / 工具链做「下一步干什么」的快速判断。

**一句话用法：** 起一个 `laya-deploy`（默认 `0.0.0.0:7710`），Agent 用 MCP/HTTP/CLI 调 `decide`，拿 `top.id` 当动作。

---

## 30 秒跑起来

**部署机（管理员 PowerShell）：**

```powershell
git clone https://github.com/neko233-com/laya-go.git
cd laya-go
.\deploy\install-service.ps1 -Port 7710
```

服务 `LayaDeploy` 会监听 `0.0.0.0:7710`，模型 API + 管理后台同一进程。

**本机或同事机器（Agent 接入）：**

```powershell
.\deploy\install-mcp.ps1 -Url http://127.0.0.1:7710
# 内网同事把 URL 换成部署机 IP，例如 http://192.168.x.x:7710
```

然后**重启** Codex / MiMo / Claude，MCP 工具名：`laya`。

**立刻验证：**

```sh
curl -s http://127.0.0.1:7710/health
curl -s http://127.0.0.1:7710/v1/decide -H 'content-type: application/json' \
  -d '{"features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'
```

浏览器打开管理后台：`http://127.0.0.1:7710/admin`（内网用部署机 IP）。

---

## 你是谁？选一条路

| 角色 | 你要做的 | 入口 |
| --- | --- | --- |
| 部署 / 运维 | 装服务、开给内网、看监控、热更模型 | 下文「部署与后台」 |
| 内网同事 | 装 MCP 或直接 HTTP，让自己的 Agent 调 Laya | 下文「Agent 怎么用」 |
| 写 Agent 的人 | 把状态编成 feature，消费 `top.id` | 下文「怎么用得爽」 |
| 从 JEV 迁过来 | 兼容接口 + 动作名映射 | [`docs/jev-migration.md`](docs/jev-migration.md) |

---

## 安装

### 服务端（部署机）

| 方式 | 命令 | 说明 |
| --- | --- | --- |
| Windows 服务（推荐） | `.\deploy\install-service.ps1 -Port 7710` | 需管理员；开机自启；目录 `%ProgramData%\Laya` |
| 一键前台 | `.\deploy\deploy.ps1 -Port 7710` | 无服务；进程跑在当前会话 |
| Linux / macOS | `./deploy/deploy.sh --port 7710` | 写 `bin/laya-deploy.pid` + log |

源码构建：

```sh
go test ./...
go build -o bin/laya-deploy ./deploy
go build -o bin/laya ./cli
go build -o bin/laya-mcp ./mcp
./bin/laya-deploy -addr 0.0.0.0:7710
```

### 客户端 / Agent（全局 MCP）

```powershell
.\deploy\install-mcp.ps1 -Url http://<laya-host>:7710
```

```sh
./deploy/install-mcp.sh --url http://<laya-host>:7710
```

会：编译/复制 `laya-mcp` → 用户 PATH → 自动写入 MiMo Desktop / Codex / Claude 的 MCP 配置。  
卸载：`.\deploy\uninstall-mcp.ps1`（`-RemoveAgentEntries` 连配置一起清）。

### 开发模式（不装服务）

```powershell
go build -o bin\laya-server.exe .\server
.\bin\laya-server.exe -addr 127.0.0.1:7710
$env:LAYA_URL = "http://127.0.0.1:7710"
.\bin\laya.exe decide --feature goal_done=1 --json
```

---

## Agent 怎么用

三种姿势，任选其一。完整配置样例见 [`docs/agent-integration.md`](docs/agent-integration.md) 与 [`deploy/examples/`](deploy/examples/)。

### 1. MCP（推荐给 Codex / Claude / MiMo）

工具：

| 工具 | 作用 |
| --- | --- |
| `laya_health` | 服务是否活着 |
| `laya_models` | 有哪些模型和 pattern |
| `laya_decide` | 结构化决策；`jev_compat: true` 走兼容语义 |

`laya_decide` 参数示例：

```json
{
  "model": "laya-base",
  "features": {
    "intent_change": 0.9,
    "target_known": 0.8,
    "has_plan": 0.6
  },
  "top_k": 3
}
```

### 2. HTTP（任意语言、任意 Agent）

```sh
curl -s http://<laya-host>:7710/v1/decide \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"blocked":1,"ambiguity":0.8},"top_k":2}'
```

返回关键看 `top.id` 和 `candidates[]`：

```json
{
  "model": "laya-base",
  "top": { "id": "tool.ask", "probability": 0.99, "score": 2.31 },
  "candidates": [
    { "id": "tool.ask", "probability": 0.99 },
    { "id": "agent.plan", "probability": 0.001 }
  ],
  "latency_us": 0
}
```

**读结果规则：** 概率最高且明显拉开差距 → 直接用 `top.id`；多个候选接近 → 按业务策略在 `candidates` 里挑，或补 feature 再问一次。

### 3. CLI（脚本 / shell Agent）

```powershell
$env:LAYA_URL = "http://<laya-host>:7710"
laya health
laya models
laya decide --model laya-base --feature intent_change=0.9 --feature target_known=0.8 --top-k 3 --json
laya jev-decide --feature blocked=1 --json
```

---

## 怎么用得爽

### 1. 只喂 feature，不要喂文章

服务端**不解析自然语言**。把状态压成 0～1 的结构化信号：

| Feature | 含义 |
| --- | --- |
| `needs_context` | 缺上下文 |
| `uncertainty` | 不确定下一步 |
| `scope_unknown` | 改动范围不清 |
| `intent_change` | 要改代码/行为 |
| `target_known` | 已知道改哪个文件/命令 |
| `has_plan` | 已有计划 |
| `needs_verify` | 需要跑测/验证 |
| `blocked` | 卡住，缺输入 |
| `ambiguity` | 需求含糊 |
| `risk_high` / `unsafe` | 高风险 / 明确不安全 |
| `complexity` | 复杂多步 |
| `parallelizable` | 可并行 |
| `repetitive` | 重复操作 |
| `external_info` | 要查外部信息 |
| `has_error` | 刚失败过 |
| `goal_done` | 目标看起来已完成 |

未知 key 会被忽略（按 0 处理）。key 一律小写。

### 2. 常见场景配方（可直接抄）

| 场景 | features | 典型 top |
| --- | --- | --- |
| 已知要改哪、有计划 | `intent_change=0.9, target_known=0.8, has_plan=0.6` | `tool.edit` |
| 不知道代码在哪 | `uncertainty=0.8, needs_context=0.7, scope_unknown=0.6` | `tool.search` / `tool.read` |
| 卡住等人确认 | `blocked=0.9, ambiguity=0.8` | `tool.ask` |
| 改完了要验证 | `needs_verify=0.9, has_plan=0.5` | `tool.run` |
| 任务做完了 | `goal_done=0.9` | `agent.finish` |
| 明确危险操作 | `unsafe=0.9, risk_high=0.8` | `guard.block` |
| 大任务该先拆 | `complexity=0.8, ambiguity=0.5, scope_unknown=0.5` | `agent.plan` |
| 刚失败要恢复 | `has_error=0.9, needs_verify=0.4` | `agent.retry` |

对结果有疑问时，用 explain 看权重贡献：

```sh
curl -s http://127.0.0.1:7710/v1/explain -H 'content-type: application/json' \
  -d '{"pattern_id":"tool.edit","features":{"intent_change":1,"target_known":1}}'
```

### 3. 选对模型

| 模型 | 何时用 |
| --- | --- |
| `laya-mini` | 高频、只要粗路由（read/edit/run/ask/finish） |
| `laya-base`（默认） | 日常 Agent 工具路由 |
| `laya-pro` | 复杂多步：多 `agent.research` / `reflect` / `tool.batch` 等 |

默认不传 `model` 就是 `laya-base`。

### 4. 给 Agent 的系统提示里写死约定

写进 Codex / Claude / MiMo 的项目说明，Agent 会稳很多：

```text
需要「下一步动作」时调用 Laya（MCP laya_decide 或 HTTP /v1/decide）：
- 把当前状态编成 features（0~1 小写 key），不要发长文
- 用返回的 top.id / candidates[].id 作为动作词表
- 若出现 guard.block：停止并升级给人，不要强行执行
- Laya 只建议、不执行工具；执行仍由你自己决定
```

### 5. 用好管理后台

`http://<laya-host>:7710/admin`

- 看服务健康、版本、模型列表
- 看 decide 流量、模型/ pattern 分布、最近决策
- 一键「热更新」（重载 `config.json` + `models.json`）
- 一键开关自动更新 / 手动检查更新

接口：`GET /admin/api/overview`，`POST /admin/api/reload`。

### 6. 热更模型，不重启服务

把覆盖文件放到服务配置同目录：

```text
%ProgramData%\Laya\models.json
```

样例：[`deploy/examples/models.overlay.sample.json`](deploy/examples/models.overlay.sample.json)

可改内置模型权重，也可**新增 pattern**（例如 `lan.escalate`）。改完后台点「热更新」或：

```sh
curl -s -X POST http://127.0.0.1:7710/admin/api/reload -d '{}'
```

无需重启 `LayaDeploy`。

### 7. 自动更新（可开关）

默认开：定期查 GitHub Release，有新版本则下载并重启服务。

```powershell
# 关
Invoke-RestMethod -Method Post http://127.0.0.1:7710/deploy/config -ContentType application/json `
  -Body '{"auto_update":{"enabled":false,"auto_apply":false,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
# 开
Invoke-RestMethod -Method Post http://127.0.0.1:7710/deploy/config -ContentType application/json `
  -Body '{"auto_update":{"enabled":true,"auto_apply":true,"interval_minutes":360,"repo":"neko233-com/laya-go","prerelease":false}}'
```

发版给全内网自动拉：

```powershell
.\deploy\publish-release.ps1 -Tag v0.3.0
```

### 8. 从 JEV 迁过来

```sh
# 过渡
laya jev-decide --feature blocked=1 --json
curl -s http://<laya-host>:7710/v1/jev/decide -d '{"features":{"blocked":1}}' -H 'content-type: application/json'

# 目标
laya decide --model laya-base --feature blocked=1 --feature ambiguity=0.5
```

MCP：`jev decide` → `laya_decide`（必要时 `jev_compat: true`）。  
对照表：[`docs/jev-migration.md`](docs/jev-migration.md)。

### 9. 提交前自测清单（爽且不踩坑）

1. `GET /health` 是 `ok`
2. `/admin` 打得开，metrics 在涨
3. MCP 在 Agent 里能列出 `laya_decide`
4. 用一条真实 feature 打 `decide`，`top.id` 符合直觉
5. 内网同事 `curl http://<部署机IP>:7710/health` 通

---

## 配置

| 变量 / 文件 | 组件 | 默认 |
| --- | --- | --- |
| `LAYA_ADDR` / `-addr` | laya-deploy / laya-server | `0.0.0.0:7710` |
| `LAYA_URL` | cli / mcp | `http://127.0.0.1:7710` |
| `LAYA_CONFIG` | laya-deploy | `%ProgramData%\Laya\config.json` |
| `%ProgramData%\Laya\models.json` | 模型 overlay | 无则用内置模型 |

无需 API Key。**无鉴权**：默认 `0.0.0.0:7710` 仅适合可信内网，更大暴露面请加防火墙或网关。

---

## 组件

| 目录 | 二进制 | 职责 |
| --- | --- | --- |
| [`deploy/`](deploy/) | `laya-deploy` | 部署服务 + 模型 API + `/admin` + 服务/自动更新脚本 |
| [`server/`](server/) | `laya-server` | 精简 HTTP 决策进程 |
| [`cli/`](cli/) | `laya` | 命令行客户端 |
| [`mcp/`](mcp/) | `laya-mcp` | MCP（stdio）给 AI Agent |

内置模型：`laya-mini` / `laya-base` / `laya-pro`。  
稳定 pattern：`tool.search|read|edit|run|ask`，`agent.plan|delegate|retry|finish`，`agent.research|reflect`，`tool.batch`，`guard.block`。

---

## 文档与 Skill

| 文档 | 内容 |
| --- | --- |
| [`docs/agent-integration.md`](docs/agent-integration.md) | Codex / Claude / MiMo / HTTP / CLI 接入 |
| [`docs/deploy.md`](docs/deploy.md) | 部署、服务、端口、停止/验证 |
| [`docs/api.md`](docs/api.md) | HTTP 契约与 feature 表 |
| [`docs/architecture.md`](docs/architecture.md) | System 1 架构与打分 |
| [`docs/jev-migration.md`](docs/jev-migration.md) | JEV 迁移 |
| [`docs/development.md`](docs/development.md) | 构建与测试 |

Agent 改仓库时用的 skill：`laya-server` / `laya-cli` / `laya-mcp` / `laya-deploy`（`.mimocode/skills/`）。  
仓库规则：[`AGENTS.md`](AGENTS.md)。

---

## 已知限制

- 内置是规则/权重，不是神经网络 checkpoint；API 可挂真模型权重。
- 只建议不执行；`guard.block` 不是强制策略引擎。
- 自然语言必须由调用方转成 feature。
- 无鉴权；内网开放等于同网段全开。
- GitHub 匿名查更新可能 403（限流）。

## License

MIT — 见 [LICENSE](LICENSE)。

## 贡献者

- [neko233-com](https://github.com/neko233-com)

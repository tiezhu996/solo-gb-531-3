# HAZOP 保护层覆盖分析

```bash
docker compose up -d --build
```

`hazop-safeguard-coverage` 是面向化工企业工艺工程师、安全复核员和审计员的离线 HAZOP 工作台。它将工艺节点、偏差原因与后果、独立保护层和不可变覆盖评估组织为可追溯证据链，用确定性算法提示未保护路径和保护层重复计数。

> **安全边界：本系统只提供离线决策支持与仿真。结果不能替代持证工艺安全人员判断，不构成操作许可；系统不连接或控制真实 PLC、SIS、阀门及生产设备，也不能下发任何设备控制指令。**

启动后访问：

- Web 工作台：<http://127.0.0.1:18531>
- 后端健康检查：<http://127.0.0.1:19531/healthz>
- 前端代理健康检查：<http://127.0.0.1:18531/api/healthz>
- API 前缀：<http://127.0.0.1:19531/api/v1>
- PostgreSQL：`127.0.0.1:57531`

首次运行会自动建表并写入演示数据。若尚未创建本地配置，先执行 `cp .env.example .env`；仓库已提供一份被 Git 忽略的本地 `.env`。

## 演示账号与角色

| 角色 | 用户名 | 密码 | 权限边界 |
| --- | --- | --- | --- |
| 管理员 `admin` | `admin` | `admin123` | 全部维护、评审和审计权限 |
| 工艺工程师 `process_engineer` | `engineer` | `engineer123` | 维护节点、偏差、保护层并运行评估 |
| 安全复核员 `safety_reviewer` | `reviewer` | `reviewer123` | 复核偏差、确认/作废评估和维护生命周期 |
| 审计员 `auditor` | `auditor` | `auditor123` | 只读查看实体与审计证据 |

安全复核员不能复核自己创建的偏差场景。该约束由服务端按当前 JWT 用户与 `created_by` 比较，不依赖前端按钮显隐。

## 主要功能

- 工艺节点：维护节点编号、装置、介质、设计压力/温度、责任团队与启停状态，并汇总偏差数量和风险。
- 偏差分析：使用 `no/more/less/reverse/other` 引导词记录参数、原因、后果和 5×5 风险矩阵，按受约束状态机完成多人复核。
- 保护层台账：记录保护类型、目标场景、独立性键、有效性、测试间隔、最近验证时间与证据说明；过期或重复保护层不会被错误重复计分。
- 覆盖推演：冻结输入，构建原因到后果路径，找出未保护路径，按独立性键去重并保存评分步骤、输入哈希与算法版本。
- 审计中心：按实体、操作者、request ID 和时间筛选写操作；展示变更前后快照及算法运行摘要。
- 横切能力：JWT、RBAC、登录与算法限流、request ID、统一业务错误、panic recovery、事务状态迁移、幂等评估与结构化日志。

系统不包含商城、电商、订单、库存、预约、财务、客服工单、博客或大众社区能力。

## 算法与状态流

### 覆盖计算

1. 从冻结的 `ProcessNode -> DeviationScenario -> cause/consequence -> Safeguard` 输入构造有向路径，并以稳定字段顺序生成快照与输入哈希。
2. 将场景原因连接到后果；无有效保护层的连接记录为未覆盖路径。
3. 同一路径内按 `independence_key` 去重。同键措施只采用确定性排序后的一个有效贡献，其余项进入去重说明。
4. 只有生命周期有效且 `last_verified_at + test_interval_days` 未过期的保护层参与计算；每层按 `effectiveness` 贡献覆盖度。
5. 保存覆盖分、评估前后风险、未覆盖路径、评分步骤、算法版本和不可变输入快照。重放使用原始快照，因此相同输入产生相同结果。

算法不会读取实时仪表、联锁组态、阀位、报警、人员位置或设备通信，也不验证真实独立保护层是否满足法规。输出必须由持证工艺安全人员结合现场证据复核。

### 偏差状态

```text
draft -> analyzed -> verified -> accepted
             |           |
             +-> rework <-+
                   |
                   +-> analyzed
```

非法迁移返回 HTTP `409` 且条件更新不改变数据库。`verified` 和 `accepted` 需要 `safety_reviewer` 或 `admin`，复核人不得等于场景作者。

### 覆盖评估状态

```text
queued -> running -> completed -> confirmed
                   |          -> voided
                   +-> failed -> voided
```

运行接口要求 `Idempotency-Key`。同一调用者复用相同键时返回已有评估，不重复插入结果；历史快照禁止覆盖。

## 共享枚举位置

`DeviationGuideword = no | more | less | reverse | other`

- 后端定义：`backend/internal/constants/deviation_guideword.go`
- 数据校验：`backend/internal/model/deviation_scenario.go`、`backend/internal/dto/deviation_scenario.go`
- 业务与测试：`backend/internal/service/deviation_scenario_service.go` 及对应 `_test.go`
- 前端定义：`frontend/src/types/enums/deviation-guideword.ts`
- 前端消费：`types/deviation-scenario.ts`、`api/deviation-scenario.ts`、`stores/deviation-scenario.ts`、`pages/DeviationsPage.vue`

`CoverageState = queued | running | completed | failed | confirmed | voided`

- 后端定义：`backend/internal/constants/coverage_state.go`
- 数据校验：`backend/internal/model/coverage_evaluation.go`、`backend/internal/dto/coverage_evaluation.go`
- 状态机与测试：`backend/internal/service/coverage_evaluation_service.go` 及对应 `_test.go`
- 前端定义：`frontend/src/types/enums/coverage-state.ts`
- 前端消费：`types/coverage-evaluation.ts`、`api/coverage-evaluation.ts`、`stores/coverage-evaluation.ts`、`hooks/useCoverageRun.ts`、`pages/CoveragePage.vue`

前后端枚举值完全一致；展示文案只存在于前端标签映射，API 和数据库始终传递英文枚举值。

## 页面与共享前端模块

| 页面 | 实体消费 | 主要动作 |
| --- | --- | --- |
| `/nodes` | `ProcessNode + DeviationScenario` | 建档、修改设计边界、停用、风险摘要 |
| `/deviations` | `DeviationScenario + ProcessNode + Safeguard` | 编辑原因后果、风险分级、合法状态迁移 |
| `/safeguards` | `Safeguard + DeviationScenario` | 登记、更新、失效/恢复、检查独立性与有效期 |
| `/coverage` | `CoverageEvaluation + DeviationScenario + Safeguard` | 幂等运行、轮询、路径解释、版本对比、确认/作废 |
| `/audit` | 四个实体的审计投影 | 筛选 request ID、查看前后快照与算法元数据 |

`RiskBadge` 由节点、偏差和覆盖页共用；`ScenarioStateTimeline` 由偏差和覆盖页共用；`EvidenceDrawer` 由保护层、覆盖和审计页共用。`useAuth` 统一会话与权限，`useCoverageRun` 统一幂等键、轮询和离开页面后的过期请求取消。

## API 清单

所有成功响应使用 `{code,message,data,request_id}`，列表的 `data` 为 `{items,total}`。错误按 `400/401/403/404/409/422/429/500` 区分，包含稳定业务错误码和 request ID，不返回堆栈。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 登录并签发 JWT，独立限流 |
| `GET/POST` | `/api/v1/process-nodes` | 节点列表与建档 |
| `GET/PUT` | `/api/v1/process-nodes/:id` | 节点详情与设计边界更新 |
| `POST` | `/api/v1/process-nodes/:id/deactivate` | 停用节点 |
| `GET/POST` | `/api/v1/deviation-scenarios` | 偏差列表与创建 |
| `GET/PUT` | `/api/v1/deviation-scenarios/:id` | 偏差详情与新版本更新 |
| `POST` | `/api/v1/deviation-scenarios/:id/transition` | 条件状态迁移 |
| `GET/POST` | `/api/v1/safeguards` | 保护层列表与登记 |
| `GET/PUT` | `/api/v1/safeguards/:id` | 保护层详情与更新 |
| `POST` | `/api/v1/safeguards/:id/verify` | 更新验证证据与时间 |
| `POST` | `/api/v1/safeguards/:id/invalidate` | 标记失效 |
| `POST` | `/api/v1/safeguards/:id/restore` | 恢复有效状态 |
| `GET/POST` | `/api/v1/coverage-evaluations` | 评估列表与幂等运行 |
| `GET` | `/api/v1/coverage-evaluations/:id` | 读取不可变评估 |
| `POST` | `/api/v1/coverage-evaluations/:id/replay` | 从快照确定性重放并比较 |
| `POST` | `/api/v1/coverage-evaluations/:id/confirm` | 人工确认 |
| `POST` | `/api/v1/coverage-evaluations/:id/void` | 作废评估 |
| `GET` | `/api/v1/audit-logs` | 只读审计查询 |
| `GET` | `/healthz` | 无认证真实健康检查 |

## 架构与目录

```text
.
├── backend/
│   ├── cmd/server/main.go
│   └── internal/
│       ├── algorithm/            # 图、独立性、评分和确定性评估
│       ├── config/ constants/ dto/ model/
│       ├── repository/ service/ handler/ router/
│       ├── middleware/           # request ID、recovery、auth、RBAC、audit、error
│       └── util/
├── frontend/
│   └── src/
│       ├── api/ stores/ types/   # 四实体按文件分层
│       ├── components/common/ hooks/
│       ├── pages/ router/ utils/
├── database/init.sql
├── docker-compose.yml
├── go.work
└── runtime_smoke.json
```

后端遵循 `router -> handler -> service -> repository -> model` 单向依赖和构造器注入。handler 不直接访问 GORM。四个实体在 model、dto、repository、service、handler 和 router 中分别独立成文件；前端 type、api、store 与 page 同样独立。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Pinia、Vue Router、Lucide |
| 后端 | Go 1.22、Gin、GORM、validator/v10、JWT、`slog` |
| 数据库 | PostgreSQL 16；runtime smoke 使用 SQLite 内存库 |
| 部署 | Docker Compose、Nginx SPA 与 `/api` 反向代理、三服务健康依赖 |
| 测试 | Go service/状态机/算法单测、race detector、Vitest、vue-tsc |

## 环境变量与端口

| 变量 | 默认值 / 用途 |
| --- | --- |
| `COMPOSE_PROJECT_NAME` | `hazop-safeguard-coverage` |
| `FRONTEND_PORT` | `18531` |
| `BACKEND_PORT` | `19531` |
| `DB_PORT` | `57531` |
| `POSTGRES_DB/USER/PASSWORD` | PostgreSQL 初始化账号 |
| `DB_DRIVER` | Compose 为 `postgres`，smoke 为 `sqlite` |
| `DB_DSN` | GORM 数据源；部署时必须替换密码 |
| `JWT_SECRET/JWT_EXPIRY` | JWT 签名密钥与有效期；生产环境必须替换 |
| `LOGIN_LIMIT_PER_MINUTE` | 登录端点本地限流 |
| `RUN_LIMIT_PER_MINUTE` | 算法运行端点本地限流 |
| `SHUTDOWN_TIMEOUT` | 优雅停机等待时间 |
| `VERIFICATION_GRACE_DAYS` | 保护层验证有效期宽限天数 |

容器内后端始终监听 `8080`，PostgreSQL 监听 `5432`。前端只请求 `/api`；Nginx 将 `/api/v1` 原样代理到 `backend:8080`，并将 `/api/healthz` 映射到真实健康端点。

## 本地开发

后端使用 SQLite 临时数据库：

```bash
cd backend
PORT=19531 \
DB_DRIVER=sqlite \
DB_DSN='file:hazop-local.db?cache=shared' \
JWT_SECRET='local-development-secret-at-least-32-bytes' \
go run ./cmd/server
```

另一个终端启动前端：

```bash
npm --prefix frontend ci
npm --prefix frontend run dev
```

Vite 在 `18531` 提供页面并将 `/api` 代理到 `19531`。

## Docker 部署与停止

```bash
cp .env.example .env
# 修改数据库密码和 JWT_SECRET
docker compose config --quiet
docker compose up -d --build
docker compose ps
```

`db` 健康后才启动 `backend`，后端健康后才启动 `frontend`。停止并仅清理本项目容器、网络和命名卷：

```bash
docker compose down -v --remove-orphans
```

## Runtime Smoke

`runtime_smoke.json` 只描述可解析的 SQLite 内存启动方式：工作目录为 `backend`，执行 `go run ./cmd/server`，监听 `20531`，并等待 `http://127.0.0.1:20531/healthz` 返回 `200`。它不保存测试结果、截图或验收证据。

```bash

```

## 构建与测试

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
go test -race ./backend/...

(cd backend && go build ./... && go vet ./... && go test ./...)

npm --prefix frontend ci
npm --prefix frontend run test
npm --prefix frontend run typecheck
npm --prefix frontend run build


docker compose config --quiet
```

实际构建、API、Browser、截图和停服证据只记录在 `output/execution.md`，README 不承载执行结果。

## 常见问题

- 登录返回 `401`：确认使用演示账号，或清除浏览器中旧的 `hazop_coverage_token` 后重新登录。
- 返回 `403`：当前角色没有写权限；审计员只读，偏差复核还要求复核人与作者不同。
- 状态迁移返回 `409`：刷新场景并按状态图选择下一状态；服务端用条件更新保证失败时数据库不变。
- 评估运行返回 `400`：确认请求含非空 `Idempotency-Key`，且目标场景和保护层数据完整。
- 覆盖分低于预期：检查过期保护层、失效生命周期和重复 `independence_key`，再打开评分解释查看去重原因。
- Compose 服务未 healthy：执行 `docker compose logs db backend frontend`，优先检查 DSN、JWT 密钥和数据库卷权限。
- 页面刷新出现 404：确认通过 Nginx 或 Vite 访问；Nginx 已配置 SPA fallback，不能直接打开构建后的单个 HTML 文件。

## License

MIT

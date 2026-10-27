# 开发、测试与部署

本文件为当前执行步骤。每轮需求、测试与发布结果写入 [开发历史与验收](development-history.md)，不在此累积日期章节。

## 1. 环境与首次启动

本机开发使用 Go 1.26.9、Node.js 24 与 npm；生产需要 Docker Engine、Compose v2 及可运行沙箱的宿主机。以仓库锁文件和 Dockerfile 为构建依据。Compose 覆盖文件若使用 `!override`，需支持该标签的版本。

```bash
cp .env.example .env
# 分别设置 JWT_SECRET、POSTGRES_PASSWORD、REDIS_PASSWORD、MINIO_ROOT_PASSWORD
# 每个值独立生成；JWT 至少 32 字节，MinIO 用户不使用默认 minioadmin
make up
make sandbox-images
```

`make up` 依次构建三个应用、启动数据服务、执行 `api --migrate-only`、启动应用。不要跳过显式 SQL 迁移。Web 默认端口 25565；健康路径 `/healthz`；PostgreSQL 默认只绑定 127.0.0.1:15432，其他数据服务不对外开放。生产 `SEED_DATA=false`，仅隔离环境可启用演示账号。

直启开发分别在 `apps/api` 运行 `go run ./cmd/api`、`apps/worker` 运行 `go run ./cmd/worker`、`apps/web` 运行 `npm ci` 和 `npm run dev`。先准备隔离的数据库、Redis、对象存储及 `.env.example` 对应连接变量。Worker 的 OJ_WORK_ROOT 必须是宿主机与容器可见的同一持久路径，不使用易被系统清理的临时目录。

## 2. 实现与本地验证

1. 检查 Git 分支、工作区与作者；在 `codex/` 分支上开发，保留用户已有修改。
2. 更新功能、权限、OpenAPI、客户端与必要的新迁移；禁止修改已经执行的历史迁移。
3. API／Worker 各运行 `go test ./... -count=1`；Web 运行 `npm test -- --run` 和 `npm run build`。构建包含契约、类型、架构和入口包体检查。
4. 对关键状态和权限使用真实隔离 PostgreSQL／Redis 集成测试；界面覆盖桌面、手机及浅暗主题；Docker 判题在具备引擎的环境另外验证。
5. 写入本轮验收记录并检查 Markdown 链接，确认提交范围不含本机 MEMORY、密钥、日志或用户题面。

### 集成测试环境


`exam_security*_test.go` 使用真实 PostgreSQL，每个用例建立随机独立 schema，结束时只清理该 schema。必须提供本机专用数据库 `oj_exam_security_test`，不能指向生产库；Redis 使用专用回环端口 `56379`，不得复用生产实例。未提供变量时集成用例会明确跳过，不应把普通 `go test` 的通过等同于集成验收完成。

PowerShell 示例（服务需先独立启动）：

```powershell
$env:OJ_TEST_DATABASE_URL='host=127.0.0.1 port=55439 user=ojtest dbname=oj_exam_security_test sslmode=disable'
$env:OJ_TEST_REDIS_ADDR='127.0.0.1:56379'
Set-Location apps/api
go test ./... -count=1
```

测试先构造现有 ORM 基线，再运行正向 SQL 迁移，当前包含 `026_team_access_constraints.sql`。这是已有版本升级回归，不等于空库首次安装验证；既有空库安装的约束命名兼容问题需单独治理。`db/migrations_integration_test.go` 另行验证单文件失败的 SQL/台账回滚、修正重试、校验和保护和并发迁移连接锁。

浏览器验收使用 `TestExamBrowserFixture`（仅存在于测试代码）启动真实 API，明确绑定 `127.0.0.1:18086`。设置 `OJ_BROWSER_FIXTURE_ADDR=127.0.0.1:18086` 后执行：

```powershell
go test ./internal/handlers -run '^TestExamBrowserFixture$' -count=1 -v -timeout 0
```

另一个终端在 `apps/web` 执行生产构建并启动隔离预览：

```powershell
npm run build
node --input-type=module -e "import('vite').then(v=>v.preview({preview:{host:'127.0.0.1',port:4176,strictPort:true,proxy:{'/api':'http://127.0.0.1:18086'}}}))"
```

最后在 `apps/web` 运行 `node tests/e2e/exam-security-qa.mjs`。脚本限定回环地址，通过真实登录、进入考试、上报、交卷和加入/审核接口操作；不得用于生产。视口为 `1440×1100`、`390×844`，结果与截图输出到根目录 `tmp/exam-security-qa/`（忽略提交）。每次完整重跑应重新启动 fixture，以获得未交卷、未入组的数据；登录限流仍然生效。

随后在同一个 fixture 上运行 `node tests/e2e/governance-qa.mjs`，检查全局/练习/作业/比赛/题单提交详情、人工阅卷、复制代码位置和原文、申请/邀请分享入口以及三级职级权限。输出到 `tmp/governance-qa/`。两套脚本应顺序运行；不要对已经交卷的旧 fixture 重跑考试流程。

Worker 的恢复测试也使用上述数据库/Redis 变量，在 `apps/worker` 运行 `go test ./... -count=1`。Redis 逻辑库 14 用于 Worker 随机测试键，15 用于 Outbox 专用 Stream；不清空整个 Redis。故障通过停止测试租约续期、调整测试消息空闲时间、关闭测试客户端和取消 SQL 请求模拟，恢复阶段连接真实 PostgreSQL/Redis。它们验证恢复边界，不等同于实际杀死生产进程或执行 Docker 沙箱。

判题终态由 fixture 预置以核验显示和脱敏，不伪装成 Worker 实际执行；图片对象使用受控 S3 传输测试验证版本和权限。真正的应用镜像构建、MinIO/Worker 沙箱端到端与生产健康检查仍须在 Docker 可用后另行执行。


## 3. 配置


| 变量 | 用途 |
| --- | --- |
| `JWT_SECRET` | JWT 签名密钥，Compose 中为必填 |
| `WEB_PORT` | Web 对外端口 |
| `POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD` | PostgreSQL 配置 |
| `MINIO_ROOT_USER`、`MINIO_ROOT_PASSWORD`、`MINIO_BUCKET` | 对象存储配置 |
| `CORS_ALLOWED_ORIGINS` | 允许的浏览器来源，逗号分隔 |
| `TRUSTED_PROXIES` | 受信反向代理 IP/CIDR，逗号分隔 |
| `OJ_WORK_ROOT` | 主 Worker 与宿主机共享的沙箱根目录 |
| `OJ_SECONDARY_WORK_ROOT` | 第二个 Worker 的独立宿主工作根，不能与主 Worker 重叠 |
| `SUBMISSION_MAX_RETRIES` | 基础设施错误最大重试次数 |
| `SUBMISSION_RETRY_IDLE_SECONDS` | Redis pending 任务回收时间 |
| `SMTP_*`、`MAIL_FROM*` | 注册、换绑和找回密码邮件配置 |

只把直接受控的代理地址加入 `TRUSTED_PROXIES`，因为接口限流依赖可信客户端 IP。生产跨域列表不得使用不受控的通配来源。


另需设置 `REDIS_PASSWORD`，API 与 Worker 必须一致。数据库角色密码需要真正通过数据库修改并同步配置，不能只改已有数据卷对应的环境变量。凭据轮换属于独立操作，普通应用发布不重新创建数据服务。

## 4. 候选构建与发布

本项目维护者已要求“本地验证 → 服务器构建、测试、启动 → 本机 Git GUI 提交 → GitHub”。按这个顺序执行，无须提前制造提交：

1. 记录本地和生产 HEAD、分支、工作区、全部生产容器 ID 与旧镜像。生产源码存在未审查变更或无法快进时先解决，不覆盖。
2. 使用临时 Git 索引从 HEAD 加入明确的任务文件，`write-tree` 记录候选树，再 `git archive <tree>` 导出。保留文件清单与 SHA-256；不得直接打包 Windows 工作区作为镜像源码。
3. 在服务器独立目录／Compose 项目构建 api、worker、web，使用独立数据库、Redis、MinIO、工作根与回环端口。真实判题、权限和页面交互都在候选环境执行，不在生产创建测试账号或题目。
4. 将生产数据库恢复到独立副本，执行候选迁移两次，核对历史迁移校验和、记录数、版本绑定及数据精度。完成升级与回退演练后再切换。
5. 备份生产数据库和配置，保留旧镜像标签。暂停 API 新写入，等待提交队列排空后同时暂停 worker 与 worker-secondary（两者消费同一 `oj.submissions` 流）；确认 `queued`／`running` 均为 0 后执行显式迁移。
6. 将已验收候选镜像标为应用发布镜像，仅重新创建 api、worker、worker-secondary、web。校验健康、实际镜像、授权会话、重要 API、公开页面和日志；数据服务容器及卷保持不变。
7. 正式验证通过后补记历史文档，按文件清单暂存；使用本机 Git GUI、确认的 TailingMing 身份提交。提交的 apps 树必须与已部署候选一致，允许最后补充验收文档。
8. 推送服务器 origin 的工作分支，生产部署分支仅 `merge --ff-only`；逐文件校验候选应用内容与提交一致。推送 GitHub 同名分支，不强推。记录实际提交 SHA。
9. 清理本轮独立测试资源及确认无引用的应用镜像，保留生产数据、回滚镜像和备份。

SQL 校验依赖原始字节，`.gitattributes` 固定迁移 LF。保留生产副本检查，不能用新建空库测试替代对已有校验值的验证。候选构建已完成时不要在上线阶段重新构建另一份未验证镜像。

## 5. 迁移 037 与回滚

037 将比赛奖项比例改为 `NUMERIC(9,6)`，保留范围与合计约束，并保留原整数值。旧 API 使用整数模型，不保证能读取小数列，不能直接把 API 镜像切回旧版后继续写入。

回退须先停止相关写入并检查三列是否全部为整数。若均为整数，可在备份和隔离演练后以事务恢复三列整数类型并撤销仅 037 的迁移记录，随后启动旧应用；不得改动 001–036 台账或 SQL。若已有小数，禁止截断、四舍五入或直接恢复旧备份丢失新数据：保留兼容小数的新 API，修复或提供兼容旧功能的版本，再实施回退。恢复数据库备份只用于明确冻结写入且确认无新数据的窗口。

其他迁移也要逐项判断：027／029 新工单类型与待补充状态不能交给不识别它们的旧程序；035／036 字段可以保留，但旧程序缺少加强后的撤销与范围保护。

## 6. 备份、恢复与清理

`BACKUP_ROOT=/opt/School-OJ-backups make backup` 暂停写入方，生成 PostgreSQL custom dump、MinIO／Redis 快照、配置、文件清单及校验和，完成或失败后恢复原先运行的服务。目录受 umask 077 保护，备份应复制到受控异地存储，不提交 Git。

`make restore BACKUP=/absolute/path` 只向新建 `school-oj-restore-*` 项目恢复并验证，不覆盖在线库、不启动副本 API／Worker 消费任务。旧备份缺新凭据时可使用受保护的 RESTORE_ENV_FILE，不能修改原备份和校验清单。

清理按明确的 Compose 项目、容器和镜像 ID 操作，不执行全局 prune，不删除生产数据卷、回滚标签或来源不明的共享缓存。`make cleanup` 仅清理本工作根过期沙箱及本项目退出的一次性容器。独立验收项目的数据卷只有确认属于本次夹具、报告与备份已保存后才删除。

## 7. 监控、旧接口与其他部署

API `/metrics`、Worker `:9091/healthz` 和 `/metrics` 仅内部可见，外部 `/metrics` 返回 404。Prometheus 告警规则文件需接入实际监控才能通知，不把文件存在当作已配置告警。`make legacy-report` 检查前 30 个完整 UTC 日覆盖和旧 `/api` 用量；缺日、Redis 数据丢失或仍有调用均不能宣告下线。

Kubernetes 清单要求专用 Docker 节点，不承诺纯 containerd 可用；先执行同配置的迁移 Job，再更新 Deployment。Worker 扩容需要独立节点与工作根。JPlag 由受控版本 JAR 配置；未配置时轻量后备报告仅保证业务可测试，不等同正式查重引擎。

## 8. 两份提交并发与容量验收

Compose 默认运行 `worker` 和 `worker-secondary`，两者复用同一镜像、数据库、Redis Stream 和 `SUBMISSION_GROUP=judge-workers`，消费者身份默认使用各自容器 hostname。每个进程串行处理一份提交，总并发为 2；不要额外对两个服务使用 `--scale`，也不要把 `SANDBOX_CPU` 当作并发数。

两个实例分别挂载 `OJ_WORK_ROOT` 和 `OJ_SECONDARY_WORK_ROOT`，seccomp 文件路径同步设置；宿主与容器的绝对路径必须相同。单个控制容器上限 1.5 CPU／768 MiB，每个独立判题沙箱上限 1 CPU，运行内存仍按题目限制，编译至少 1 GiB。控制容器的内存限额不包含它通过 Docker socket 启动的沙箱。配置共用 [Compose YAML 合并](https://docs.docker.com/reference/compose-file/fragments/)，任务分配沿用 [Redis 消费组](https://redis.io/docs/latest/commands/xreadgroup/)。

`make up` 启动两个 Worker；备份必须同时暂停两者，清理脚本分别清理两个工作根。监控应分别检查两个容器的健康；`:9091` 不映射到宿主端口。扩缩容验证须同时覆盖唯一消费者、排队任务分配、租约恢复、终态幂等和工作目录隔离。

`scripts/judge_capacity.py` 在真实 API → PostgreSQL Outbox → Redis → 两个 Worker → MinIO/Docker → 结果入库链路运行合成负载。只允许名称以 `school-oj-judge-qa-` 开头、开启种子数据且 Web 仅绑定回环地址的独立环境；不向生产库写压测用户、题目或提交。执行：`python3 scripts/judge_capacity.py /opt/School-OJ-judge-qa-YYYYMMDD`。脚本输出突发吞吐、连续 5 分钟每分钟 20 份的响应耗时、并发峰值、宿主 CPU／可用内存和生产健康采样；可用内存低于 2 GiB 或健康请求失败时停止加压。中断后先停止测试 API／Worker 并确认测试沙箱结束，再清理明确属于该项目的容器、卷和工作目录。

20 份／分钟指完整提交完成率，不能视作任何题目都保证的固定速率。测试点数量、每点 CPU 时间、语言和编译复杂度都会改变吞吐；报告必须同时记录负载定义和排队延迟，不能仅按接口接收成功数宣称达标。

迁移 038 仅新增排名名单列，既有场次默认为全体。上线前在生产数据库的隔离副本执行两次并核验旧迁移校验和及记录数。旧 API 会忽略新增名单列；若已经保存指定名单，不能直接回退旧 API 后继续提供榜单，否则排名范围将扩大。保留新 API 修复，或在冻结写入并确认无新名单写入的发布窗口内回退；不删除真实业务数据。

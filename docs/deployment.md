# 开发、测试与部署

工单改造的迁移 027、隔离验收及发布闸口见 [工单反馈与发布验收](ticket-feedback.md)。该版本回滚前必须停止工单写入，防止旧程序处理新类型数据。

## 1. 环境要求

- Docker Engine 与 Docker Compose 2.20 或更高版本
- 本地直接开发时需要 Go 1.26.9、Node.js 24 LTS 和 npm
- 判题宿主机需要能够运行 Docker，并预先准备 C/C++、Python、Java 沙箱镜像
- 生产环境必须使用强随机 `JWT_SECRET`，并替换数据库与 MinIO 默认凭据

## 2. Compose 快速启动

```bash
cp .env.example .env
# 编辑 .env，至少设置 32 字节以上的随机 JWT_SECRET
docker compose up -d --build
```

首次接收提交前准备沙箱镜像：

```bash
./scripts/pull_sandbox_images.sh
docker compose restart worker
```

默认入口：

- Web：`http://localhost:${WEB_PORT}`，`.env.example` 默认端口为 `25565`
- API 健康检查：通过 Web 反向代理访问 `/healthz`
- PostgreSQL：仅绑定 `127.0.0.1:15432`
- Redis、MinIO 和 API 默认只在 Compose 网络中提供服务

本地种子数据由 `SEED_DATA` 控制，默认关闭。只有隔离开发环境才设置 `SEED_DATA=true`；生产已有账号不受关闭种子数据影响。API 启动时拒绝短于 32 字节或以 `dev-secret` 开头的 JWT 密钥；更换签名密钥会使现有登录令牌失效，用户需重新登录。

## 3. 本地直接开发

分别启动三个应用：

```bash
cd apps/api && go run ./cmd/api
cd apps/worker && go run ./cmd/worker
cd apps/web && npm install && npm run dev
```

基础设施仍可由 Compose 提供。Worker 默认使用 `/var/lib/school-oj-worker` 作为宿主机可见工作目录；该目录必须位于持久存储，不能放在可能被系统清理的 `/tmp` 下。需要调整时在启动 Compose 前设置 `OJ_WORK_ROOT`。

## 4. 测试与构建

运行全部单元测试：

```bash
make test
```

也可分别执行：

```bash
cd apps/api && go test ./...
cd apps/worker && go test ./...
cd apps/web && npm run api:check
cd apps/web && npm test -- --run
cd apps/web && npm run build
docker compose config --quiet
```

涉及页面结构、权限或结果可见性的修改，还应使用 Playwright 覆盖桌面和移动视口，检查控制台错误、网络失败、横向溢出和关键状态。

### 4.1 考试与团队准入集成验收

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

### 4.2 本轮验收记录（2026-09-07）

以下为开发基线 `99c78d5` 上的历史本地验收记录；当前服务器发布结果见 4.3。

| 检查 | 结果 |
| --- | --- |
| Web 完整单测 | 29 个测试文件、104 项通过 |
| 契约与生产构建 | 164 个操作、生成一致性、前端架构检查、vue-tsc 和 Vite 构建通过；保留已有大分包体积警告 |
| API / Worker | `go test ./... -count=1` 通过；已提供隔离 PostgreSQL/Redis 变量，故障恢复与权限集成用例实际执行 |
| 数据库环境 | 本地 PostgreSQL 18、Redis 7；生产 PostgreSQL 16 的容器部署验收不以此替代 |
| Playwright | 桌面 1440×1100、手机 390×844，考试/准入 68 个检查点，共享详情/治理 18 个检查点通过；无运行时异常 |
| 视觉复核 | 复制位置、原文、行号、白底、三级职级、弹窗横向边界与手机阅卷布局通过；记录在 `tmp/governance-qa/design-qa.md` |
| Compose / 文档 | 根 Compose 与部署入口配置通过；259 个文档相对链接、`git diff --check` 通过 |
| 应用镜像构建 | 未完成：本机 Docker Desktop 的 `dockerInference` 初始化失败，Linux Engine 命名管道不存在；实际执行构建命令失败，未重置 Docker |
| 发布 | 等待维护者确认本机 Git 提交，以及容器构建验收安排；服务器、数据卷及 GitHub 本轮均未变更 |

截图和 JSON 检查结果属于忽略提交的本地产物，不长期纳入源码仓库。`final result: passed` 仅指页面验收，不表示容器发布已通过。

### 4.3 完整治理与组件优化发布（2026-09-18）

应用版本 `269729c` 已从旧版 `4e781b5` 更新到生产服务器；此次包含此前待发布的权限、契约、迁移、任务恢复治理及复用组件优化。服务器部署分支以快进方式更新，API、Worker、Web 均重新构建并启动，应用镜像保留 `269729c` 版本标签。

| 检查 | 结果 |
| --- | --- |
| 服务器 API / Worker | 完整 Go 测试通过；使用独立 PostgreSQL 16、Redis 7 容器，权限、迁移与任务恢复集成用例实际执行 |
| 服务器 Web | 32 个测试文件、117 项通过；164 个操作契约检查、类型检查、生产构建通过 |
| 真实 Docker 判题沙箱 | 正常执行及资源计量、CPU 超时、内存超限、运行错误场景通过 |
| 数据库升级 | 迁移 026 前角色/唯一创建者预检通过；升级后迁移校验和、CHECK 约束与唯一索引校验通过 |
| 运行状态 | API / Worker 健康，Web 正常运行，外部 `/healthz` 返回 `status: ok`；运行镜像 ID 与发布记录一致，应用日志无 panic 或 HTTP 5xx |
| 线上浏览器 | 1440×1100、390×844 两个视口的概览、登录页、题库、题面共 8 项只读检查通过；无运行时异常、失败响应或整页横向溢出。此次线上检查未使用登录会话；完整鉴权交互见此前隔离环境的 86 项验收 |
| 备份与清理 | PostgreSQL 备份及旧镜像保留；原 PostgreSQL / Redis / MinIO 容器 ID 不变。已移除本次 4 个测试镜像和 13 个无用构建层，释放约 2.27 GB；测试容器已清理，数据卷未删除 |

服务器发布证据与备份保存在 `/opt/School-OJ-backups/release-components-269729c6449a/`，其中 `release.json` 记录旧/新镜像、数据容器 ID、备份校验和及清理结果。线上页面截图与结果在忽略提交的 `tmp/components-production-qa/`。源码已移除不安全的旧格式化逻辑、失效标题参数与样式；完整范围见 [复用组件性能与生命周期](component-performance.md)。

### 4.4 工单与提交界面更新（2026-09-23）

应用版本 `3986614` 已从 `435a226` 更新上线，API、Worker、Web 使用同一代码版本构建，数据服务保持原容器。此次包括工单返回／创建跳转、4 字说明、新题 50 附件、固定栏目题面编辑、Profile 工作台清理、默认公开代码、本人提交筛选及题目反馈入口调整。

服务器隔离 PostgreSQL 16 / Redis 7 下 API 140 项、Worker 36 项、Web 128 项测试通过；真实镜像两个视口 33 项交互、MinIO 题包与 Worker 实际判题、Docker 沙箱专项均通过。上线后健康、镜像匹配、管理员／学生只读权限与本人筛选、10 项线上浏览器检查通过，无 HTTP 5xx 或 panic。

已清理本次 8 个测试容器、4 个测试镜像和 13 个无用构建层，保留 6 个正式服务、回滚镜像、数据库及附件备份。证据目录为 `/opt/School-OJ-backups/release-ticket-update-3986614b4ad4/`，完整说明见 [工单反馈与发布验收](ticket-feedback.md)。

## 5. 关键环境变量

| 变量 | 用途 |
| --- | --- |
| `JWT_SECRET` | JWT 签名密钥，Compose 中为必填 |
| `WEB_PORT` | Web 对外端口 |
| `POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD` | PostgreSQL 配置 |
| `MINIO_ROOT_USER`、`MINIO_ROOT_PASSWORD`、`MINIO_BUCKET` | 对象存储配置 |
| `CORS_ALLOWED_ORIGINS` | 允许的浏览器来源，逗号分隔 |
| `TRUSTED_PROXIES` | 受信反向代理 IP/CIDR，逗号分隔 |
| `OJ_WORK_ROOT` | Worker 与宿主机共享的沙箱根目录 |
| `SUBMISSION_MAX_RETRIES` | 基础设施错误最大重试次数 |
| `SUBMISSION_RETRY_IDLE_SECONDS` | Redis pending 任务回收时间 |
| `SMTP_*`、`MAIL_FROM*` | 注册、换绑和找回密码邮件配置 |

只把直接受控的代理地址加入 `TRUSTED_PROXIES`，因为接口限流依赖可信客户端 IP。生产跨域列表不得使用不受控的通配来源。

## 6. 生产更新流程

更新应用时保持 PostgreSQL、Redis 和 MinIO 数据服务不变，仅重建 API、Worker 和 Web：

```bash
git pull --ff-only
docker compose config --quiet
docker compose build api worker web
docker compose up -d --no-deps api worker web
docker compose ps
curl -fsS http://127.0.0.1:${WEB_PORT:-25565}/healthz
```

推荐发布顺序：

1. 取得维护者发布授权后，使用本机 Git 提交并推送部署用 origin。备份 PostgreSQL，记录当前提交和应用镜像编号，保留带标签的回滚镜像。
2. 以快进方式更新部署分支，禁止在生产目录制造合并提交。
3. 校验 Compose 配置，构建三个应用镜像。
4. 使用 `up -d --no-deps api worker web` 替换应用容器，不重建数据服务或删除卷。
5. 等待 API、Worker 及依赖服务健康。
6. 运行健康接口、登录、题库和关键业务 Playwright 验收。
7. 验收通过后，只清理核实属于本项目的闲置资源；最后推送 GitHub 同名分支，不强推。失败则恢复上一应用提交/镜像，不上传失败版本。

API 和 Worker 必须部署同一 Git 版本，避免新数据库结构或提交上下文被旧 Worker 错误消费。

发布镜像必须从 Linux Git checkout 或 `git archive <commit>` 的原始提交内容构建，不可把 Windows 工作区直接打包后作为候选镜像。SQL 迁移校验和依赖文件字节，CRLF 与 LF 不同；`.gitattributes` 固定迁移为 LF。发布前须用生产数据库副本执行真实 API 镜像启动与重启，不能仅凭使用同一换行符建立的测试 schema 判断历史迁移兼容。

迁移 026 前执行只读预检：确认 `team_memberships.role` 没有 `owner/admin/member` 以外的值、每个团队没有多个 `owner`。发现异常应停止部署并人工核对，不自动降级用户或删除关系。迁移失败会阻止 API 启动；不要通过改写已执行 SQL 的方式绕过检查。

## 7. 回滚与清理

如果新容器未通过健康检查，切回更新前提交或镜像，重新创建 API、Worker 和 Web；不要回滚或删除数据卷。数据库迁移应保持向前兼容，生产迁移前仍应完成备份。

清理前查看 `docker system df`，按 `com.docker.compose.project=school-oj` 标签、应用服务和镜像引用核对归属，只删除确认无用的停止容器及悬空镜像，并保留回滚镜像。构建缓存只有能确认属于本项目或使用专用 Builder 时才清理，不能确认归属就保留；任何情况下不清理数据卷。

注意：当前 `scripts/cleanup_docker.sh` 包含宿主机级 `prune`，没有项目过滤。它虽不删除卷，也可能影响其他项目；多项目服务器不要直接执行，应使用核实后的资源 ID 或项目过滤操作。`DOCKER_CACHE_MAX_AGE` 只限制缓存年龄，不限制项目归属。

## 8. Kubernetes 与 JPlag

Kubernetes 参考清单位于 `deploy/k8s/school-oj.yaml`。使用前需要构建并推送三个应用镜像、配置 PostgreSQL/Redis/MinIO，并替换清单中的密钥。

原生 JPlag 通过 `JPLAG_JAR_PATH` 配置。未挂载 JAR 时，API 会生成基于轻量令牌重叠算法的可测试报告，保证业务流程可验证，但正式查重应配置受控版本的 JPlag。

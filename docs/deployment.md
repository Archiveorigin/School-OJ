# 开发、测试与部署

## 1. 环境要求

- Docker Engine 与 Docker Compose 2.20 或更高版本
- 本地直接开发时需要 Go 1.24、Node.js 和 npm
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

本地种子数据由 `SEED_DATA` 控制。开发环境可保留管理员、教师和学生测试账号；生产环境必须更换初始凭据或关闭种子数据。

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

测试先构造现有 ORM 基线，再运行正向 SQL 迁移，包含 `025_exam_activity.sql`。这是已有版本升级回归，不等于空库首次安装验证；既有空库安装的约束命名兼容问题需单独治理。

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

判题终态由 fixture 预置以核验显示和脱敏，不伪装成 Worker 实际执行；图片对象使用受控 S3 传输测试验证版本和权限。真正的应用镜像构建、MinIO/Worker 沙箱端到端与生产健康检查仍须在 Docker 可用后另行执行。

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
docker compose up -d --remove-orphans
docker compose ps
curl -fsS http://127.0.0.1:${WEB_PORT:-25565}/healthz
```

推荐发布顺序：

1. 记录当前提交和应用镜像编号，准备回滚点。
2. 以快进方式更新部署分支，禁止在生产目录制造合并提交。
3. 校验 Compose 配置，构建三个应用镜像。
4. 使用 `up -d --remove-orphans` 替换应用容器。
5. 等待 API、Worker 及依赖服务健康。
6. 运行健康接口、登录、题库和关键业务 Playwright 验收。
7. 验收通过后再清理无用镜像和缓存并推送最终远端。

API 和 Worker 必须部署同一 Git 版本，避免新数据库结构或提交上下文被旧 Worker 错误消费。

## 7. 回滚与清理

如果新容器未通过健康检查，切回更新前提交或镜像，重新创建 API、Worker 和 Web；不要回滚或删除数据卷。数据库迁移应保持向前兼容，生产迁移前仍应完成备份。

仓库提供安全清理脚本：

```bash
./scripts/cleanup_docker.sh
```

该脚本只清理停止容器、悬空镜像和超过七天的构建缓存，不删除卷。可通过 `DOCKER_CACHE_MAX_AGE` 调整缓存年龄。执行更激进的镜像清理前先查看 `docker system df`。

## 8. Kubernetes 与 JPlag

Kubernetes 参考清单位于 `deploy/k8s/school-oj.yaml`。使用前需要构建并推送三个应用镜像、配置 PostgreSQL/Redis/MinIO，并替换清单中的密钥。

原生 JPlag 通过 `JPLAG_JAR_PATH` 配置。未挂载 JAR 时，API 会生成基于轻量令牌重叠算法的可测试报告，保证业务流程可验证，但正式查重应配置受控版本的 JPlag。

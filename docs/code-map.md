# 代码模块、接口与路由索引

本文档说明当前项目中各主要代码文件的职责，并建立“业务功能—后端接口—前端路由—实现文件”之间的对应关系。它用于定位代码入口，不替代[功能说明](features.md)、[系统架构](architecture.md)或[部署说明](deployment.md)。

## 1. 阅读约定

- HTTP 接口同时注册在 `/api` 和 `/api/v1` 下。本文接口表省略此前缀，例如 `GET /problems` 实际可通过 `GET /api/problems` 或 `GET /api/v1/problems` 访问。
- 除登录、注册、找回密码和公开题库读取外，接口默认需要 `Authorization: Bearer <token>`。
- `admin`、`teacher`、`student` 是基础角色；`users.can_author` 是可与基础角色并行的出题权限标识。
- 题目、考试、团队比赛和提交均可能绑定 `problem_version_id`。修改题目时应优先检查版本绑定，不能只修改 `problems` 当前记录。
- 前端路由在 [`apps/web/src/router/index.ts`](../apps/web/src/router/index.ts) 中集中声明；后端路由在 [`apps/api/internal/handlers/routes.go`](../apps/api/internal/handlers/routes.go) 中集中声明。

## 2. 总体调用结构

```text
浏览器（Vue 页面）
  └─ apps/web/src/api/client.ts
      └─ HTTP /api 或 /api/v1
          └─ Gin 路由 routes.go
              ├─ Handler：参数、权限、业务编排、响应脱敏
              ├─ PostgreSQL：业务数据、题目版本、提交、事务发件箱
              ├─ MinIO：题目包、题面资源、工单附件、查重报告
              └─ Redis：限流、提交消息流、提交状态通知
                  └─ Worker 消费消息
                      ├─ 读取提交绑定的题目版本
                      ├─ Docker 沙箱编译和运行
                      └─ 回写结果并发布状态事件
```

## 3. 根目录与运行入口

| 文件 | 结构作用 |
| --- | --- |
| [`README.md`](../README.md) | 项目总览、快速启动和文档入口。 |
| [`compose.yaml`](../compose.yaml) | 本地/生产常用的 Compose 总入口，编排 PostgreSQL、Redis、MinIO、API、Worker 和 Web。 |
| [`.env.example`](../.env.example) | 可配置环境变量样例；实际密钥放在未提交的 `.env`。 |
| [`Makefile`](../Makefile) | 封装构建、测试、启动等常用命令。 |
| [`apps/api/cmd/api/main.go`](../apps/api/cmd/api/main.go) | API 进程入口：加载配置、连接数据库/Redis/MinIO、执行迁移与种子、启动提交发件箱调度器和 Gin。 |
| [`apps/api/cmd/healthcheck/main.go`](../apps/api/cmd/healthcheck/main.go) | API 容器健康检查程序，访问 `/healthz` 判断服务是否可用。 |
| [`apps/api/openapi/openapi.json`](../apps/api/openapi/openapi.json) | HTTP 接口的 OpenAPI 单一事实源，用于服务端文档、路由一致性检查和前端 Client 生成。 |
| [`apps/worker/cmd/worker/main.go`](../apps/worker/cmd/worker/main.go) | 判题进程入口：准备沙箱、连接依赖并启动 Redis Streams 消费循环。 |
| [`apps/web/src/main.ts`](../apps/web/src/main.ts) | Web 入口：创建 Vue 应用并安装 Pinia、Router 和 Element Plus。 |
| [`apps/web/src/App.vue`](../apps/web/src/App.vue) | Web 根布局，承载全局导航、路由出口及考试锁定等应用级交互。 |
| [`apps/web/src/style.css`](../apps/web/src/style.css) | 全局设计令牌、布局、响应式和通用组件样式。 |

## 4. API 服务代码

### 4.1 基础设施与公共层

| 文件 | 主要职责 |
| --- | --- |
| [`internal/config/config.go`](../apps/api/internal/config/config.go) | 读取服务地址、数据库、Redis、MinIO、JWT、SMTP、JPlag、CORS、可信代理和迁移开关等环境变量。 |
| [`internal/db/db.go`](../apps/api/internal/db/db.go) | 创建 GORM/PostgreSQL 连接，并执行旧数据兼容、难度回填、出题权限迁移和孤立关系清理。 |
| [`internal/db/migrations.go`](../apps/api/internal/db/migrations.go) | 维护迁移台账，按编号读取内嵌 SQL，并兼容旧库基线。 |
| [`migrations/embed.go`](../apps/api/migrations/embed.go) | 通过 Go Embed 将 `migrations/*.sql` 编译进 API 可执行文件。 |
| [`internal/models/models.go`](../apps/api/internal/models/models.go) | GORM 领域模型和枚举的集中定义，是数据库结构与 Handler 数据关系的主要代码入口。 |
| [`internal/middleware/auth.go`](../apps/api/internal/middleware/auth.go) | JWT 签发与校验、必选/可选认证、角色中间件和当前用户读取。 |
| [`internal/middleware/security.go`](../apps/api/internal/middleware/security.go) | 安全响应头和基于 Redis Lua 令牌桶的 IP/用户限流。 |
| [`internal/rbac/rbac.go`](../apps/api/internal/rbac/rbac.go) | 基础角色能力判断，如课程管理、用户管理和审计查看。 |
| [`internal/streams/redis.go`](../apps/api/internal/streams/redis.go) | 连接 Redis，定义提交流 `oj.submissions`、入队函数和单提交通知频道。 |
| [`internal/services/audit.go`](../apps/api/internal/services/audit.go) | 将管理和关键业务操作写入审计日志。 |
| [`internal/services/mailer.go`](../apps/api/internal/services/mailer.go) | SMTP 邮件发送实现。 |
| [`internal/services/verification.go`](../apps/api/internal/services/verification.go) | 邮箱验证码生成、哈希、校验、过期和消费逻辑。 |
| [`internal/services/seed.go`](../apps/api/internal/services/seed.go) | 初始化 MinIO Bucket，并在开发配置开启时写入示例账号和示例题目。 |
| [`internal/jplag/service.go`](../apps/api/internal/jplag/service.go) | 准备查重源文件、调用 JPlag 或轻量回退算法、上传报告并更新任务状态。 |

### 4.2 Handler 文件分工

| 文件 | 对应功能 |
| --- | --- |
| [`handlers/routes.go`](../apps/api/internal/handlers/routes.go) | 创建 Gin、设置 CORS/安全中间件，并将全部接口同时挂载到 `/api` 与 `/api/v1`。 |
| [`openapi/embed.go`](../apps/api/openapi/embed.go) | 将 OpenAPI 规范嵌入 API 二进制，供 `GET /openapi.json` 返回。 |
| [`handlers/server.go`](../apps/api/internal/handlers/server.go) | 主要业务 Handler：登录、用户/权限、课程班级、旧题目读写兼容、作业、考试、提交、查重和审计；同时包含跨业务访问控制与统计辅助函数。 |
| [`handlers/account.go`](../apps/api/internal/handlers/account.go) | 注册验证码、注册、找回密码、个人资料、换绑邮箱、改密、注销和反馈。 |
| [`handlers/problem_catalog.go`](../apps/api/internal/handlers/problem_catalog.go) | 公共题库分页、关键字/难度/标签/完成状态筛选、通过率和可选标签聚合。 |
| [`handlers/problem_change_tickets.go`](../apps/api/internal/handlers/problem_change_tickets.go) | 工单创建、查询、附件权限、取消、驳回和执行；执行新增/替换时生成版本，删除时归档题目。 |
| [`handlers/problem_test_uploads.go`](../apps/api/internal/handlers/problem_test_uploads.go) | 测试点大文件分片的临时接收与合并辅助逻辑；当前公开写接口已由工单入口接管。 |
| [`handlers/problem_versions.go`](../apps/api/internal/handlers/problem_versions.go) | 将不可变题目版本覆盖到运行时题目快照，并为考试/比赛题目关系补全绑定版本。 |
| [`handlers/submission_events.go`](../apps/api/internal/handlers/submission_events.go) | 提交状态 SSE：先返回数据库快照，再订阅 Redis 通知直至终态。 |
| [`handlers/submission_visibility.go`](../apps/api/internal/handlers/submission_visibility.go) | OI 与封榜期的统一服务端脱敏，隐藏状态、分数、测试点、资源消耗等结果。 |
| [`handlers/teams.go`](../apps/api/internal/handlers/teams.go) | 团队、成员申请、比赛、赛题、三赛制排行榜、团队题单、讨论与团队范围提交。 |

### 4.3 认证、账号与个人资料接口

| 方法与路由 | Handler/文件 | 作用与权限 |
| --- | --- | --- |
| `GET /healthz` | `health` / `server.go` | 容器和负载均衡健康检查，不在 `/api` 前缀下。 |
| `GET /openapi.json` | `Router` / `routes.go` | 返回编译进 API 的 OpenAPI 单一契约，不在 `/api` 前缀下。 |
| `POST /auth/login` | `login` / `server.go` | 邮箱密码登录，带 IP 限流并返回 JWT。 |
| `POST /auth/send-code` | `sendEmailCode` / `account.go` | 发送注册或找回密码验证码，带 IP 限流。 |
| `POST /auth/register` | `register` / `account.go` | 验证邮箱验证码并创建账号。 |
| `POST /auth/password-reset` | `resetPassword` / `account.go` | 使用验证码重置密码。 |
| `GET /me` | `me` / `server.go` | 返回当前用户、基础角色和 `can_author` 权限。 |
| `GET /me/active-exam` | `activeExam` / `server.go` | 返回学生正在进行且需要锁定界面的考试。 |
| `GET/PUT /profile` | `getProfile`、`updateProfile` / `account.go` | 读取或更新姓名、头像等个人信息。 |
| `POST /profile/password` | `updateProfilePassword` / `account.go` | 校验原密码后修改密码。 |
| `POST /profile/email-code`、`POST /profile/email` | `sendProfileEmailCode`、`rebindEmail` / `account.go` | 发送换绑验证码并更新邮箱。 |
| `DELETE /profile` | `deleteProfile` / `account.go` | 逻辑注销当前账号并使其不可登录。 |
| `POST /feedback` | `createFeedback` / `account.go` | 提交用户反馈。 |
| `GET /author-applications/me`、`POST /author-applications` | `server.go` | 查看或提交本人的出题资格申请。 |

### 4.4 题库、题目版本与工单接口

| 方法与路由 | 实现文件 | 作用与权限 |
| --- | --- | --- |
| `GET /problems` | `handlers/server.go` | 兼容题目列表接口，按当前用户可见范围返回题目。 |
| `GET /problems/catalog` | `handlers/problem_catalog.go` | 题库分页目录；支持 `page`、`page_size`、关键字、难度、多个标签和个人状态筛选。 |
| `GET /problems/:id` | `handlers/server.go` | 按数字 ID 或展示编号读取题目；校验公共题库、课程、考试和团队范围。 |
| `GET /problems/:id/assets/*asset_path` | `handlers/server.go` | 从题目包/MinIO 读取题面图片等资源。 |
| `GET /problems/:id/tests` | `handlers/server.go` | 返回有权限题目的测试点目录。 |
| `GET /problems/:id/tests/download` | `handlers/server.go` | 下载题目测试数据包。 |
| `GET /problems/:id/tests/file/*file_path` | `handlers/server.go` | 下载单个测试输入或输出文件。 |
| `POST /problems/parse-markdown` | `handlers/server.go`、`services/markdown_parser.go` | 将批量 Markdown 解析为题目草稿，仅做解析，不直接发布。 |
| `POST /problems`、`POST /problems/upload`、`PUT/DELETE /problems/:id` | `handlers/routes.go` | 旧直接写入口，统一返回“请使用工单修改”，不再改变题目。 |
| `POST /prepared-problems`、`POST /prepared-problems/upload`、`PUT /prepared-problems/:id`、`POST /prepared-problems/:id/publish` | `handlers/routes.go` | 旧预备题直接写/上传/发布入口，同样被工单接管。只读 `GET /prepared-problems` 与 `GET /prepared-problems/:id` 保留。 |
| `PUT /problem-test-uploads/:upload_id/:chunk_index` | `handlers/routes.go` | 旧测试点分片写入口，当前返回工单提示；底层分片辅助代码仅为兼容保留。 |
| `POST /problem-change-tickets` | `handlers/problem_change_tickets.go` | 出题人创建新增、替换或归档工单；修改说明和“修改后的正确内容”参考附件均为必填，替换/归档还必须确认 `overwrite` 覆盖性操作。 |
| `GET /problem-change-tickets/mine` | 同上 | 查询本人发起的工单。 |
| `GET /problem-change-tickets/eligible-problems` | 同上 | 返回当前用户可发起替换/归档的题目。 |
| `GET /problem-change-tickets` | 同上 | 管理员查询待处理及历史工单。 |
| `GET /problem-change-tickets/:id` | 同上 | 申请人或管理员查看工单详情、状态及影响范围。 |
| `GET /problem-change-tickets/:id/attachment` | 同上 | 申请人或管理员下载参考附件。 |
| `POST /problem-change-tickets/:id/cancel` | 同上 | 申请人取消尚未完成的工单。 |
| `POST /problem-change-tickets/:id/reject` | 同上 | 管理员驳回工单并记录处理说明。 |
| `POST /problem-change-tickets/:id/apply` | 同上 | 管理员上传最终完整题包并执行新增/替换，或确认归档；事务中更新版本和未开始场次绑定。 |

题目包的解析、构建和校验集中在 [`internal/services/problem_package.go`](../apps/api/internal/services/problem_package.go)：负责 `manifest.json`、测试点配对、权重、比较器、资源路径、大小限制、BOM 清理及 ZIP 路径穿越防护。详细格式见[题目包规范](problem-package.md)。

### 4.5 课程、班级与作业接口

| 方法与路由 | Handler | 作用 |
| --- | --- | --- |
| `GET /courses` | `listCourses` | 按身份返回可见课程；管理端可包含归档课程。 |
| `POST /courses`、`PUT /courses/:id` | `createCourse`、`updateCourse` | 教师/管理员创建和修改课程。 |
| `DELETE /courses/:id`、`POST /courses/:id/archive` | `archiveCourse` | 课程逻辑归档或恢复。 |
| `GET /courses/preview`、`POST /courses/join`、`POST /courses/:id/leave` | `server.go` | 通过邀请码预览、加入和退出课程。 |
| `POST /courses/:id/classes` | `createClass` | 在课程下创建班级。 |
| `GET/POST /courses/:id/members`、`DELETE /courses/:id/members/:user_id` | `server.go` | 教师/管理员维护课程成员。 |
| `GET /courses/:id/students` | `listCourseStudents` | 汇总课程学生及其班级归属。 |
| `GET /classes`、`GET /me/classes` | `listClasses`、`myClasses` | 返回管理范围或当前用户班级。 |
| `GET /classes/join-preview`、`POST /classes/join` | `previewClassJoin`、`joinClassByCode` | 学生通过邀请码预览并加入班级。 |
| `POST /classes/:id/join`、`POST /classes/:id/leave` | `joinClass`、`leaveClass` | 按班级 ID 加入或退出。 |
| `PUT /classes/:id`、`DELETE /classes/:id`、`POST /classes/:id/archive` | `server.go` | 教师/管理员编辑和归档班级。 |
| `GET /classes/:id/students`、`DELETE /classes/:id/students/:user_id` | `server.go` | 查看和移除班级学生。 |
| `POST /classes/:id/students/import` | `importClassStudents` | 接收文本、CSV 或简单 XLSX，批量建号并加入课程/班级。 |
| `GET /assignments`、`GET /assignments/:id` | `server.go` | 返回作业目录或包含题目、个人进度的详情。 |
| `POST /assignments` | `createAssignment` | 教师/管理员创建作业并绑定题目。 |
| `GET /assignments/:id/report` | `assignmentReport` | 教师/管理员查看完成和得分汇总。 |
| `DELETE /assignments/:id` | `deleteAssignment` | 逻辑删除作业并清理未来释放关系。 |

### 4.6 考试与提交接口

| 方法与路由 | Handler/文件 | 作用 |
| --- | --- | --- |
| `GET /exams`、`GET /exams/:id` | `server.go` | 考试列表和详情；详情按用户身份返回可见题面、进度和结果。 |
| `POST /exams` | `createExam` / `server.go` | 教师/管理员创建考试，绑定题目版本、OI/IOI/ACM 赛制与封榜参数。 |
| `POST /exams/:id/finish` | `finishExam` / `server.go` | 学生主动交卷并记录结束时间。 |
| `GET /exams/:id/ranking` | `examRanking` / `server.go` | IOI/ACM 排行榜及封榜视图；OI 对参赛者不提供排行榜。 |
| `GET /exams/:id/report`、`GET /exams/:id/report/export` | `server.go` | 教师成绩详情和 Markdown 报告下载。 |
| `POST /exams/:id/submissions/:submission_id/judge` | `judgeManualExamSubmission` | 将主观/人工评阅提交转入阅卷流程。 |
| `PUT /exams/:id/submissions/:submission_id/grade` | `gradeManualExamSubmission` | 教师写入人工得分。 |
| `DELETE /exams/:id` | `deleteExam` | 逻辑删除考试。 |
| `POST /submissions` | `createSubmission` / `server.go` | 创建练习、作业、考试、团队比赛或题单提交；固定题目版本并在同一事务写入发件箱。 |
| `GET /submissions` | `listSubmissions` / `server.go` | 按当前用户权限和查询上下文列出提交。 |
| `GET /submissions/latest` | `latestSubmissions` / `server.go` | 获取指定题目/作业/考试/团队上下文的最近提交。 |
| `GET /submissions/:id` | `getSubmission` / `server.go` | 返回提交源码、状态和测试点详情，并执行访问控制与结果脱敏。 |
| `GET /submissions/:id/events` | `submissionEvents` / `submission_events.go` | SSE 实时推送排队、运行和终态结果；推送前同样执行脱敏。 |

提交可靠入队位于 [`internal/services/submission_outbox.go`](../apps/api/internal/services/submission_outbox.go)。它扫描 `submission_outbox`，把未发布事件写入 Redis Stream，成功后记录 `published_at`，失败则退避重试，避免“数据库已有提交但消息丢失”。

### 4.7 团队、比赛与题单接口

团队业务均由 [`handlers/teams.go`](../apps/api/internal/handlers/teams.go) 实现。`/teams/:id/contests/:contest_id/*` 与 `/contests/:contest_id/*` 是同一组 Handler 的两种路由入口；前者适合团队管理上下文，后者供独立比赛页面使用。

| 接口族 | 主要路由 | 作用 |
| --- | --- | --- |
| 团队 | `GET/POST /teams`、`GET/PUT /teams/:id`、`POST /teams/:id/join`、`POST /teams/:id/leave` | 查询、创建、设置、申请/邀请码加入和退出团队。 |
| 成员 | `GET/PUT/DELETE /teams/:id/members/*` | 查看成员，调整 owner/admin/member 权限，移除成员。 |
| 加入申请 | `GET /teams/:id/applications`、`PUT /teams/:id/applications/:application_id` | 团队管理员审核加入申请。 |
| 比赛 | `GET/POST /teams/:id/contests`、`GET/PUT/DELETE /contests/:contest_id` | 创建和维护草稿、已发布、进行中、已结束比赛。 |
| 发布与赛题 | `POST /contests/:contest_id/publish`、`POST/DELETE /contests/:contest_id/problems/*` | 发布比赛，按题目为最小单位绑定/移除赛题版本。 |
| 比赛提交 | `GET/POST /contests/:contest_id/submissions` | 查询比赛提交或创建指定赛题提交。 |
| 比赛排行 | `GET /contests/:contest_id/ranking` | 按 OI/IOI/ACM 规则聚合，并按身份、结束时间和封榜点隐藏结果。 |
| 团队题单 | `GET/POST /teams/:id/problem-sets`、`GET/PUT/DELETE /problem-sets/:set_id` | 创建和维护团队题单。 |
| 题单题目 | `POST/DELETE /problem-sets/:set_id/problems/*` | 将可访问题目加入或移出题单。 |
| 题单提交 | `GET/POST /problem-sets/:set_id/submissions` | 查询或创建团队题单上下文提交。 |
| 题单讨论 | `GET/POST /problem-sets/:set_id/discussions` | 团队成员读取和发布讨论。 |

完整的团队范围别名包括：`POST /teams/:id/contests/:contest_id/publish`、`POST/DELETE /teams/:id/contests/:contest_id/problems`、`GET/POST /teams/:id/contests/:contest_id/submissions`、`GET /teams/:id/contests/:contest_id/ranking`，以及 `GET/PUT/DELETE /teams/:id/problem-sets/:set_id`、`POST/DELETE /teams/:id/problem-sets/:set_id/problems`、`GET/POST /teams/:id/problem-sets/:set_id/submissions`、`GET/POST /teams/:id/problem-sets/:set_id/discussions`。其中删除赛题/题单题目的具体路径分别为 `/teams/:id/contests/:contest_id/problems/:problem_id`、`/contests/:contest_id/problems/:problem_id`、`/teams/:id/problem-sets/:set_id/problems/:problem_id` 和 `/problem-sets/:set_id/problems/:problem_id`；成员更新/删除路径为 `/teams/:id/members/:user_id`。

### 4.8 管理与审计接口

| 方法与路由 | 作用与权限 |
| --- | --- |
| `GET/POST /plagiarism/jobs` | 教师/管理员查询或创建 JPlag 查重任务。 |
| `GET /audit-logs` | 仅管理员查看审计记录。 |
| `GET/POST /users`、`GET/PUT/DELETE /users/:id` | 管理员查询、创建、编辑和逻辑删除用户。 |
| `POST /users/:id/reset-password` | 管理员重置指定用户密码。 |
| `GET /permissions` | 返回可扩展权限定义，目前包含出题权限。 |
| `PUT /users/:id/permissions/:permission` | 更新指定独立权限，例如 `problem_author`，不覆盖基础角色。 |
| `GET /author-applications`、`PUT /author-applications/:id/review` | 管理员查询并审核出题资格申请。 |
| `GET /problem-authors`、`DELETE /problem-authors/:id` | 查询或撤销出题权限的兼容管理接口。 |
| `GET /problem-reviews/mine`、`GET /problem-reviews` | 读取旧题目审批历史。 |
| `PUT /problem-reviews/:id/review`、`PUT /problem-reviews/:id/withdraw` | 旧写接口已停止变更，返回工单入口提示。 |

## 5. 数据模型与迁移

### 5.1 主要模型分组

所有结构定义均在 [`apps/api/internal/models/models.go`](../apps/api/internal/models/models.go)，Worker 只在 [`apps/worker/internal/models/models.go`](../apps/worker/internal/models/models.go) 复制判题必需的最小字段。

| 领域 | 主要结构 | 关系作用 |
| --- | --- | --- |
| 账号权限 | `User`、`AuthorApplication`、`EmailVerification`、`LoginAttempt`、`Feedback` | 基础角色、并行出题权限、邮箱验证、失败登录和反馈。 |
| 课程班级 | `Course`、`Class`、`CourseMembership`、`ClassMembership`、`ClassProblem` | 教师课程、学生归属、班级邀请码和题目释放。 |
| 题目 | `Problem`、`ProblemVersion`、`PreparedProblem`、`ProblemProgress` | 当前题目索引、不可变版本、预备题元数据和个人进度。 |
| 工单 | `ProblemChangeTicket`、`ProblemChangeImpactSummary`、`ProblemReview` | 新增/替换/归档工作流、影响范围和旧审批兼容。 |
| 作业考试 | `Assignment`、`AssignmentProblem`、`AssignmentAttempt`、`Exam`、`ExamProblem`、`ExamAttempt` | 教学任务、题目及版本绑定、尝试和交卷。 |
| 判题 | `Submission`、`SubmissionResult` | 源码、上下文、题目版本、总状态和逐测试点结果。 |
| 团队 | `Team`、`TeamMembership`、`TeamJoinApplication` | 团队设置、角色和加入申请。 |
| 团队内容 | `TeamContest`、`TeamContestParticipant`、`TeamContestProblem`、`TeamProblemSet`、`TeamProblemSetProblem`、`TeamDiscussion` | 比赛生命周期、赛题版本、题单和讨论。 |
| 管理追踪 | `PlagiarismJob`、`AuditLog` | 查重任务和关键操作审计。 |

### 5.2 SQL 迁移对应功能

| 迁移 | 主要内容 |
| --- | --- |
| [`001_init.sql`](../apps/api/migrations/001_init.sql) | 初始用户、课程、班级、题目、作业、考试、提交、查重和审计结构。 |
| [`002_account_profile.sql`](../apps/api/migrations/002_account_profile.sql) | 个人资料、邮箱验证、登录尝试和反馈。 |
| [`003_class_progress.sql`](../apps/api/migrations/003_class_progress.sql) | 班级题目关联与个人题目进度。 |
| [`004_prepared_problems.sql`](../apps/api/migrations/004_prepared_problems.sql) | 预备题库及班级题目发布时间。 |
| [`005_work_scoring_manual_review.sql`](../apps/api/migrations/005_work_scoring_manual_review.sql) | 作业/考试逻辑删除、尝试记录、分值和人工评阅字段。 |
| [`006_problem_assets_deleted.sql`](../apps/api/migrations/006_problem_assets_deleted.sql) | 题目逻辑删除字段。 |
| [`007_exam_finish.sql`](../apps/api/migrations/007_exam_finish.sql) | 考试主动交卷时间。 |
| [`008_problem_slug_active_unique.sql`](../apps/api/migrations/008_problem_slug_active_unique.sql) | 仅对未删除题目实施 slug 唯一约束。 |
| [`009_class_join_code.sql`](../apps/api/migrations/009_class_join_code.sql) | 班级邀请码。 |
| [`010_course_class_archived_roles.sql`](../apps/api/migrations/010_course_class_archived_roles.sql) | 课程/班级归档和成员角色约束。 |
| [`011_course_join_code.sql`](../apps/api/migrations/011_course_join_code.sql) | 课程邀请码。 |
| [`012_public_problem_bank.sql`](../apps/api/migrations/012_public_problem_bank.sql) | 预备题发布时间和公共题库发布语义。 |
| [`013_course_college.sql`](../apps/api/migrations/013_course_college.sql) | 课程所属学院。 |
| [`014_problem_privacy_author_applications.sql`](../apps/api/migrations/014_problem_privacy_author_applications.sql) | 题目难度、可见性和早期出题资格申请。 |
| [`015_problem_author_permissions_and_reviews.sql`](../apps/api/migrations/015_problem_author_permissions_and_reviews.sql) | `can_author` 并行权限和旧题目审批表。 |
| [`016_remove_problem_setter_role.sql`](../apps/api/migrations/016_remove_problem_setter_role.sql) | 移除独立 `problem_setter` 基础角色，迁移为 `can_author`。 |
| [`017_problem_difficulty_and_teams.sql`](../apps/api/migrations/017_problem_difficulty_and_teams.sql) | 统一题目难度并引入团队、成员、比赛和题单。 |
| [`018_team_contest_workspaces.sql`](../apps/api/migrations/018_team_contest_workspaces.sql) | 比赛赛题、比赛/题单提交上下文和题单讨论。 |
| [`019_team_contest_scoring_rule.sql`](../apps/api/migrations/019_team_contest_scoring_rule.sql) | 团队比赛旧 `score/penalty` 计分字段。 |
| [`020_contest_lifecycle_and_submission_indexes.sql`](../apps/api/migrations/020_contest_lifecycle_and_submission_indexes.sql) | 比赛 draft/published/running/closed 生命周期及提交查询索引。 |
| [`021_exam_scoring_rule.sql`](../apps/api/migrations/021_exam_scoring_rule.sql) | 考试计分规则字段及旧数据兼容。 |
| [`022_team_content_management_and_awards.sql`](../apps/api/migrations/022_team_content_management_and_awards.sql) | 比赛奖项比例、团队内容逻辑删除和管理约束。 |
| [`023_submission_outbox.sql`](../apps/api/migrations/023_submission_outbox.sql) | 可靠提交入队的事务发件箱。 |
| [`024_problem_change_tickets_and_scoring.sql`](../apps/api/migrations/024_problem_change_tickets_and_scoring.sql) | 不可变题目版本、工单、归档、考试/比赛/提交版本回填、OI/IOI/ACM 与封榜字段。 |

## 6. Worker 判题服务

| 文件 | 结构作用 |
| --- | --- |
| [`internal/config/config.go`](../apps/worker/internal/config/config.go) | 加载数据库、Redis、MinIO、消费者名称、镜像、超时、重试、沙箱目录和资源上限。 |
| [`internal/db/db.go`](../apps/worker/internal/db/db.go) | 建立 Worker 使用的 PostgreSQL/GORM 连接。 |
| [`internal/models/models.go`](../apps/worker/internal/models/models.go) | Worker 最小模型：题目、题目版本、提交、逐点结果和进度。 |
| [`internal/streams/consumer.go`](../apps/worker/internal/streams/consumer.go) | 创建/加入消费组，读取和回收 pending 消息，取得判题租约，读取 `submission.problem_version_id`，调用 Runner，事务回写并发布通知。 |
| [`internal/runner/package.go`](../apps/worker/internal/runner/package.go) | 解析完整题目包，校验 ZIP、清单、比较器和测试点路径，向 Runner 提供输入输出。 |
| [`internal/runner/docker.go`](../apps/worker/internal/runner/docker.go) | C/C++/Python/Java 编译运行、Docker 资源限制、超时/内存/输出限制分类、比较器、加权得分和诊断截断。 |
| [`seccomp/oj-seccomp.json`](../apps/worker/seccomp/oj-seccomp.json) | 判题容器系统调用白名单/限制配置。 |
| [`Dockerfile`](../apps/worker/Dockerfile) | 构建 Worker 镜像并带入 seccomp 配置。 |

判题主链路为：`createSubmission` 写 `submissions + submission_outbox` → API 调度器写 `oj.submissions` → `Consumer.Run` 取消息和租约 → 从 MinIO 下载提交所绑定版本的题包 → `DockerRunner.Judge` → 写 `submission_results` 和终态 → Redis Pub/Sub 通知 SSE。

## 7. Web 前端代码

### 7.1 公共客户端、状态和导航

| 文件 | 结构作用 |
| --- | --- |
| [`api/http.ts`](../apps/web/src/api/http.ts) | Axios 实例、API 基址和 Bearer Token 注入。 |
| [`api/generated/client.ts`](../apps/web/src/api/generated/client.ts) | 根据 OpenAPI 自动生成的 Schema 类型、操作表和 `generatedApi`；禁止手工修改。 |
| [`api/client.ts`](../apps/web/src/api/client.ts) | 对外汇总生成 Client、过渡期领域类型、最近提交查询和带认证 SSE 客户端。 |
| [`api/errors.ts`](../apps/web/src/api/errors.ts) | 将 Axios/服务端错误统一转换为中文可读消息。 |
| [`api/modules/courses.ts`](../apps/web/src/api/modules/courses.ts) | 课程列表的轻量 API 封装。 |
| [`api/modules/exams.ts`](../apps/web/src/api/modules/exams.ts) | 创建考试的 API 封装。 |
| [`api/modules/problems.ts`](../apps/web/src/api/modules/problems.ts) | 题目/预备题查询及旧创建、Markdown 解析兼容封装。 |
| [`stores/auth.ts`](../apps/web/src/stores/auth.ts) | 登录、Token 持久化、`/me` 恢复和退出。 |
| [`stores/classroom.ts`](../apps/web/src/stores/classroom.ts) | 当前用户班级列表缓存。 |
| [`stores/examLock.ts`](../apps/web/src/stores/examLock.ts) | 查询活动考试、维持强制考试锁并控制允许跳转范围。 |
| [`features/navigation/menu.ts`](../apps/web/src/features/navigation/menu.ts) | 主导航分组及按身份/出题权限过滤。 |
| [`components/AppSidebar.vue`](../apps/web/src/components/AppSidebar.vue) | 桌面/移动端应用侧栏、用户入口和课程加入入口。 |
| [`styles/tokens.css`](../apps/web/src/styles/tokens.css) | 全局语义视觉 Token、深色模式值和后台 Token 别名。 |

### 7.2 前端页面路由对应表

| 浏览器路由 | 页面文件 | 主要接口/结构作用 |
| --- | --- | --- |
| `/` | [`Dashboard.vue`](../apps/web/src/views/Dashboard.vue) | 公共概览；读取题目，登录后补充课程。 |
| `/login` | [`Login.vue`](../apps/web/src/views/Login.vue) | 调用 `POST /auth/login`。 |
| `/register` | [`Register.vue`](../apps/web/src/views/Register.vue) | 发送验证码并调用 `POST /auth/register`。 |
| `/forgot-password` | [`ForgotPassword.vue`](../apps/web/src/views/ForgotPassword.vue) | 发送验证码并调用 `POST /auth/password-reset`。 |
| `/profile` | [`Profile.vue`](../apps/web/src/views/Profile.vue) | 资料、密码、邮箱、注销及出题资格申请。 |
| `/problems` | [`ProblemsLayout.vue`](../apps/web/src/views/problems/ProblemsLayout.vue) + [`ProblemCatalog.vue`](../apps/web/src/views/problems/ProblemCatalog.vue) | 题库外壳与分页筛选目录，调用 `GET /problems/catalog`。 |
| `/problems/:id` | [`ProblemDetail.vue`](../apps/web/src/views/problems/ProblemDetail.vue) | 公开/受限题目详情、代码提交和工单入口。 |
| `/problems/:id/submissions` | [`ProblemSubmissions.vue`](../apps/web/src/views/problems/ProblemSubmissions.vue) | 指定题目的个人提交记录和详情。 |
| `/problem-changes`、`/problem-changes/new` | [`ProblemChangeTickets.vue`](../apps/web/src/views/problems/ProblemChangeTickets.vue) | 出题人“我的工单”和发起工单；附件上传、目标选择和取消。 |
| `/my/courses` | [`MyCourses.vue`](../apps/web/src/views/courses/MyCourses.vue) | 学生/教师课程入口。 |
| `/my/courses/:courseId` | [`CourseWorkspace.vue`](../apps/web/src/views/courses/CourseWorkspace.vue) + [`CourseOverview.vue`](../apps/web/src/views/courses/CourseOverview.vue) | 课程容器、概况和子路由出口。 |
| `/my/courses/:courseId/assignments` | [`Assignments.vue`](../apps/web/src/views/Assignments.vue) | 课程作业列表；教师可创建、查看报告和删除。 |
| `/assignments/:id` | [`AssignmentDetail.vue`](../apps/web/src/views/AssignmentDetail.vue) | 作业题目、进度、代码提交及提交记录。 |
| `/my/courses/:courseId/exams` | [`Exams.vue`](../apps/web/src/views/Exams.vue) | 课程考试列表；教师可查看/导出报告、人工评分和删除。 |
| `/exams/:id/*` | [`ExamDetail.vue`](../apps/web/src/views/ExamDetail.vue) | 考试数据和提交状态的父容器，向下级页面提供上下文。 |
| `/exams/:id/overview` | [`exam/ExamOverview.vue`](../apps/web/src/views/exam/ExamOverview.vue) | 考试说明、状态、时间和赛制。 |
| `/exams/:id/problems` | [`exam/ExamProblems.vue`](../apps/web/src/views/exam/ExamProblems.vue) | 考试题目导航和状态。 |
| `/exams/:id/submit` | [`exam/ExamSubmit.vue`](../apps/web/src/views/exam/ExamSubmit.vue) | 当前考试题目和提交入口。 |
| `/exams/:id/records` | [`exam/ExamRecords.vue`](../apps/web/src/views/exam/ExamRecords.vue) | 当前用户考试提交记录。 |
| `/exams/:id/ranking` | [`exam/ExamRanking.vue`](../apps/web/src/views/exam/ExamRanking.vue) | 调用考试排名接口并使用统一榜单组件。 |
| `/teams` | [`TeamList.vue`](../apps/web/src/views/teams/TeamList.vue) | 我的/可发现团队、创建团队和加入申请。 |
| `/teams/:teamSlug/*` | [`TeamWorkspace.vue`](../apps/web/src/views/teams/TeamWorkspace.vue) | 团队容器、设置、退出和子路由出口。 |
| `/teams/:teamSlug/contests` | [`TeamContests.vue`](../apps/web/src/views/teams/TeamContests.vue) | 团队比赛列表、创建和删除。 |
| `/contest/:contestId/description` | [`ContestWorkspace.vue`](../apps/web/src/views/teams/ContestWorkspace.vue) | 比赛说明、状态、赛制、时间和管理操作。 |
| `/contest/:contestId/problems` | 同上 | 比赛题目列表和题号跳转。 |
| `/contest/:contestId/problems/:label` | 同上 | 赛题详情、相邻题导航、本题记录和提交。 |
| `/contest/:contestId/scoreboard` | 同上 | IOI/ACM 排行榜；使用与考试一致的榜单 UI。 |
| `/teams/:teamSlug/problem-sets` | [`TeamProblemSets.vue`](../apps/web/src/views/teams/TeamProblemSets.vue) | 团队题单列表、创建和编辑。 |
| `/problem-set/:setId` | [`TeamProblemSetDetail.vue`](../apps/web/src/views/teams/TeamProblemSetDetail.vue) | 题单题目、提交、讨论和管理。 |
| `/teams/:teamSlug/members` | [`TeamMembers.vue`](../apps/web/src/views/teams/TeamMembers.vue) | 成员列表、加入审核和团队角色调整。 |
| `/submissions` | [`Submissions.vue`](../apps/web/src/views/Submissions.vue) | 当前用户全局提交历史与提交详情。 |
| `/admin` | [`admin/AdminLayout.vue`](../apps/web/src/views/admin/AdminLayout.vue) + [`admin/AdminHome.vue`](../apps/web/src/views/admin/AdminHome.vue) | 后台布局和教学概览，聚合考试、课程、班级、工单、资格申请和审计数据。 |
| `/admin/courses` | [`CourseList.vue`](../apps/web/src/views/CourseList.vue) | 课程、课程成员和班级创建。 |
| `/admin/courses/:id/students` | [`CourseStudents.vue`](../apps/web/src/views/CourseStudents.vue) | 课程学生与班级归属。 |
| `/admin/classes` | [`ClassList.vue`](../apps/web/src/views/ClassList.vue) | 班级、邀请码、学生和批量导入。 |
| `/admin/exams` | [`Exams.vue`](../apps/web/src/views/Exams.vue) | 考试管理视图。 |
| `/admin/exams/new` | [`ExamCreate.vue`](../apps/web/src/views/ExamCreate.vue) | 分步考试创建、题目选择、赛制/封榜设置和发布确认。 |
| `/admin/prepared-problems` | [`PreparedProblems.vue`](../apps/web/src/views/PreparedProblems.vue) | 预备题只读管理和工单入口。 |
| `/admin/problem-authors` | [`admin/ProblemTicketManagement.vue`](../apps/web/src/views/admin/ProblemTicketManagement.vue) | 管理员处理题目工单、下载附件、上传最终题包、执行或驳回。 |
| `/admin/plagiarism` | [`Plagiarism.vue`](../apps/web/src/views/Plagiarism.vue) | 创建和查看查重任务。 |
| `/admin/audit-logs` | [`AuditLogs.vue`](../apps/web/src/views/AuditLogs.vue) | 审计日志。 |
| `/admin/users` | [`admin/UserAccessLayout.vue`](../apps/web/src/views/admin/UserAccessLayout.vue) + [`Users.vue`](../apps/web/src/views/Users.vue) | 用户与权限模块外壳、用户增删改和重置密码。 |
| `/admin/users/permissions` | [`admin/PermissionManagement.vue`](../apps/web/src/views/admin/PermissionManagement.vue) | 出题资格申请审核和独立权限开关，为后续新增权限保留扩展结构。 |

兼容重定向也在 `router/index.ts` 中维护：旧 `/courses`、`/courses/list`、`/classes`、`/assignments`、`/exams`、`/admin/exam-rankings` 等入口转到新课程/后台结构；`/admin/author-permissions`（路由项 `author-permissions`）转到权限管理；`/teams/:teamId/problem-sets/:setId` 转到独立题单页；团队下的 `contests/:contestId` 转到独立比赛页。旧比赛 `#overview/#problems/#records/#ranking` 哈希分别转到 description、problems 或 scoreboard 子路由。

### 7.3 复用组件

| 组件文件 | 用途 |
| --- | --- |
| [`CodeEditor.vue`](../apps/web/src/components/CodeEditor.vue) | Monaco 代码编辑器和语言切换。 |
| [`SubmissionComposer.vue`](../apps/web/src/components/SubmissionComposer.vue) | 统一代码提交表单，配合草稿保存。 |
| [`SubmissionDetailDialog.vue`](../apps/web/src/components/SubmissionDetailDialog.vue) | 提交源码、状态、逐点结果和资源消耗详情。 |
| [`SubmissionStatusMark.vue`](../apps/web/src/components/SubmissionStatusMark.vue)、[`VerdictTag.vue`](../apps/web/src/components/VerdictTag.vue)、[`StatusBadge.vue`](../apps/web/src/components/StatusBadge.vue) | 提交状态和业务状态的统一视觉映射。 |
| [`LeaderboardBoard.vue`](../apps/web/src/components/LeaderboardBoard.vue) | 考试与比赛共用排行榜外壳、说明、封榜提示和表格选择。 |
| [`PenaltyLeaderboardTable.vue`](../apps/web/src/components/PenaltyLeaderboardTable.vue) | ACM 通过数/罚时榜单。 |
| [`ScoreLeaderboardTable.vue`](../apps/web/src/components/ScoreLeaderboardTable.vue) | IOI 分数榜单。 |
| [`MarkdownRenderer.vue`](../apps/web/src/components/MarkdownRenderer.vue) | Markdown、KaTeX 和题目包私有资源渲染。 |
| [`ProblemStatementView.vue`](../apps/web/src/components/ProblemStatementView.vue)、[`ProblemOverview.vue`](../apps/web/src/components/ProblemOverview.vue)、[`ProblemMetaCard.vue`](../apps/web/src/components/ProblemMetaCard.vue) | 题面、样例、限制和题目元信息。 |
| [`ProblemSwitcher.vue`](../apps/web/src/components/ProblemSwitcher.vue) | 作业/考试/比赛内的题号切换。 |
| [`ProblemTagSelector.vue`](../apps/web/src/components/ProblemTagSelector.vue) | 弹出式多选算法标签。 |
| [`ProblemTestDownloads.vue`](../apps/web/src/components/ProblemTestDownloads.vue) | 有权限用户查看、预览和下载测试数据。 |
| [`ListPagination.vue`](../apps/web/src/components/ListPagination.vue) | 列表分页器。 |
| [`CourseJoinDialogs.vue`](../apps/web/src/components/CourseJoinDialogs.vue) | 课程邀请码加入对话框。 |
| [`TeamIconUpload.vue`](../apps/web/src/components/TeamIconUpload.vue) | 团队图标选择、校验和预览。 |
| [`TeamContestAwardFields.vue`](../apps/web/src/components/TeamContestAwardFields.vue) | 金银铜获奖比例输入与校验。 |
| [`components/admin/AdminPageHeader.vue`](../apps/web/src/components/admin/AdminPageHeader.vue) | 后台页面统一标题、英文眉题和操作区。 |
| [`components/admin/AdminSectionHeader.vue`](../apps/web/src/components/admin/AdminSectionHeader.vue) | 后台卡片/区块统一标题和操作区。 |
| [`components/exams/*`](../apps/web/src/components/exams) | 考试创建向导头部、基本信息、选题、发布复核和总分摘要。 |

### 7.4 纯业务逻辑模块

| 文件 | 用途 |
| --- | --- |
| [`features/problems/problemMeta.ts`](../apps/web/src/features/problems/problemMeta.ts) | 难度、标签、题号、通过状态和前端筛选辅助函数。 |
| [`features/problems/changeTicket.ts`](../apps/web/src/features/problems/changeTicket.ts) | 工单草稿类型、必填校验和覆盖性操作模式。 |
| [`features/problems/testFileUpload.ts`](../apps/web/src/features/problems/testFileUpload.ts) | 大测试文件分片计划、进度和上传；保留给兼容/受控流程。 |
| [`features/exams/types.ts`](../apps/web/src/features/exams/types.ts) | 考试创建草稿、选题和请求类型。 |
| [`features/exams/builder.ts`](../apps/web/src/features/exams/builder.ts) | 自动题号、重复检查、总分、日期格式和创建草稿校验。 |
| [`features/exams/draft.ts`](../apps/web/src/features/exams/draft.ts) | 考试创建草稿的本地存储恢复。 |
| [`features/leaderboard/adapters.ts`](../apps/web/src/features/leaderboard/adapters.ts) | 把考试/比赛接口数据转换为统一榜单模型，并兼容旧 `score/penalty` 命名。 |
| [`features/leaderboard/sorting.ts`](../apps/web/src/features/leaderboard/sorting.ts) | 发布排名和表现排名排序。 |
| [`features/leaderboard/awards.ts`](../apps/web/src/features/leaderboard/awards.ts) | 奖项比例规范化、名额截止位和奖牌等级。 |
| [`features/leaderboard/types.ts`](../apps/web/src/features/leaderboard/types.ts) | 统一榜单行、题目、单元格、状态和奖项类型。 |
| [`features/assignments/assignmentMeta.ts`](../apps/web/src/features/assignments/assignmentMeta.ts) | 作业状态、分数文本、时间状态和列表筛选。 |
| [`features/submissions/drafts.ts`](../apps/web/src/features/submissions/drafts.ts) | 按用户、资源、题目和语言隔离的本地代码草稿。 |
| [`features/teams/contestAwards.ts`](../apps/web/src/features/teams/contestAwards.ts) | 团队比赛奖项比例默认值与总和校验。 |
| [`features/clipboard.ts`](../apps/web/src/features/clipboard.ts) | 现代 Clipboard API 与兼容复制回退。 |
| [`features/time.ts`](../apps/web/src/features/time.ts) | 统一日期时间格式化。 |

## 8. 部署、容器与运维文件

| 文件 | 结构作用 |
| --- | --- |
| [`apps/api/Dockerfile`](../apps/api/Dockerfile) | 多阶段构建 API 和健康检查二进制。 |
| [`apps/worker/Dockerfile`](../apps/worker/Dockerfile) | 构建 Worker，并提供判题运行所需 Docker 客户端/配置。 |
| [`apps/web/Dockerfile`](../apps/web/Dockerfile) | 构建前端静态文件并由 Web Server 提供服务。 |
| [`deploy/compose/compose.yaml`](../deploy/compose/compose.yaml) | 独立部署目录下的 Compose 编排版本。 |
| [`deploy/k8s/school-oj.yaml`](../deploy/k8s/school-oj.yaml) | Kubernetes 工作负载、Service、配置和存储声明。 |
| [`scripts/smoke.sh`](../scripts/smoke.sh) | 部署后的 API 冒烟检查。 |
| [`scripts/cleanup_docker.sh`](../scripts/cleanup_docker.sh) | 清理停止容器、悬空镜像和可安全删除的构建缓存，不清理业务数据卷。 |
| [`scripts/pull_sandbox_images.sh`](../scripts/pull_sandbox_images.sh) | 预拉取 C/C++、Python、Java 判题沙箱镜像。 |
| [`scripts/create_problem_zip.sh`](../scripts/create_problem_zip.sh) | 从规范目录生成可上传的完整题目 ZIP。 |
| [`scripts/jplag.sh`](../scripts/jplag.sh) | JPlag 命令行包装和参数转发。 |
| [`apps/web/scripts/generate-openapi-client.mjs`](../apps/web/scripts/generate-openapi-client.mjs) | 从 OpenAPI 生成 TypeScript Client。 |
| [`apps/web/scripts/check-openapi-routes.mjs`](../apps/web/scripts/check-openapi-routes.mjs) | 校验 OpenAPI 与 Gin 注册路由完全一致。 |

## 9. 测试文件与验证范围

| 测试文件 | 主要覆盖 |
| --- | --- |
| [`handlers/server_test.go`](../apps/api/internal/handlers/server_test.go) | Handler 业务、权限、计分、工单、版本和可见性等 API 行为。 |
| [`services/problem_package_test.go`](../apps/api/internal/services/problem_package_test.go) | 题目包构建/解析、测试点、比较器、资源与恶意路径。 |
| [`services/submission_outbox_test.go`](../apps/api/internal/services/submission_outbox_test.go) | 提交发件箱发布、失败重试和并发领取。 |
| [`middleware/auth_test.go`](../apps/api/internal/middleware/auth_test.go) | JWT 签名算法、可选认证和禁止查询参数 Token。 |
| [`middleware/security_test.go`](../apps/api/internal/middleware/security_test.go) | 限流键匿名化和安全响应头。 |
| [`db/migrations_test.go`](../apps/api/internal/db/migrations_test.go) | 内嵌迁移排序和关键迁移存在性。 |
| [`runner/package_test.go`](../apps/worker/internal/runner/package_test.go) | Worker 题包解析、BOM、权重、比较器和错误输入。 |
| [`runner/docker_test.go`](../apps/worker/internal/runner/docker_test.go) | 沙箱准备、限制、状态分类和指标解析。 |
| [`runner/docker_integration_test.go`](../apps/worker/internal/runner/docker_integration_test.go) | Docker 沙箱实际运行与资源指标集成。 |
| [`streams/consumer_test.go`](../apps/worker/internal/streams/consumer_test.go) | 消费重试、租约、题目版本读取和结果持久化。 |
| [`features/**/*.test.ts`](../apps/web/src/features) | 前端考试构建、草稿、提交草稿和剪贴板等纯逻辑单测。 |
| [`apps/web/tests/e2e`](../apps/web/tests/e2e) | Playwright 页面流程与设计验收脚本。 |

常用验证入口见[开发、测试与部署](deployment.md)。文档改动至少执行 Markdown 相对链接检查和 `git diff --check`；业务代码改动还应执行 API/Worker `go test ./...`、Web `npm test` 与 `npm run build`，涉及交互时补充 Playwright。

## 10. 按修改目标快速定位

| 要修改的功能 | 首要入口 | 通常还要检查 |
| --- | --- | --- |
| 新增或调整 HTTP 接口 | `handlers/routes.go` | 对应 Handler、`models.go`、权限中间件、前端调用和测试。 |
| 修改题目字段或题包 | `models.go`、`services/problem_package.go` | 新迁移、工单执行、版本快照、Worker `package.go`。 |
| 修改题目增删改流程 | `problem_change_tickets.go` | 工单前台/后台页面、附件权限、审计和未开始场次版本同步。 |
| 修改判题状态或资源限制 | Worker `runner/docker.go` | Consumer、API 提交视图、状态组件和安全文档。 |
| 修改提交创建或消息可靠性 | API `createSubmission`、`submission_outbox.go` | Worker Consumer、SSE、迁移和恢复测试。 |
| 修改 OI/IOI/ACM 规则 | API `examRanking`、`teamContestRanking` | 服务端脱敏、榜单 adapters/表格、封榜测试。 |
| 修改题库 UI/筛选 | `ProblemCatalog.vue` | `problem_catalog.go`、`ProblemTagSelector.vue`、`problemMeta.ts`。 |
| 修改考试创建 | `ExamCreate.vue`、`components/exams/*` | `features/exams/*`、`POST /exams` 和迁移约束。 |
| 修改比赛页面 | `ContestWorkspace.vue` | `router/index.ts`、`teams.go`、统一榜单组件。 |
| 修改权限体系 | `middleware/auth.go`、`rbac.go`、用户模型 | `PermissionManagement.vue`、导航守卫、接口角色限制和迁移。 |
| 修改后台首页 | `admin/AdminHome.vue` | 聚合接口的权限和失败降级。 |
| 修改部署拓扑 | 根 `compose.yaml` | Dockerfile、部署 Compose、K8s、环境变量和回滚说明。 |

## 11. 维护要求

1. 新增前端页面时，在 `router/index.ts` 注册路由，并同步更新本文“前端页面路由对应表”。
2. 新增后端接口时，在 `handlers/routes.go` 注册，同时补充权限、限流、审计和接口测试；不要只在 Handler 中增加未注册函数。
3. 修改数据库结构必须新增递增编号迁移，不直接改写已部署迁移；同步更新 `models.go` 和本文迁移表。
4. 涉及题目内容的写操作必须经过工单和不可变版本，不恢复旧直接写接口。
5. 涉及考试/比赛结果的响应必须经过服务端脱敏，不依赖前端隐藏。
6. 删除代码前应同时搜索前端路由、API 注册、组件引用、测试和文档，避免留下不可达接口或冗余组件。

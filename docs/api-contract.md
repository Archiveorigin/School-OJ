# OpenAPI 契约与 TypeScript Client

## 1. 单一事实源

HTTP 接口契约以 [`apps/api/openapi/openapi.json`](../apps/api/openapi/openapi.json) 为唯一规范源。它包含当前 `/api` 与 `/api/v1` 共用的路径、方法、`operationId`、认证要求、角色限制、请求体和核心响应结构。

API 启动后可通过 `GET /openapi.json` 读取同一份内嵌契约。服务端不维护第二份运行时文档，前端也不手写另一套路由常量。

规范中的路径不带 `/api` 前缀，`servers` 同时声明：

- `/api`：旧客户端兼容入口，仍正常提供服务；
- `/api/v1`：当前 Web 默认入口和新集成入口。

## 2. 生成 TypeScript Client

前端生成器位于 [`apps/web/scripts/generate-openapi-client.mjs`](../apps/web/scripts/generate-openapi-client.mjs)，输出文件为 [`apps/web/src/api/generated/client.ts`](../apps/web/src/api/generated/client.ts)。生成文件包含：

- OpenAPI `components.schemas` 对应的 TypeScript 类型；
- 全部 `operationId`、HTTP 方法和路径表；
- 自动替换路径参数、传递查询参数和请求体的 `requestOperation`；
- 按 `operationId` 调用的 `generatedApi`；
- 供统一响应拦截器使用的 Axios 兼容导出 `client`；业务页面不再调用它。

修改契约后执行：

```bash
cd apps/web
npm run api:generate
```

业务请求使用生成方法：

```ts
import { generatedApi } from '@/api/client'

const { data } = await generatedApi.login({
  body: { email, password },
})

const { data: submission } = await generatedApi.getSubmission({
  path: { id: submissionId },
})
```

文件上传、Blob 下载、超时等 Axios 配置可直接放在 options 中；JSON 请求体使用 `body`，查询参数使用 `query`，路径参数使用 `path`。

请求体支持 `FormData` 和 `Blob`，保留批量导入、上传进度和二进制下载。题面资源、测试文件的多级路径通过 `x-path-segments` 逐段编码，保留目录斜线；禁止 `.`、`..` 和空段。考试图片适配器验证考试及题目范围，不回退到普通题库接口。

生成方法默认采用契约响应类型。部分历史复杂报表仍保留显式泛型响应适配（含旧 `any`），本轮完成的是调用入口迁移，不是所有历史响应的运行时 Schema 校验；新增业务应补全 Schema 并使用默认类型，不继续扩大逃逸类型。

## 3. 契约一致性检查

[`apps/web/scripts/check-openapi-routes.mjs`](../apps/web/scripts/check-openapi-routes.mjs) 会比较 OpenAPI 与 [`handlers/routes.go`](../apps/api/internal/handlers/routes.go) 实际注册的每一个方法和路径，并检查 `operationId` 唯一性。生成器的 `--check` 模式同时确认生成文件未过期。

```bash
cd apps/web
npm run api:check
```

`npm run build` 已将 `api:check` 设为前置步骤，因此出现以下任一情况时生产构建会失败：

- Gin 新增了路由但 OpenAPI 未登记；
- OpenAPI 声明了服务端未注册的路由；
- 两个操作使用相同 `operationId`；
- 修改规范后没有重新生成 TypeScript Client；
- 页面重新引入直接 Axios 业务调用，或已收敛的团队/提交详情组件出现硬编码颜色。

## 4. 修改接口的顺序

1. 先修改 `apps/api/openapi/openapi.json` 的路径、请求和响应结构。
2. 修改 Handler、模型及 `handlers/routes.go` 使实现符合契约。
3. 运行 `npm run api:generate` 更新生成客户端。
4. 前端通过 `generatedApi` 使用新增或变更后的操作。
5. 运行 Web 构建、API/Worker 单测以及相关 Playwright 流程。

`check-architecture.mjs` 阻止业务页面重新直接调用 Axios。身份拦截器仍使用统一传输实例；SSE 和页面离开时的 keepalive 请求使用契约中的 operationId 路径。新增接口不得再创建独立路径常量或复制响应类型；可复用结构应补充到 OpenAPI `components.schemas`。

## 5. 扩展字段

规范使用少量 `x-*` 字段记录实现约束：

| 字段 | 作用 |
| --- | --- |
| `x-handler` | 对应的 Go Handler 名称，便于代码定位。 |
| `x-auth` | `public`、`optional` 或 `bearer`，标记认证方式。 |
| `x-roles` | 需要的基础角色列表。 |
| `x-path-segments` | 通配文件路径参数按目录段编码，而不是把目录斜线整体转义。 |

旧题目直接写接口仍保留在契约中，并标记 `deprecated: true`；它们只返回工单入口提示，不代表可以恢复直接修改题目。

## 6. 考试工作区与团队准入契约

当前包含 163 个 operationId。新增操作为 `enterExam`、`recordExamActivity`、`listExamActivity`、`getExamProblemAsset`、`previewTeamJoin`。详情见 [接口索引](code-map.md#46-考试与提交接口)。

- `getExam` 返回 `{ exam, problems, can_manage, can_submit, ranking_visible, manual_review, now, ... }`，学生需先显式入场，GET 不再建立考试记录。
- `getSubmission` 返回 `{ submission, results, case_summary }`，不是裸 Submission。`case_summary` 是 `{ passed, total } | null`；总数来自提交的 `problem_version_id`，隐藏结果时统计为空。
- `Team` 通过 `can_manage`、`can_edit_permissions` 驱动管理操作；`join_code` 只对管理者存在，`TeamJoinPreview` 不提供内部团队资料。
- `ExamActivityRequest` 每批 1–30 项，事件只允许 `window_leave`、`window_return`、`devtools_shortcut`；IP、用户及接收时间不由客户端指定。日志查询支持 `user_id/event/from/to/page/page_size`。
- 全局考试拦截错误：HTTP 403，`code: EXAM_SCOPE_REQUIRED`，附带 `exam_id`；状态查询失败为 HTTP 503，`EXAM_STATE_UNAVAILABLE` 或 `AUTH_STATE_UNAVAILABLE`。前端不得据此退出登录或解除考试锁。
- 同账号不同场次入场、已处理申请的相反审核结果冲突返回 409；重复入场和重复同向审核幂等，交卷后入场返回 403。

生成器支持 OpenAPI 3.0 `nullable`，并自动产出 `| null`，不得在前端手动消除统计或人工评分的空值类型。

## 7. 团队邀请码与职级契约

新增 `POST /team-invitations/preview`，Bearer 认证，请求体为 `{ "join_code": "..." }`。成功只返回 `TeamJoinPreview`，不自动入组，不包含公告、成员、所有者或邀请码。无效、过期及存在歧义的代码统一拒绝，避免返回不确定的私有团队。查询按账号限流，每分钟 20 次，Redis 不可用时失败关闭；成功响应 `Cache-Control: no-store`。

确认加入继续使用 `POST /teams/:id/join`。申请制无需邀请码；邀请制在事务中重新验证当前加入模式与邀请码。`PUT /teams/:id/members/:user_id` 只能设置 `admin` 或 `member`，不能产生第四种职级或转移创建者身份。数据库迁移 026 校验职级取值与每团队至多一个创建者；旧非法数据会阻止迁移，不自动改写人员权限。

错误响应统一包含 `error`，领域错误同时包含 `code`：`team_invalid_request`（400）、`team_forbidden` / `team_invitation_invalid`（403）、`team_not_found`（404）、`team_state_conflict`（409）。SSE 在心跳及更新时复查访问权限；团队管理权限收回后发送 `SUBMISSION_ACCESS_REVOKED` 并结束流，考试限制仍使用 `EXAM_SCOPE_REQUIRED`。

## 8. `/api` 兼容入口的下线门槛

本轮不删除、不重定向 `/api`，两套入口仍由同一注册函数生成，并有等价测试。响应头 `X-API-Version` 标识 `legacy` 或 `v1`。`api_usage_first` 与按路由聚合的 `api_usage` 日志只记录版本、方法和路由模板，不记录查询参数或邀请码；计数随进程重启清零，窗口末尾在下一次请求时输出，不能作为精确计费指标或零调用的唯一证据。

后续先收集所有应用实例及代理日志，排除监控探针，确认外部客户端归属；至少连续 30 天没有业务旧入口调用、覆盖完整考试周期且通知集成方之后，才能提出独立下线变更。下线日期、弃用响应头及回滚窗口由维护者确认后实施。Web 切换到 `/api/v1` 本身不证明所有旧客户端已迁移。

## 9. 提交列表不含源码

提交列表（`/submissions`、比赛和题单的提交记录、个人资料的最近提交）不再返回 `source_code`，用 `code_length` 表示长度，查看代码须调用 `getSubmission`。`/submissions/latest` 仍返回源码，提交框靠它恢复上次代码。不要用列表响应判断源码可见性。

## 10. 榜单交互沿用接口（2026-10-31）

ICPC 榜单动效与播放器不新增接口。滚榜仍由管理员专用接口返回 v1 封榜／最终快照，客户端逐题揭晓；不开放公开回放或逐提交协议。题格抽屉按需调用现有本场提交或提交列表接口，权限仍由服务端执行，客户端另按用户、场次和题目精确过滤。列表最多最近 200 条，不作为完整审计历史。未揭晓回放题格不加载详情，播放计时器不触发网络刷新。

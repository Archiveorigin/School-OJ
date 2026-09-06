# OpenAPI 契约与 TypeScript Client

## 1. 单一事实源

HTTP 接口契约以 [`apps/api/openapi/openapi.json`](../apps/api/openapi/openapi.json) 为唯一规范源。它包含当前 `/api` 与 `/api/v1` 共用的路径、方法、`operationId`、认证要求、角色限制、请求体和核心响应结构。

API 启动后可通过 `GET /openapi.json` 读取同一份内嵌契约。服务端不维护第二份运行时文档，前端也不手写另一套路由常量。

规范中的路径不带 `/api` 前缀，`servers` 同时声明：

- `/api`：当前稳定入口；
- `/api/v1`：版本化兼容入口。

## 2. 生成 TypeScript Client

前端生成器位于 [`apps/web/scripts/generate-openapi-client.mjs`](../apps/web/scripts/generate-openapi-client.mjs)，输出文件为 [`apps/web/src/api/generated/client.ts`](../apps/web/src/api/generated/client.ts)。生成文件包含：

- OpenAPI `components.schemas` 对应的 TypeScript 类型；
- 全部 `operationId`、HTTP 方法和路径表；
- 自动替换路径参数、传递查询参数和请求体的 `requestOperation`；
- 按 `operationId` 调用的 `generatedApi`；
- 现有页面过渡期使用的 Axios 兼容导出 `client`。

修改契约后执行：

```bash
cd apps/web
npm run api:generate
```

新代码优先使用生成方法：

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
- 修改规范后没有重新生成 TypeScript Client。

## 4. 修改接口的顺序

1. 先修改 `apps/api/openapi/openapi.json` 的路径、请求和响应结构。
2. 修改 Handler、模型及 `handlers/routes.go` 使实现符合契约。
3. 运行 `npm run api:generate` 更新生成客户端。
4. 前端通过 `generatedApi` 使用新增或变更后的操作。
5. 运行 Web 构建、API/Worker 单测以及相关 Playwright 流程。

旧客户端兼容期间，页面仍可从 `api/client.ts` 导入 `client` 使用 Axios 风格调用，但底层实例已由生成模块统一导出。新增接口不得再创建独立路径常量或复制响应类型；可复用的响应结构应补充到 OpenAPI `components.schemas`。

## 5. 扩展字段

规范使用少量 `x-*` 字段记录实现约束：

| 字段 | 作用 |
| --- | --- |
| `x-handler` | 对应的 Go Handler 名称，便于代码定位。 |
| `x-auth` | `public`、`optional` 或 `bearer`，标记认证方式。 |
| `x-roles` | 需要的基础角色列表。 |

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

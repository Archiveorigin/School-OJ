# HTTP 场次组题 UUID 兼容修复

## 原因与修复

生产站点使用普通 HTTP。Web Crypto 的 `randomUUID()` 带有 `SecureContext` 限制，普通 HTTP 域名/IP 下可能不提供该方法。`localhost` 和回环地址被视为可信来源，因此之前的本地与 SSH 回环地址验收未覆盖这一差异。见 [Web Crypto 规范](https://www.w3.org/TR/webcrypto/) 和 [Secure Contexts 规范](https://w3c.github.io/webappsec-secure-contexts/)。

Markdown 单题导入和场次大测试点分片上传都曾直接调用该方法。现在统一使用 `features/uuid.ts`：有原生方法则使用原生方法，否则以 `crypto.getRandomValues()` 的 16 字节安全随机数生成标准 UUID v4，并设置正确的版本和 variant 位。题库/工单测试点上传也复用相同实现。没有修改浏览器全局对象，没有使用弱随机数，没有改变后台上传权限或文件协议。

## 回归验证

- 前端 40 个测试文件、150 项测试通过，包括原生方法、缺少方法、UUID 位格式、Markdown 内容保留、跨分片 ID 一致及跨文件 ID 不同。
- OpenAPI 校验、类型检查、生产构建与首屏包体预算检查通过。
- 新增 `apps/web/tests/e2e/event-http-uuid-qa.mjs`，对真实 API 执行考试和团队比赛的单题导入、手动新增和大于 1 MiB 文件分片上传，覆盖桌面/手机。
- 浏览器必须实测 `isSecureContext === false`、`crypto.randomUUID === undefined`、`getRandomValues` 可用；不使用安全上下文豁免标志或 Crypto mock。

从 `apps/web` 运行：

```powershell
# 先运行 TestEventBrowserFixture，并完成 npm run build。
# 若本机网络软件拦截 .test 域名，可指定本机虚拟网卡的非回环 IPv4。
$env:OJ_HTTP_QA_BIND = '<本机虚拟网卡 IPv4>'
$env:OJ_IMPORT_MARKDOWN_FIXTURE = '<单题 Markdown 完整路径>'
node tests/e2e/event-http-uuid-qa.mjs
```

隔离容器验收可设置 `OJ_HTTP_QA_TARGET=http://127.0.0.1:18091`，该端口必须转发至独立测试 Compose 项目。脚本仅接受该候选环境地址，使用隔离环境账号，不应将它转发至正式站点。输出保存在仓库忽略的 `tmp/uuid-local-browser` 或 `tmp/uuid-candidate-browser`。

本次仅修改前端，不涉及数据库迁移、赛制或权限规则。

## 发布结果

- 应用提交 `32ec639d66b51aabbd894d0e16511e98b792815c`，本机 Git 作者 `TailingMing`。
- 本机生产构建和服务器隔离 Compose 构建，各通过 4 条真实 HTTP 浏览器流程。使用用户的 `二阶矩阵.md`，未将该题面提交到仓库。
- 服务器重建并发布已验收的 web 镜像，API、worker、PostgreSQL、Redis、MinIO 容器未重建；数据库无迁移，生产 `.env` 未变化。
- 正式域名 HTTP 地址的桌面/手机登录页、登录保护、健康检查通过，浏览器无运行异常；生产运行镜像 ID 与隔离验收镜像一致。
- 回退镜像：`school-oj-web:rollback-uuid-19c8818`；发布证据：`/opt/School-OJ-backups/uuid-20261021/`。

# 复用组件性能与生命周期

## 范围与结论

审查范围为 `apps/web/src/components` 下全部 33 个 Vue 组件，保持现有 Vue 3、Element Plus、Monaco 与 operationId API 架构。优先消除重复工作、同步存储、无效请求和资源泄漏；简单展示组件不引入额外缓存、深比较或组件层级。性能没有脱离数据量与设备的绝对“最佳值”，本轮结论只覆盖下述实测与机制。

原始分析需要校正：全局样式原本已为 Monaco 设置 420px 高度；默认主题是浅色；异步导入后创建编辑器读取的是最新 props，不会天然丢失等待期间的更新；动态 import 仍有主线程解析执行成本；`getLanguages()` 不能判断格式化 provider 是否存在。

## 全组件审查

| 组件 | 结论与处理 |
| --- | --- |
| CodeEditor | 显式拥有并释放 model、editor、内容监听和主题观察器；支持加载占位和失败后的文本输入；使用缓存源码避免 v-model 回传时重复全量读取；仍即时 emit，避免提交旧代码。 |
| SubmissionComposer | 300ms 内合并草稿存储写入，持续输入也定期落盘；切题、切语言、提交、页面隐藏、pagehide、卸载时刷新；清空草稿取消待写入内容；存储额度错误不阻塞输入。 |
| SourceCodeViewer | 缓存行号计算；复制原始源码；按钮固定在代码区域右上角，桌面悬停和键盘可见焦点显示，移出隐藏，触屏设备常显。 |
| MarkdownRenderer | 全实例共用无实例状态的解析器；每次源码变化只解析一次，渲染与图片发现共用 tokens；公式采用有限容量缓存；图片路径不变时不重复请求，外部 URL 不重复加载；并发上限 4；作用域变化、卸载取消请求并释放包括部分完成的 Blob URL；清理复制反馈定时器。 |
| ProblemTestDownloads | 仅打开弹窗请求，关闭、切题、卸载时取消请求；最多 3 个测试点 / 6 个文件同时读取；固定请求的题目 ID，阻止旧响应覆盖当前题目；只读响应使用 shallowRef；下载也可取消。 |
| SubmissionDetailDialog | 关闭后销毁详情 DOM，释放大源码展示树；重新打开仍使用当前详情。 |
| SubmissionDetailContent | 摘要已有 computed，源码隔离在专用组件；保持。详情请求组合函数另改 shallowRef、取消旧请求，保留序号防护。 |
| ScoreLeaderboardTable、PenaltyLeaderboardTable | 表头统计集中在 computed 中，数据变化时才计算；横向滚动只修改表头 transform，不触发整张结果表响应式重渲染；缺省结果复用对象。 |
| LeaderboardBoard | 排序、过滤、名次、奖项已有 computed；保持全数据统计与筛选语义。不直接加入虚拟列表，避免改变原生查找、全屏和表格访问方式。 |
| TeamIconUpload | 限制并发读取、卸载时中止 FileReader，捕获读取错误，避免卸载后回写。保持现有 2MB 限制与数据协议。 |
| ProblemStatementView、EventProblemPanel | 题面和内容区复用保持；清理此前已取消题目标题留下的无效 showTitle 参数和 CSS。 |
| ProblemMetaCard | 标签、难度、编号已有 computed；保持。 |
| ProblemOverview、ProblemSwitcher、EventProblemList | 以题目 ID 为 key，轻量列表与原生布局；保持。大题单瓶颈应结合分页数据量定位。 |
| ProblemTagSelector | 固定规模标签、弹窗按需销毁、过滤已有 computed；保持，未增加收益有限的缓存。 |
| AppSidebar | 权限导航已有 computed，低频更新；保持。 |
| ListPagination | 仅转发分页状态，无额外请求与重计算；保持。 |
| VerdictTag、StatusBadge、SubmissionStatusMark | 轻量状态映射已有 computed；保持不同显示语义，不为微小映射创建更多组件。 |
| CourseJoinDialogs | 用户操作才发送加入请求，loading 提供反馈；保持。 |
| EventWorkspaceHeader | 纯展示与路由链接；保持。 |
| TeamContestAwardFields | 三项比例与校验已有 computed；保持。 |
| ExamBasicInfoStep、ExamScoreSummary | 表单派生值与得分比例已有 computed；保持。 |
| ExamProblemSelectionStep | 已有分页、computed 过滤和 Set 选中索引；保持。 |
| ExamPublishReviewStep、ExamWizardHeader | 低频摘要与向导控制；保持。 |
| AdminPageHeader、AdminSectionHeader | 纯布局、标题与插槽；保持。 |

## 编辑器格式化边界

Monaco 仅加载现有 C/C++、Java、Python 语法定义，并通过 Vite `?worker` 提供编辑器 Worker。不引入新的语言服务或自动诊断承诺。关闭缺少专用 provider 的符号出现位置高亮，避免跨模型扫描及 Monaco 0.52 快速卸载时延迟高亮任务的未捕获取消异常；语法着色、文本选中、查找、补全等功能保留。

优先检查格式化 action 的 `isSupported()`。已安装的 formatter 即使没有改动文本，也不会再次执行降级逻辑。当前未配置这些语言的专业 formatter：C/C++/Java 降级仅调整已有行的缩进，不重写运算符、注释、字符串，不拆分单行程序；原始字符串、Java 文本块、转义续行等复杂文档保留原文。Python 缩进具有语义，缺少真实 formatter 时保留原文并提示。格式化编辑通过 executeEdits 与撤销边界执行，保留视图状态；不再使用 setValue 清空撤销栈。

这是对原有不安全“自动格式化”的明确收敛，不等同于 clang-format、Black 或 Java formatter。后续若要求完整重排，应单独集成对应解析器/格式化引擎，评估包体、Worker、超时与语言版本。

## 验证与可量化变化

- 同一工作区、同一依赖和构建配置下，Monaco 主 JS 从 3,914.27 kB 降为 3,318.11 kB（约 15.2%）；gzip 从 993.73 kB 降为 853.64 kB（约 14.1%）。另输出 232.35 kB 的独立 Worker，不能把主包减少量当成总下载减少量。构建耗时受本机状态影响，不作为稳定收益。
- 每次按键仍同步源码；草稿存储在 300ms 窗口内合并。测试断言连续三次编辑即时发出三次更新而只写入一次 localStorage。
- Markdown 文本编辑且图片路径不变，不重新请求图片；测试覆盖外部 URL、作用域隔离和卸载前部分请求完成时的资源回收。
- 未打开测试数据弹窗时请求数为零；测试覆盖切题后旧请求不会继续下载或覆盖新题目。
- 117 项 Web 测试通过（`npm test -- --run --maxWorkers=2`）；API、Worker 测试使用隔离 PostgreSQL/Redis。浏览器验证包括桌面 1440×1100、手机 390×844 的考试/准入 68 项、详情/治理 18 项，以及编辑器与复制按钮专项检查。专项包含原文复制、触屏/键盘显示、格式化撤销、语言切换、深色主题、五次重挂载后的模型数量。

## 后续性能边界

全量榜单 DOM、超大源码行号和测试文件全文仍随数据量增长；未人为截断展示、修改分页协议或省略内容。若真实负载出现长任务，应记录设备、行数、文件字节数与交互耗时，再选择榜单虚拟化、服务端测试文件预览范围或后台格式化。Element Plus 全量注册已在下述优化中移除，KaTeX 和 Monaco 保持按需加载。

设计参考：[Vue 性能指南](https://vuejs.org/guide/best-practices/performance.html)、[Monaco ESM 与 Vite Worker 集成](https://github.com/microsoft/monaco-editor/blob/main/docs/integrate-esm.md)。实现以仓库当前安装版本及浏览器验证为准。

## 网站性能与安全联合复核（2026-10-10）

基线为服务器已发布源码 `a8fdc6a`。使用现有 Vue 3/Vite、Gin/GORM/PostgreSQL、Redis 和 Nginx 技术完成以下优化；附件上传、题目版本及权限治理沿用线上最新实现。

| 定位 | 修改与验证 |
| --- | --- |
| 首屏全量加载 Element Plus | 模板组件、指令、消息与确认框样式按需导入，移除整包注册和整套 CSS；独立 runtime 分包防止共享 helper 将 Monaco 提前拉入首屏。桌面与手机实际镜像交互复核。 |
| 题库统计 N+1 | 每题两次 COUNT 合为当前页 ID 的一次聚合。真实 PostgreSQL 的 20 题回归：匿名列表查询从 43 次降至 4 次，六种判题状态的统计和通过率口径一致。 |
| 静态资源压缩与缓存 | 构建预生成 gzip；成功的哈希资源使用一年 immutable，404 不长期缓存；HTML 继续 no-cache。浏览器验证压缩、缓存及安全头继承。 |
| API 防护 | 默认 private/no-store；限制页码、标签数、关键词长度；题库数据库错误不向客户端回显；查询随请求取消。JWT 必须带期限和非零用户 ID，限制令牌长度。HTTP 头读取和空闲连接设置超时，保留 SSE。 |
| 依赖及构建漂移 | Go 1.26.9 与安全依赖升级；npm 锁文件修复；镜像使用 npm ci、Go readonly module 构建，Nginx 固定已验证镜像摘要。npm audit 原有 10 项（High 6）降至 0；API 可达 Go 漏洞 20 降至 0。 |

以下字节数来自相同 Chromium、全新浏览器上下文的桌面/手机加载。基线为旧生产镜像，新值为同机独立候选镜像；网络路径不同，不将两者的 LCP/TTFB 差值作为提速结论。单位 KB 为十进制，压缩体积采用 Resource Timing encodedBodySize。

| 页面 | JS 解码大小：原 → 新 | JS 压缩体积：原 → 新 |
| --- | --- | --- |
| 首页 | 1,284 KB → 521 KB | 429 KB → 181 KB（减少约 58%） |
| 登录 | 1,280 KB → 465 KB | 427 KB → 161 KB（减少约 62%） |
| 题库 | 1,294 KB → 626 KB | 434 KB → 221 KB（减少约 49%） |

构建入口的静态 JS 依赖合计 413,421 字节，gzip 142,978 字节；新增 250 KiB gzip 构建预算，发现 Monaco/KaTeX 被提前引入即失败。更多小请求替代整包，收益主要为减少下载与解析。候选与线上页面 CLS 均低于 0.062，无整页横向溢出；这些实验室记录不等于真实用户 P75 INP/FPS 或 Lighthouse 评分。原 Go 扫描采用本机原工具链 1.26.5，20 项为源码可达分析，不是旧生产 Go 1.24 二进制的完整漏洞数量；新构建与复扫统一为 1.26.9。线上最终复核及发布过程见 [部署记录](deployment.md#45-性能与安全优化发布2026-10-10)。

验收：API 154 项通过，两个驻留浏览器 fixture 在常规测试中跳过，其中工单 fixture 单独启动；Worker 36 项通过，真实 Docker 沙箱正常执行/CPU 超时/OOM/运行错误另行通过；Web 34 文件、133 项通过；200 项接口契约、类型检查及生产构建通过。公开页面两种视口共 6 项检查；工单 33 项、导入/团队 17 项、审核 14 项在隔离 PostgreSQL 16、Redis 7 和实际 Web 镜像下全部通过。服务端证据在 `/opt/School-OJ-backups/performance-audit-20261010/`，本地截图与 JSON 在忽略提交的 `tmp/performance-*`、`tmp/ticket-*-qa/`。

安全边界见 [安全设计](security.md)。全量旧版 `/problems` 接口、全量榜单和 900 KB 原始 PNG 校徽仍有后续优化空间，需保持计数和显示语义再分页/裁剪资源。本轮未进行高并发压力、堆内存持续增长、完整 ZAP 或容器操作系统 CVE 扫描，不宣称零安全风险。

依据：[Element Plus 按需导入](https://element-plus.org/en-US/guide/quickstart)、[Nginx gzip_static](https://nginx.org/en/docs/http/ngx_http_gzip_static_module.html)、[Nginx 响应头继承](https://nginx.org/en/docs/http/ngx_http_headers_module.html)、[Go 安全更新](https://pkg.go.dev/vuln/GO-2026-6617)。

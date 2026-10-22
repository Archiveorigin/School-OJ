# 前端组件与交互规范

Vue、Element Plus、Monaco 和生成的 operationId 客户端继续复用。页面按现有模块修改，避免仅为视觉调整引入第二套状态或请求系统。历史性能数字属于特定测量环境，集中见 [开发历史](development-history.md)，不能用旧数字宣称本次更快。

## 1. 主题与图标

全局主题从 `styles/tokens.css` 的语义变量取得画布、表面、文字、边框、状态色和阴影。Element Plus 的背景／填充／边框变量与同一主题相连，teleport 到 body 的对话框、日期面板和下拉列表也随暗色模式更新。代码查看器同步暗色主题。

排行榜默认“随页面”，保留独立深色与马卡龙主题。默认题目格使用主题表面，AC、错误、封榜使用对应状态色；奖项只作用于排名与姓名。新增图标使用现有图标组件并保持对齐，不用文字字符模拟工具图标。

学校及团队 Logo 使用居中的 `object-fit: contain` 和内边距，避免拉伸或圆形裁掉徽章内容；可点击上传容器另为底部说明预留空间。头像的圆形语义与机构 Logo 分开处理。

Vue scoped 样式中全局主题与局部组件组合使用 `.dark .component` 等可正确编译的选择器，避免 `:global(.dark)` 与局部选择器组合后丢失组件限制。修改后通过实际浏览器计算样式和截图检查。

## 2. 编辑、请求与生命周期

设置表单使用 `features/events/editSnapshot.ts` 管理已保存基线、脏状态和内部 revision。只在真正打开新资源或成功保存后重置基线；后台刷新不能覆盖脏字段。并发服务器修改返回冲突，保留原输入。textarea 按内容自动增高并允许手动调整。

切换路由／资源时取消旧请求，并用请求序号拒绝迟到响应；关闭或卸载时释放定时器、监听、Blob URL 与编辑器模型。排行榜只在进入或手动刷新时加载；考试生命周期检查和判题 SSE 各自保留职责。

HTTP 来源上的上传 ID 统一使用 `features/uuid.ts`：优先原生 `crypto.randomUUID()`，不可用时用 `crypto.getRandomValues()` 生成正确版本及 variant 位的 UUID v4，不使用 Math.random。验证必须包含真实非安全 HTTP 来源，不能只用被浏览器视为可信的 localhost。

## 3. 组件职责与资源释放


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
| TeamContestAwardFields | 比例／人数单位、累计边界与六位小数校验共用纯函数，切换单位不修改比例。 |
| ExamBasicInfoStep、ExamScoreSummary | 表单派生值与得分比例已有 computed；保持。 |
| ExamProblemSelectionStep | 已有分页、computed 过滤和 Set 选中索引；保持。 |
| ExamPublishReviewStep、ExamWizardHeader | 低频摘要与向导控制；保持。 |
| AdminPageHeader、AdminSectionHeader | 纯布局、标题与插槽；保持。 |

## 编辑器格式化边界

Monaco 仅加载现有 C/C++、Java、Python 语法定义，并通过 Vite `?worker` 提供编辑器 Worker。不引入新的语言服务或自动诊断承诺。关闭缺少专用 provider 的符号出现位置高亮，避免跨模型扫描及 Monaco 0.52 快速卸载时延迟高亮任务的未捕获取消异常；语法着色、文本选中、查找、补全等功能保留。

优先检查格式化 action 的 `isSupported()`。已安装的 formatter 即使没有改动文本，也不会再次执行降级逻辑。当前未配置这些语言的专业 formatter：C/C++/Java 降级仅调整已有行的缩进，不重写运算符、注释、字符串，不拆分单行程序；原始字符串、Java 文本块、转义续行等复杂文档保留原文。Python 缩进具有语义，缺少真实 formatter 时保留原文并提示。格式化编辑通过 executeEdits 与撤销边界执行，保留视图状态；不再使用 setValue 清空撤销栈。

这是对原有不安全“自动格式化”的明确收敛，不等同于 clang-format、Black 或 Java formatter。后续若要求完整重排，应单独集成对应解析器/格式化引擎，评估包体、Worker、超时与语言版本。


## 4. 性能与验证边界

Element Plus 按需导入，Monaco 与 KaTeX 保持懒加载；构建检查入口 gzip 预算（250 KiB），哈希静态资源预生成 gzip 并长期缓存，HTML 保持 no-cache。颜色存量预算是具体字面值债务，新增样式使用语义变量。

榜单全量 DOM、超大源码和文件仍随数据规模增长。优化前记录设备、数据量、请求数及耗时，再决定分页、虚拟化或后台处理，不能静默截断结果。小样本 TTFB 不作为 SLA，主包缩小不等于全部传输缩小。

界面验证覆盖 1440×1100 与 390×844、浅暗主题、键盘／触屏、横向溢出、请求失败和控制台异常。弹窗等待主题与过渡结束再截图；页面截图不能替代实际保存、权限拒绝和服务器数据校验。通用测试命令见 [部署与测试](deployment.md)。

# 工单化题目管理与三赛制设计 QA

## 对比目标

- 参考视觉真值：
  - `luogu-problem-list.png`
  - `luogu-contest-problems.png`
  - `luogu-contest-problem.png`
  - `luogu-ioi-scoreboard.png`
  - `luogu-acm-scoreboard.png`
- 参考截图目录：`C:\Users\intimifeng\.codex\visualizations\2026\08\30\01a052a2-1037-7573-843a-97b4b8809958`
- 参考截图像素：以上五张均为 1440 × 1100。
- 实现路径：`/problems`、`/problem-changes/new`、`/admin/problem-authors`、`/contest/:id/description`、`/contest/:id/problems`、`/contest/:id/problems/:label`、`/contest/:id/scoreboard`。
- 桌面 CSS 视口：1440 × 1100，device scale factor 1。
- 移动 CSS 视口：390 × 844，device scale factor 1；使用 full-page 截图验证全部内容，题库输出为 390 × 1853，排行榜输出为 390 × 1020。
- 状态：固定的管理员身份与确定性 API 模拟数据，覆盖题库筛选、作者发单、管理员处理、比赛题目、OI、IOI、ACM 和封榜。
- 密度归一化：参考图与桌面实现均以原生 1440 × 1100 渲染；对比图以等宽双栏并排输入进行目视检查。

## 渲染证据

- 题库桌面：`qa-problem-catalog-desktop.png`
- 题库移动：`qa-problem-catalog-mobile.png`
- 作者发起工单：`qa-ticket-author-desktop.png`
- 管理员工单抽屉：`qa-ticket-admin-desktop.png`
- 比赛题目列表：`qa-contest-problems-desktop.png`
- 比赛题目页：`qa-contest-problem-desktop.png`
- IOI 排行榜：`qa-ioi-scoreboard-desktop.png`
- ACM 排行榜：`qa-acm-scoreboard-desktop.png`
- 移动排行榜：`qa-scoreboard-mobile.png`
- OI 比赛说明：`qa-oi-description-desktop.png`
- 旧地址兼容跳转：`qa-legacy-route-desktop.png`

## 同屏对比证据

- 题库：`qa-compare-problem-catalog.png`
- 比赛题目列表：`qa-compare-contest-problems.png`
- 比赛题目页：`qa-compare-contest-problem.png`
- IOI 排行榜：`qa-compare-ioi-scoreboard.png`
- ACM 排行榜：`qa-compare-acm-scoreboard.png`

以上证据均位于参考截图目录，已作为相同对比输入逐张检查。

## 结论

- 没有剩余可执行的 P0、P1 或 P2 视觉问题。
- 实现沿用黄海在线测题平台的校徽、品牌蓝、第三版后台设计语言和 Element Plus 图标；仅借鉴参考站点的信息架构、列表密度、标签导航和矩阵榜单，不复制其素材或品牌。
- 题库保持“筛选卡 + 密集表格”的桌面结构，移动端转为可读题目卡；个人状态、题号、标签、难度、通过率、分页和工单入口均可见。
- 比赛保持“比赛说明 / 题目列表 / 排行榜”三段导航；OI 不渲染排行榜入口，IOI 与 ACM 使用对应矩阵单元格，封榜状态有明确提示。
- 题目页保留 A/B/C/D 快速导航、题面、提交入口和本题记录，重复标题已移除。
- 作者与管理员上传控件已本地化；管理员抽屉在动画完成后截图，影响场次、最终题包、驳回和执行操作均完整可见。

## 必需保真面

- 字体与层级：标题使用现有品牌字重和蓝色英文 eyebrow，正文与表格使用系统中文字体；题库、比赛标题、榜单总分和状态的层级稳定。
- 间距与布局：桌面内容宽度、筛选密度、题表行高和榜单矩阵接近参考节奏；移动端没有页面级横向溢出，榜单矩阵在独立容器内横向滚动并固定名次、参赛者和汇总列。
- 色彩：延续深蓝、亮蓝、白色卡面和浅灰蓝背景；AC、尝试、封榜和危险操作分别使用一致的绿、橙、琥珀和红色语义。
- 图片与图标：使用仓库内真实校徽及 Element Plus 图标；没有用 emoji、手绘 SVG、CSS 图形或占位图片伪造可见素材。
- 文案：所有入口和说明均映射真实工单、题库、比赛和计分业务；上传按钮、封榜提示和赛制名称均为中文且无浏览器原生英文残留。
- 响应式：390 px 下题库筛选纵向排列、题目卡完整显示；榜单通过横向滚动保留高密度信息，固定列在首屏可读。

## Playwright 验证

- 题库桌面筛选与移动卡片渲染。
- 作者替换工单的预填目标和表单控件。
- 管理员工单详情、影响范围与完整 ZIP 题包操作。
- 比赛说明、题目列表、题目详情与提交入口。
- OI 无排行榜入口。
- IOI、ACM 矩阵榜单与封榜提示。
- 390 px 榜单横向滚动能力。
- 旧 `#problems` 地址兼容到真实子路由。
- 浏览器 console error：0；uncaught page error：0。

## 对比迭代记录

1. 第一轮：五组同屏对比确认整体信息架构和密度方向正确，发现三个 P2 问题——比赛题目页重复题名、工单上传显示英文原生控件、管理抽屉截图处于过渡动画中。
2. 修正：题面组件关闭重复标题；作者和管理员上传统一为本地化 Element Plus 控件；截图前等待抽屉动画稳定。
3. 第二轮：重新运行全部 11 个 Playwright 场景并生成五组同屏对比，逐张复核桌面和移动证据；前述问题均已消除，无新增 P0/P1/P2。

## 开放问题

- 本轮设计验收无开放问题。

final result: passed

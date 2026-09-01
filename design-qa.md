# 本轮设计 QA：工单、统一排行榜与权限管理

## 验收目标

- 工单附件改为必填，并明确上传“修改后的正确内容”；替换、删除采用覆盖性执行。
- 团队比赛不再维护独立排行榜视觉，OI/IOI/ACM 复用考试排行榜组件。
- 题库算法标签恢复弹出式选择器。
- 后台“教学概览”保持简洁；出题者作为独立附加权限，与学生、教师、管理员基础角色并行。
- “用户与权限”增加独立权限管理子路由，工单管理只处理题目数据工单。
- 桌面视口为 1440 × 1100；移动视口为 390 × 844，截图均使用 full-page。

## 自动化与渲染证据

最终证据目录：

`C:\Users\intimifeng\.codex\visualizations\2026\08\30\01a052a2-1037-7573-843a-97b4b8809958\update-107c36f-final`

主要截图：

- 题库桌面与移动：`qa-problem-catalog-desktop.png`、`qa-problem-catalog-mobile.png`
- 算法标签弹窗：`qa-problem-tag-popup-desktop.png`
- 出题人工单与管理员处理：`qa-ticket-author-desktop.png`、`qa-ticket-admin-desktop.png`
- 教学概览桌面与移动：`qa-admin-overview-desktop.png`、`qa-admin-overview-mobile.png`
- 权限管理桌面与移动：`qa-permission-management-desktop.png`、`qa-permission-management-mobile.png`
- IOI、ACM 与移动排行榜：`qa-ioi-scoreboard-desktop.png`、`qa-acm-scoreboard-desktop.png`、`qa-scoreboard-mobile.png`
- OI 无排行榜与旧路由兼容：`qa-oi-description-desktop.png`、`qa-legacy-route-desktop.png`

参考站点的同视口信息架构对比继续覆盖题库、比赛题目列表和比赛题目页：

- `qa-compare-problem-catalog.png`
- `qa-compare-contest-problems.png`
- `qa-compare-contest-problem.png`

排行榜本轮以现有考试排行榜为唯一视觉真值，因此不再与外部站点排行榜截图比较。

## 目视审计

- 算法标签以居中弹窗展示，算法、时间、来源分栏明确；标签可搜索、可多选，没有回退为下拉列表。
- 工单页把附件和覆盖方式放在主要表单流程中；管理员抽屉同时展示申请附件、影响场次、最终完整题包及覆盖执行按钮。
- 教学概览在桌面使用四项核心指标、近期考试、待办、快捷入口和最近动态；移动端顺序降级完整，没有横向裁切。
- 权限管理明确区分基础角色和“出题者”附加权限。首次移动审计发现桌面表格在窄屏截断操作列，已改为移动权限卡片并复测，授予、收回操作均可见。
- 团队比赛 IOI/ACM 榜单复用考试排行榜的标题、工具栏、状态单元格和奖牌语义；移动端矩阵在独立容器内横向滚动。OI 不显示排行榜入口。
- 未发现剩余 P0、P1 或 P2 视觉及核心交互问题。

## Playwright 结果

- 题库筛选、移动卡片和算法标签弹窗：通过。
- 工单必填附件、覆盖操作与管理员处理抽屉：通过。
- 教学概览和权限管理桌面/移动：通过。
- 比赛题目列表、题目详情、IOI/ACM 共用排行榜、封榜、OI 隐藏榜单：通过。
- 390 px 排行榜横向滚动和权限操作可见性：通过。
- 旧 `#problems` 路由兼容：通过。
- 浏览器 console error：0；uncaught page error：0。

final result: passed

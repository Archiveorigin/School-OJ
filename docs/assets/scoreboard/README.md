# 榜单文档配图

本目录只服务于 [ICPC/CCPC 榜单与滚榜动画](../../scoreboard.md)，不是应用运行时资源。

| 文件 | 来源/用途 |
| --- | --- |
| `xcpcio-nanjing-1440.png` | 第 49 届 ICPC 南京站公开榜单截图，来源 <https://board.xcpcio.com/icpc/49th/nanjing>，仅作视觉参考，版权归原站。 |
| `xcpcio-cells-2x.png` | 同一截图裁切并放大，用于说明题格的“尝试数/通过分钟”写法。 |
| `cell-states.svg` / `.png` | 按现有 `PenaltyLeaderboardTable.vue` 的颜色、尺寸绘制的题格状态与交互反馈图；SVG 可编辑，PNG 用于 Markdown 直接展示。 |
| `rollcall-storyboard.png` | 滚榜三帧目标效果（封榜 → 揭晓 → 落位）；由本地静态 HTML 渲染，不是生产页面。 |
| `flip-phases.svg` / `.png` | FLIP 四步示意；SVG 可编辑，PNG 用于 Markdown。 |

约定：

1. 截图与示意图只放在 `docs/assets/`，不要放进 `apps/web/public`，避免被打进前端包体和生产静态资源。
2. 新增/替换图片时同步更新 `docs/scoreboard.md` 的图注与 `docs/index.md` 登记。
3. 第三方截图必须保留来源链接与“仅作视觉参考”的说明；不要裁剪掉来源信息后声称是自制。
4. SVG 优先保留，便于后续修改颜色/尺寸；PNG 用浏览器或矢量工具重新导出即可。

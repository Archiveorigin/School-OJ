# Ranking and submission UI topology

References used for this update:

- Penalty scoreboard extraction: `YOUR-NEW-REPOSITORY/docs/research/` and its desktop/tablet/mobile screenshots.
- Score scoreboard reference: `workspace/榜单设计.png` (1951 × 227 px crop).
- Submission detail reference: `workspace/提交代码详情弹窗查看.png` (1220 × 691 px).
- Obsolete inline result reference: `workspace/题库题目提交小组件.png` (1624 × 96 px).

## Scoreboard flow

1. Shared title and control area: title, ranking display mode, time slider, participant filter, refresh, summary, theme, and fullscreen controls.
2. Scoring-rule router: `penalty` renders the penalty table; `score` renders the score table. The rule is fixed by the exam/contest creation choice and is not a viewer-side toggle.
3. Sticky scoreboard header: student identity, ranking metrics, and one column per problem. The sticky viewport is outside all horizontal overflow ancestors.
4. Horizontally scrollable rows: each scoring branch owns a body-only horizontal scroller and mirrors its `scrollLeft` onto the header track. Student name and student number replace all team, coach, player, school, class, and organization identity from the source pages.
5. Footer legend and last-updated time.
6. Summary overlay.

The scoreboard document keeps a fixed minimum table width and scrolls horizontally on narrow screens. The page scrolls vertically at its natural height; there is no 650 px internal vertical viewport. Horizontal overflow belongs only to the row-body scroller, while its sibling header stays sticky and follows the body scroller's horizontal offset.

## Submission flow

1. The public problem detail page has only the explicit “提交记录” and “提交代码” actions after a submission. The old full-width result strip below the statement is removed.
2. The submission records page opens a reusable submission-code dialog.
3. The dialog shows four summary cards, an SVG verdict graphic, a source-code panel, and a copy action.
4. The old compile/runtime message alert below the source panel is intentionally absent.

# SubmissionCodeDialog specification

## Overview

- Target: reusable submission source dialog used by problem submission records (and equivalent source-view dialogs where the same data is available).
- Screenshot: `workspace/提交代码详情弹窗查看.png` (1220 × 691 px).
- Interaction model: dialog open/close and click-to-copy.

## DOM structure

Element Plus dialog titled `提交代码` > four-card summary grid > source heading with copy button > scrollable source code panel. No compile/runtime message alert follows the source panel.

## Reference geometry and styles

- Reference dialog: approximately 1220 × 683 px, white background, small corner radius.
- Title: 26 px regular text with 24 px left/top spacing.
- Summary grid: four equal columns with 14–15 px gaps and 24 px horizontal margins.
- Summary card: approximately 282 × 93 px, 1 px pale-blue border (`#d5deec`), 11 px radius, 16 px padding.
- Card label: muted blue-gray, 14 px; card value: dark gray, 20 px, weight 700.
- Status value is an inline SVG verdict graphic, not an Element Plus tag.
- Source heading: 20 px/700; copy button aligned right.
- Source panel: dark navy `#0f172a`, near-white `#e2e8f0`, 12 px radius, 16–22 px padding, monospace text, minimum height 260 px and maximum height constrained by viewport.

## Verdict states

- Accepted: green SVG pill with white `Accepted`.
- Queued/running: blue SVG pill with white Chinese pending label.
- Any other completed result: red SVG pill with white `Unaccepted`.
- SVG wrapper exposes an accessible Chinese status label.

## Removed elements

- Do not render `submission.message` beneath the source panel.
- Remove the full-width post-submit status strip from the public problem detail page.

## Responsive behavior

- Below 760 px, summary cards become one column and the dialog remains within 14 px of viewport edges.
- Source content scrolls horizontally without wrapping and vertically within the viewport.

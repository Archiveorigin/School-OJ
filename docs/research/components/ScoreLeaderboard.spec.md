# ScoreLeaderboard specification

## Overview

- Target: score branch of `apps/web/src/components/LeaderboardBoard.vue`.
- Screenshot: `workspace/榜单设计.png` (1951 × 227 px crop).
- Interaction model: input/click-driven shared controls; scroll-driven sticky table header.

## DOM structure

Shared scoreboard shell > score table > sticky header viewport + sibling horizontal body scroller > repeated score rows. The body scroller synchronizes its `scrollLeft` to the header track, while the sticky viewport remains on the natural page-scroll chain. The header contains the student identity label, rank, full-score problem count, total score, and problem headers. Each row uses the same columns and a student-only identity block.

## Reference geometry and styles

- Reference crop: 1951 × 227 px.
- Header band: approximately 79 px high, light gray `#f7f8fa`, with an 8 px bottom separator.
- Student identity region: approximately 625 px wide in the reference; production uses 560 px on desktop to retain the reference proportions.
- Row: approximately 118 px high, white background, blue 1 px active border in the captured state.
- Rank cell: approximately 86 × 44 px; full-score cell approximately 96 × 44 px; total-score cell approximately 122 × 44 px.
- Problem score cells: at least 104 × 44 px with 7–8 px gaps.
- Header cards: white, `#d9e0e8` border, 6 px radius; problem headers use a 4 px colored top bar.
- Student avatar: 70 px circle, very low-contrast blue/gray initial.
- Student metadata: muted 13 px submission count; main identity: bold student number and bold name separated by a blue-gray diamond.
- Metric text: 16–18 px, weight 800–900.
- Full score: green `#198754`; partial/non-full score: red `#dc3545`; no submission: gray `#adb5bd`; pending: blue `#0d6efd`.

## States and content

- Header labels are `排名`, `满分`, and `总分`.
- Total score displays `score/maxScore`.
- Problem headers display problem label and `fullScore/attempted` participant counts.
- Submitted cells display `bestScore/maxScore`; full-score cells add `满分`, partial cells add `未满分`, and pending cells add `待评分`.
- Published mode follows API rank. Performance mode sorts total score descending, full-score count descending, then published rank.

## Student identity customization

- Do not render team, class, organization, school, coach, or player identity.
- Render only student name, student number, and submission-count metadata.

## Responsive behavior

- Preserve the reference’s fixed column proportions and horizontal scrolling.
- At application widths below the reference crop, reduce only the identity region to 500 px at 991.98 px and 420 px at 576 px; never stack result columns.
- Only `.score-body-scroller` owns horizontal overflow. The sticky header is its sibling, and the page does not gain a fixed-height/internal vertical scroll area.

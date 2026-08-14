# PenaltyLeaderboard specification

## Overview

- Target: penalty branch of `apps/web/src/components/LeaderboardBoard.vue`.
- Source screenshot: `YOUR-NEW-REPOSITORY/docs/design-references/original-desktop-full.png`.
- Source extraction: `YOUR-NEW-REPOSITORY/docs/research/components/ScoreboardTable.spec.md` and `ScoreboardControls.spec.md`.
- Interaction model: input/click-driven controls; scroll-driven sticky table header.

## DOM structure

Shared scoreboard shell > title > controls > penalty table > sticky header viewport + sibling horizontal body scroller > repeated participant rows > footer legend. The header track mirrors the body scroller's `scrollLeft`, so the header remains attached to natural page scrolling instead of an overflow ancestor. Each row has one full-width participant-name line followed by rank, solved count, penalty, and one result cell per problem. Identity is not a dedicated table column.

## Extracted styles

- Source header row: 1357 × 73 px, sticky `top: 0`, with deep-navy metric/problem cards (`#182235`), white Chinese labels, and smaller English labels.
- Source row: minimum height 85 px with a 1323 × 85 px main area.
- Rows use flat alternating white/light-gray stripes and fine separators. They do not use rounded outer cards, vertical gaps, or per-row drop shadows.
- Metric groups: flex layout with 5 px gaps.
- The header and result grid begin at the same left edge: rank width 64 px; solved width 72 px; penalty width 88 px; problem width 80 px with an 85 px stride.
- Result cell: 80 × 32 px, 4 px radius, bold centered content.
- Accepted: background `#198754`, border `#146c43`, white text.
- Wrong: background `#dc3545`, border `#b02a37`, white text.
- No submission: background `#adb5bd`, border `#e9ecef`, dark text.
- Pending/frozen: background `#0d6efd`, border `#0a58ca`, white text.
- Result hover transition: 150 ms transform, shadow, and saturation.
- Gold, silver, and bronze treatment applies to percentage-based row ranges. Solved cells use a pale-blue fill; penalty cells use a pale-orange fill.

## Student identity customization

- The original logo/team/organization/coach/player content must not render.
- There is no identity header or identity property column.
- The upper line inside each row renders only the participant name.
- Student number, email, avatar, submission metadata, team, class, school, organization, coach, and player identity do not render.
- Search indexes the participant name only.
- A large `/logo1.png` college mark is positioned behind the row at roughly 3–5% opacity with `pointer-events:none`.

## States and content

- Header labels are `排名 / Rank`, `题数 / Solved`, and `罚时 / Penalty`.
- Accepted cells display attempt count and accepted elapsed minutes (`attempts | minutes'`).
- Wrong cells display attempt count and latest elapsed minutes.
- Empty cells display the problem label, as in the extracted source page.
- First accepted submission per problem has the extracted gold star overlay.
- Published mode follows API rank. Performance mode sorts solved descending, penalty ascending, then published rank.

## Responsive behavior

- Desktop keeps the extracted fixed-width row geometry.
- Below 1100 px controls wrap; below 520 px they use the compact control button. Metric/result columns never collapse or stack.
- The table does not reflow at tablet/mobile widths and remains horizontally scrollable.
- The page keeps natural vertical scrolling. The sticky header is outside the horizontal overflow element; only `.penalty-body-scroller` scrolls horizontally, and no fixed-height/internal vertical viewport is introduced.

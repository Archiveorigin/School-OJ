# Live scoreboard awards specification

## Overview

- Target files: `LeaderboardBoard.vue`, both leaderboard table branches, and the team-contest create/edit forms.
- Visual source: `YOUR-NEW-REPOSITORY/index.html` plus its extracted desktop/mobile screenshots.
- Interaction model: awards are derived from the current full ranking; creation/edit inputs are click/input driven.

## Award model

- Three integer percentages: gold, silver, and bronze.
- Defaults are 10%, 10%, and 10%, so the default award area is the top 30% of the actual participant count.
- Each value is in `0..100`; their sum cannot exceed 100.
- Cumulative cutoffs use `ceil(totalParticipants * cumulativePercent / 100)` so small contests still receive visible awards.
- Filtering never changes a participant's award. Performance-mode ranks are calculated against the complete unfiltered ranking.

## Row treatment

- Gold, silver, and bronze are row-level states, not merely colors on ranks 1/2/3.
- Award rows use a restrained left accent and a matching rank badge while preserving verdict-cell contrast.
- The participant identity inside a row contains the name only. Student number, email, avatar, organization, team, coach, and member lists must not render.
- A decorative `/logo1.png` college watermark sits behind each row at very low opacity and never obscures text or captures pointer events.

## Responsive behavior

- Award inputs stack below 680 px and always show their combined percentage.
- Scoreboard rows retain fixed result-column geometry and horizontal scrolling at tablet/mobile widths.

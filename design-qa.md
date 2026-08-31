# Admin Theme Design QA

## Comparison target

- Admin source visual truth: `docs/design-references/admin-management-v3.png`
- Exam source visual truth:
  - `docs/design-references/exam-create-basic-info.png`
  - `docs/design-references/exam-create-problem-selection.png`
  - `docs/design-references/exam-create-publish-review.png`
- Source pixels: 1487 × 1058 for each source image
- Implementation routes: `/admin`, `/admin/exams/new`, and the legacy `/exams/new` redirect
- Desktop CSS viewport: 1440 × 1024, device scale factor 1
- Mobile CSS viewport: 390 × 844, device scale factor 1
- State: administrator, light theme, populated browser draft, current/upcoming/closed exams, pending approvals, and audit activity
- Density normalization: each source was downsampled to 1440 × 1024 with high-quality bicubic interpolation before comparison; implementation screenshots are native 1440 × 1024 pixels

## Rendered implementation evidence

- Admin home: `docs/qa/admin-v3/admin-home-1440x1024.png`
- Exam basic information: `docs/qa/admin-v3/exam-create-basic-1440x1024.png`
- Exam problem selection: `docs/qa/admin-v3/exam-create-problems-1440x1024.png`
- Exam publish confirmation: `docs/qa/admin-v3/exam-create-publish-1440x1024.png`
- Mobile navigation: `docs/qa/admin-v3/admin-home-mobile-390x844.png`
- Playwright result record: `docs/qa/admin-v3/playwright-results.json`

## Full-view comparison evidence

- Admin: `docs/qa/admin-v3/comparison-admin-full.jpg`
- Exam basic information: `docs/qa/admin-v3/comparison-exam-basic-full.jpg`
- Exam problem selection: `docs/qa/admin-v3/comparison-exam-problems-full.jpg`
- Exam publish confirmation: `docs/qa/admin-v3/comparison-exam-publish-full.jpg`

## Focused comparison evidence

- Admin header and hero: `docs/qa/admin-v3/comparison-admin-header.jpg`
- Admin plan and publication panels: `docs/qa/admin-v3/comparison-admin-panels.jpg`
- Exam basic form and overview: `docs/qa/admin-v3/comparison-exam-basic-focus.jpg`
- Exam library, selected problems, and score summary: `docs/qa/admin-v3/comparison-exam-problems-focus.jpg`
- Exam document and publish summary: `docs/qa/admin-v3/comparison-exam-publish-focus.jpg`

## Findings

- No actionable P0, P1, or P2 differences remain.
- The implementation preserves the third design's institutional blue-and-paper visual hierarchy, compact bordered surfaces, serif editorial headings, dense examination plan, restrained red publication action, and paired plan/confirmation layout.
- Product-driven deviations are intentional:
  - The sidebar separates courses, classes, exams, prepared problems, authoring, plagiarism, users, and audit logs because those are independent working modules with different role rules.
  - The existing verified campus photograph is used at low opacity instead of reproducing the generated line-art campus illustration. It remains a real, sharp product asset and keeps the same placement, tonal balance, and academic art direction.
  - The current three-step authoring flow integrates scoring into basic information and problem selection rather than adding an otherwise redundant fourth step; the same required settings and validation remain present.

## Required fidelity surfaces

- Fonts and typography: Chinese serif display headings use `STSong` / `Songti SC` fallbacks; body and small UI text use the existing system stack. The post-fix 60 px desktop hero heading restores the source's optical hierarchy without causing wrapping. Labels, metadata, and table text remain legible and consistently weighted.
- Spacing and layout rhythm: the post-fix hero and section gap align the first content row closely with the source. At 1440 × 1024, the admin document height is 1030 px and all lower management panels are visibly present; the remaining 6 px is border-level vertical overflow rather than hidden content. All desktop and mobile captures have `scrollWidth === clientWidth`.
- Colors and visual tokens: deep institutional blue, paper white, slate text, light blue selected states, green readiness states, amber status labels, and restrained red publish actions map consistently across admin and exam screens. Contrast remains clear in the inspected light theme.
- Image quality and asset fidelity: the supplied school logo and campus photograph are used directly, remain sharp, and are cropped without stretching. No visible image asset is replaced with CSS art, custom SVG art, emoji, or a placeholder.
- Copy and content: labels map to the real course, class, exam, question, approval, user, plagiarism, and audit capabilities. Draft wording distinguishes browser-local state from server-created exams, and publication checks reflect actual validation.
- Icons: Element Plus icons provide a consistent outlined family across navigation, filters, status checks, dates, documents, and actions; inspected icons are aligned and not clipped.
- Accessibility and responsiveness: the mobile menu is 44 × 44 px, accepts keyboard focus with an automatic outline, opens and closes the navigation drawer, and the 390 px capture has no horizontal overflow. Semantic buttons and navigation labels remain reachable and readable.

## Primary interactions verified with Playwright

- Exam status filtering to “待开始” and back to “全部”
- Teaching overview draft action into `/admin/exams/new`
- Browser draft save success state
- Basic information → problem selection → publish confirmation
- Publish confirmation → return for changes
- Legacy `/exams/new?course_id=2#authoring` redirect with query and hash preserved
- Mobile navigation drawer open and close
- Browser console errors: 0
- Uncaught page errors: 0
- Unhandled fixture requests: 0

## Comparison history

1. Pre-browser pass: build and unit tests passed, but visual QA was blocked because no browser-rendered implementation evidence existed.
2. First Playwright pass: combined source/implementation comparisons identified three P2 issues: the admin hero started too low and its title hierarchy was weak; compact lower panels extended below the target first viewport; the mobile menu target measured only 38 × 38 px.
3. Fixes applied: reduced hero padding and section gap, raised the display heading to 60 px, compacted lower-panel headers and rows, reduced bottom whitespace, and enlarged the mobile menu target to 44 × 44 px.
4. Post-fix pass: recaptured all desktop and mobile states, regenerated combined full and focused comparisons, confirmed all lower panels are visible, confirmed no horizontal overflow, verified the 44 × 44 px focused menu control, and reran all primary interactions with zero console or page errors.

## Open questions

- None for the selected third-design implementation.

## Follow-up polish

- P3 only: a future brand-asset pass could commission a dedicated blue campus line illustration while retaining the current real campus photograph as a fallback.

final result: passed

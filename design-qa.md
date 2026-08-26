# Exam Authoring Design QA

**Source visual truth**

- `docs/design-references/exam-create-basic-info.png`
- `docs/design-references/exam-create-problem-selection.png`
- `docs/design-references/exam-create-publish-review.png`

**Implementation**

- Route: `/exams/new`
- Components: `apps/web/src/components/exams/`
- Implementation screenshot: unavailable
- Viewport, pixel dimensions, CSS size, density normalization: unavailable
- State: three-step authoring workflow implemented; browser-rendered states not captured

**Evidence status**

- Full-view comparison: blocked because the current Codex Desktop environment exposes no browser capture/inspection tool for the running authenticated route.
- Focused-region comparison: blocked for the same reason.
- Primary interactions tested in browser: not available.
- Browser console errors checked: not available.
- Source images are present and inspectable, but source-only inspection is not a valid comparison.

**Findings**

- [P1] Rendered fidelity is not yet verified.
  Location: all three `/exams/new` steps.
  Evidence: source references exist, but no browser-rendered implementation screenshot can be captured in this environment.
  Impact: typography, spacing, responsive layout, colors, icon alignment, asset quality, and copy cannot be truthfully approved from code or build output alone.
  Fix: run the app with an authenticated fixture, capture all three steps at the reference viewport, combine each source/capture pair, then perform and iterate on visual comparison.

**Required fidelity surfaces**

- Fonts and typography: blocked pending rendered capture.
- Spacing and layout rhythm: blocked pending rendered capture.
- Colors and visual tokens: blocked pending rendered capture.
- Image quality and asset fidelity: no additional raster assets are required by the UI; icon-library rendering remains blocked pending capture.
- Copy and content: implementation copy exists, visual wrapping and truncation remain blocked pending capture.

**Comparison history**

- No valid comparison iteration has been run. Build and unit-test success are intentionally not counted as visual QA evidence.

**Implementation checklist**

1. Start API and web application with an authenticated exam-authoring user.
2. Capture steps 1–3 at the source viewport and at narrow-screen breakpoints.
3. Check the primary next/back/save/publish interactions and browser console.
4. Compare each source and implementation capture in one combined visual input.
5. Fix all P0/P1/P2 findings and repeat until the report can be changed to `passed`.

final result: blocked

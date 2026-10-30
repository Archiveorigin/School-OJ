export const MOTION = {
  rowMove: 700,
  rowEnter: 180,
  cellReveal: 520,
  number: 620,
  rankBadge: 2400,
  stepGap: 120,
  expand: 320,
} as const;
export const EASE = "cubic-bezier(.2,.7,.2,1)";
export function prefersReducedMotion() {
  return (
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true
  );
}
export function motionScale(speed = 1) {
  return prefersReducedMotion() ? 0 : 1 / Math.max(0.5, Math.min(2, speed));
}
export function stepDuration(
  speed = 1,
  rows = 0,
  reduced = prefersReducedMotion(),
) {
  return reduced || rows > 1000
    ? 200
    : ((rows > 300 ? 300 : MOTION.rowMove) + MOTION.stepGap) /
        Math.max(0.5, Math.min(2, speed));
}

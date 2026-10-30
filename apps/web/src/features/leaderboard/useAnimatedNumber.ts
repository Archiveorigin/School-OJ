import { ref, watch, onBeforeUnmount, type Ref } from "vue";
import { MOTION } from "./motion";
import { useReducedMotion } from "./useReducedMotion";
const animations = new Set<(now: number) => boolean>();
let frame: number | undefined;
function tick(now: number) {
  frame = undefined;
  for (const animate of animations)
    if (!animate(now)) animations.delete(animate);
  if (animations.size) frame = requestAnimationFrame(tick);
}
function remove(animate: ((now: number) => boolean) | undefined) {
  if (animate) animations.delete(animate);
  if (!animations.size && frame !== undefined) {
    cancelAnimationFrame(frame);
    frame = undefined;
  }
}
export function useAnimatedNumber(
  value: () => number,
  enabled: () => boolean = () => true,
  speed: () => number = () => 1,
) {
  const display = ref(value()),
    reduced = useReducedMotion();
  let active: ((now: number) => boolean) | undefined;
  watch([value, reduced, enabled, speed], ([target]) => {
    remove(active);
    const to = Number(target);
    if (
      reduced.value ||
      !enabled() ||
      typeof requestAnimationFrame === "undefined"
    ) {
      display.value = to;
      return;
    }
    const from = display.value,
      start = performance.now(),
      duration = MOTION.number / Math.max(0.5, Math.min(2, speed()));
    active = (now) => {
      const progress = Math.min(1, (now - start) / duration);
      display.value =
        progress === 1
          ? to
          : Math.round(from + (to - from) * (1 - (1 - progress) ** 3));
      return progress < 1;
    };
    animations.add(active);
    if (frame === undefined) frame = requestAnimationFrame(tick);
  });
  onBeforeUnmount(() => remove(active));
  return display as Ref<number>;
}

import { ref, onMounted, onBeforeUnmount } from "vue";
import { prefersReducedMotion } from "./motion";
export function useReducedMotion() {
  const reduced = ref(prefersReducedMotion());
  let query: MediaQueryList | undefined;
  const update = () => (reduced.value = query?.matches ?? false);
  onMounted(() => {
    query = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    update();
    query?.addEventListener("change", update);
  });
  onBeforeUnmount(() => query?.removeEventListener("change", update));
  return reduced;
}

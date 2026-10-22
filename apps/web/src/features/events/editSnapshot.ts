import { computed, reactive, ref } from "vue";

/** Keep local edits; accept a newer concurrency token only for unchanged server fields. */
export function useEditSnapshot<T extends object>(initial: T) {
  const form = reactive({ ...initial }) as T;
  const revision = ref(0);
  const baseline = ref(JSON.stringify(form));
  const dirty = computed(() => JSON.stringify(form) !== baseline.value);
  function begin(value: Partial<T>, version: number) {
    for (const key of Object.keys(initial) as Array<keyof T>) {
      form[key] = value[key] ?? initial[key];
    }
    revision.value = version;
    baseline.value = JSON.stringify(form);
  }
  function markSaved() {
    baseline.value = JSON.stringify(form);
  }
  function sync(value: Partial<T>, version: number) {
    if (!dirty.value) {
      begin(value, version);
      return;
    }
    const incoming = Object.fromEntries(
      Object.keys(initial).map((key) => [
        key,
        value[key as keyof T] ?? initial[key as keyof T],
      ]),
    );
    if (JSON.stringify(incoming) === baseline.value) revision.value = version;
  }
  return { form, revision, dirty, begin, sync, markSaved };
}

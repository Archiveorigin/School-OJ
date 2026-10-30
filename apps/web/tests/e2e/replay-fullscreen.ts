import { createApp, h } from "vue";
import { createPinia } from "pinia";
import ContestReplay from "../../src/components/events/ContestReplay.vue";
import { useAuthStore } from "../../src/stores/auth";
import "../../src/styles/tokens.css";
import "../../src/style.css";
const pinia = createPinia();
const app = createApp({
  render: () => h(ContestReplay, { kind: "contest", eventId: 99 }),
}).use(pinia);
useAuthStore(pinia).setSession("qa-local-only", {
  id: 1,
  name: "QA",
  role: "admin",
  permissions: ["event:review"],
} as any);
app.mount("#app");

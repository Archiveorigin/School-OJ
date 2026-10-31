import { createApp, h } from "vue";
import { createPinia } from "pinia";
import TeamMembers from "../../src/views/teams/TeamMembers.vue";
import EventRankingReview from "../../src/components/events/EventRankingReview.vue";
import { useAuthStore } from "../../src/stores/auth";
import "../../src/styles/tokens.css";
import "../../src/style.css";
const pinia = createPinia();
const app = createApp({
  render: () =>
    h("main", [
      h(TeamMembers, { team: { id: 1, can_manage: false } as any }),
      h(EventRankingReview, { kind: "contest", eventId: 99 }),
    ]),
}).use(pinia);
useAuthStore(pinia).setSession("qa-local-only", {
  id: 1,
  name: "QA",
  role: "admin",
  permissions: ["event:review"],
} as any);
app.mount("#app");

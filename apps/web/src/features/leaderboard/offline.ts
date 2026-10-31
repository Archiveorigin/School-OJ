import { createApp, h } from "vue";
import { createPinia } from "pinia";
import LeaderboardBoard from "../../components/LeaderboardBoard.vue";
import { adaptExamRanking } from "./adapters";
import "../../styles/tokens.css";
import "../../style.css";
const snapshot = JSON.parse(
  document.getElementById("ranking-data")!.textContent!,
);
const board = adaptExamRanking({
  ...snapshot,
  exam: snapshot,
  now: snapshot.ends_at,
  final_ready: true,
  frozen: false,
});
board.contextKind = snapshot.kind;
board.participantCount = board.rows.filter((row) => !row.unranked).length;
board.currentTimeSeconds = board.durationSeconds;
document.title = snapshot.title;
createApp({
  render: () =>
    h(
      "main",
      { style: { padding: "16px", maxWidth: "1800px", margin: "auto" } },
      [
        h(LeaderboardBoard, { data: board }),
        h(
          "footer",
          { style: { padding: "16px", fontSize: "12px" } },
          `开始 ${new Date(snapshot.starts_at).toLocaleString("zh-CN", { timeZone: "Asia/Shanghai" })} · 结束 ${new Date(snapshot.ends_at).toLocaleString("zh-CN", { timeZone: "Asia/Shanghai" })}`,
        ),
      ],
    ),
})
  .use(createPinia())
  .mount("#app");

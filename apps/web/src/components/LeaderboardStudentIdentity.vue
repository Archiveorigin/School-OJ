<template>
  <div class="student-identity" :class="`student-identity-${variant}`">
    <div class="student-avatar" aria-hidden="true">{{ initial }}</div>
    <div class="student-copy">
      <div v-if="row.meta" class="student-meta">{{ row.meta }}</div>
      <div class="student-main">
        <span class="student-diamond" aria-hidden="true"></span>
        <strong class="student-number">学号 {{ row.studentNo || '-' }}</strong>
        <strong class="student-name">{{ row.name }}</strong>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { LeaderboardRow, LeaderboardScoringRule } from '../features/leaderboard/types'

const props = defineProps<{ row: LeaderboardRow, variant: LeaderboardScoringRule }>()
const initial = computed(() => props.row.name.trim().slice(0, 1) || '学')
</script>

<style scoped>
.student-identity {
  position: relative;
  min-width: 0;
  display: flex;
  align-items: center;
  color: var(--score-text);
}
.student-avatar {
  flex: none;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--student-avatar-color, #5b78a1);
  background: var(--student-avatar-bg, #e8f0fb);
  font-weight: 900;
  opacity: .34;
}
.student-copy { min-width: 0; display: grid; }
.student-meta {
  overflow: hidden;
  color: var(--score-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.student-main { min-width: 0; display: flex; align-items: center; }
.student-diamond { flex: none; width: 14px; height: 14px; transform: rotate(45deg); background: #5879a6; }
.student-number, .student-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.student-number { flex: none; }
.student-name { min-width: 0; }

.student-identity-penalty { height: 85px; gap: 14px; padding: 8px 16px 6px 10px; }
.student-identity-penalty .student-avatar { width: 68px; height: 68px; font-size: 30px; }
.student-identity-penalty .student-copy { gap: 9px; }
.student-identity-penalty .student-meta { font-size: 12px; }
.student-identity-penalty .student-main { gap: 13px; }
.student-identity-penalty .student-number { font-size: 16px; }
.student-identity-penalty .student-name { margin-left: 18px; font-size: 19px; }

.student-identity-score { height: 118px; gap: 18px; padding: 10px 22px 10px 16px; }
.student-identity-score .student-avatar { width: 70px; height: 70px; font-size: 32px; }
.student-identity-score .student-copy { gap: 15px; }
.student-identity-score .student-meta { font-size: 13px; }
.student-identity-score .student-main { gap: 16px; }
.student-identity-score .student-number { font-size: 18px; }
.student-identity-score .student-name { margin-left: 24px; font-size: 21px; }

@media (max-width: 520px) {
  .student-identity-penalty .student-name { margin-left: 8px; }
  .student-identity-score .student-name { margin-left: 10px; }
}
</style>

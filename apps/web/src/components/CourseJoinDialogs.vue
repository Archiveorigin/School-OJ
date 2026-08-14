<template>
  <el-dialog
    v-model="inviteVisible"
    title="邀请码加入课程"
    width="min(460px, calc(100vw - 28px))"
    align-center
    @closed="inviteCode = ''"
  >
    <div class="invite-content">
      <div class="invite-icon" aria-hidden="true">
        <img :src="courseAddIcon" alt="" />
      </div>
      <div>
        <p>输入任课教师提供的课程邀请码即可加入课程。</p>
        <el-input
          v-model="inviteCode"
          size="large"
          clearable
          autocomplete="off"
          placeholder="请输入课程邀请码"
          @keyup.enter="joinInvitedCourse"
        />
      </div>
    </div>
    <template #footer>
      <el-button @click="inviteVisible = false">取消</el-button>
      <el-button
        type="primary"
        :loading="joining"
        :disabled="!inviteCode.trim()"
        @click="joinInvitedCourse"
      >
        加入课程
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { ref } from 'vue'
import { client } from '../api/client'
import courseAddIcon from '../assets/course-add.svg'

const emit = defineEmits<{
  (event: 'joined', courseID: number): void
}>()

const inviteVisible = ref(false)
const inviteCode = ref('')
const joining = ref(false)

function openInvite() {
  inviteCode.value = ''
  inviteVisible.value = true
}

async function joinInvitedCourse() {
  const code = inviteCode.value.trim()
  if (!code) {
    ElMessage.warning('请输入课程邀请码')
    return
  }

  joining.value = true
  try {
    const response = await client.post('/courses/join', { join_code: code })
    const courseID = Number(response.data?.course_id)
    inviteVisible.value = false
    inviteCode.value = ''
    ElMessage.success('已加入课程')
    emit('joined', Number.isInteger(courseID) && courseID > 0 ? courseID : 0)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '课程邀请码无效')
  } finally {
    joining.value = false
  }
}

defineExpose({ openInvite })
</script>

<style scoped>
.invite-content {
  display: grid;
  grid-template-columns: 84px minmax(0, 1fr);
  gap: 18px;
  align-items: center;
}

.invite-icon {
  display: grid;
  width: 72px;
  height: 72px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--accent) 22%, var(--border));
  border-radius: 18px;
  background: color-mix(in srgb, var(--accent) 8%, var(--surface-strong));
}

.invite-icon img {
  width: 50px;
  height: 50px;
  object-fit: contain;
}

.invite-content p {
  margin: 0 0 12px;
  color: var(--muted);
  line-height: 1.7;
}

@media (max-width: 520px) {
  .invite-content {
    grid-template-columns: 1fr;
  }

  .invite-icon {
    justify-self: center;
  }
}
</style>

<template>
  <section class="exam-create">
    <div class="create-actions">
      <el-button @click="router.push('/admin/exams')">返回考试管理</el-button
      ><el-button type="primary" :loading="saving" @click="create"
        >创建草稿并组题</el-button
      >
    </div>
    <el-alert
      title="先创建本场草稿，再自由添加已有题目、新题和附件，完成后提交整场审核。"
      type="info"
      :closable="false"
    />
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <ExamBasicInfoStep
      v-model="form"
      :courses="courses"
      :classes="classes"
      :course-label="courseLabel"
      :class-label="classLabel"
      :problem-count="0"
      :total-score="0"
      @class-change="syncCourse"
      @course-change="form.class_id = -1"
    />
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { apiErrorMessage } from "../api/errors";
import { listCourses } from "../api/modules/courses";
import { createExam } from "../api/modules/exams";
import ExamBasicInfoStep from "../components/exams/ExamBasicInfoStep.vue";
import type { CourseSummary, ExamDraft } from "../features/exams/types";
import { useClassroomStore } from "../stores/classroom";
const router = useRouter(),
  route = useRoute(),
  classroom = useClassroomStore();
const courses = ref<CourseSummary[]>([]),
  saving = ref(false),
  error = ref("");
const form = reactive<ExamDraft>({
  course_id: Number(route.query.course_id) || undefined,
  class_id: -1,
  title: "",
  description: "",
  starts_at: null,
  ends_at: null,
  manual_review: false,
  ranking_visible: false,
  scoring_rule: "acm",
  freeze_enabled: false,
  freeze_duration_minutes: 60,
});
const classes = computed(() =>
  form.course_id
    ? classroom.classes.filter((c) => c.course_id === form.course_id)
    : classroom.classes,
);
const courseLabel = computed(() => {
  const c = courses.value.find((c) => c.id === form.course_id);
  return c ? `${c.code} ${c.name}` : "";
});
const classLabel = computed(() =>
  form.class_id === -1
    ? "全课程"
    : classroom.classes.find((c) => c.class_id === form.class_id)?.class_name ||
      "",
);
function syncCourse() {
  const c = classroom.classes.find((c) => c.class_id === form.class_id);
  if (c) form.course_id = c.course_id;
}
async function create() {
  if (saving.value) return;
  error.value = "";
  if (
    !form.course_id ||
    !form.title.trim() ||
    !form.starts_at ||
    !form.ends_at ||
    new Date(form.starts_at).getTime() <= Date.now() ||
    new Date(form.ends_at).getTime() <= new Date(form.starts_at).getTime()
  ) {
    error.value = "请填写课程、考试名称以及有效的未来开始和结束时间";
    return;
  }
  saving.value = true;
  try {
    const result = await createExam({
      ...form,
      class_id: form.class_id && form.class_id > 0 ? form.class_id : null,
      problems: [],
    });
    await router.push(`/exams/${result.id}/description`);
  } catch (e) {
    error.value = apiErrorMessage(e);
  } finally {
    saving.value = false;
  }
}
onMounted(async () => {
  try {
    await classroom.load();
    courses.value = await listCourses();
    if (!form.course_id) form.course_id = courses.value[0]?.id;
  } catch (e) {
    error.value = apiErrorMessage(e);
  }
});
</script>
<style scoped>
.exam-create {
  display: grid;
  gap: 16px;
  padding: 24px;
}
.create-actions {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
@media (max-width: 600px) {
  .exam-create {
    padding: 14px;
  }
}
</style>

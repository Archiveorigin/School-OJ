<template>
  <div class="exam-authoring">
    <div class="authoring-workspace">
      <ExamWizardHeader
        :step="step"
        :can-continue="canContinue"
        :saving="saving"
        :saved-at="savedAt"
        @back="returnToExams"
        @save="saveDraft(true)"
        @continue="continueWizard"
        @go="step = $event"
      />

      <ExamBasicInfoStep
        v-if="step === 0"
        v-model="form"
        :courses="courses"
        :classes="scopedClasses"
        :course-label="courseLabel"
        :class-label="classLabel"
        :problem-count="selectedProblems.length"
        :total-score="selectedTotalScore"
        @class-change="syncCourseFromClass"
        @course-change="onCourseWideSelect"
      />

      <ExamProblemSelectionStep
        v-else-if="step === 1"
        v-model:problems="selectedProblems"
        :class-choices="classProblemOptions"
        :prepared-choices="preparedProblemOptions"
        :exam-title="form.title"
        :total="selectedTotalScore"
        :scoring-rule="form.scoring_rule"
        @add="addSelectedProblem"
        @create="openMarkdownDialog"
        @batch="openBatchDialog"
      />

      <ExamPublishReviewStep
        v-else
        :draft="form"
        :problems="selectedProblems"
        :course-label="courseLabel"
        :class-label="classLabel"
        :total="selectedTotalScore"
        :saving="saving"
        @publish="submitCreate"
        @back="step = 1"
      />

      <!-- Batch Markdown Import Dialog -->
      <el-dialog
        v-model="batchDialogVisible"
        title="批量导入 Markdown 题目"
        width="960px"
        destroy-on-close
        :close-on-click-modal="false"
      >
        <div v-if="!batchParsed.length" class="batch-upload-area">
          <div class="batch-upload-hint">
            <p>上传一个包含多道题目的 <code>.md</code> 文件。</p>
            <p class="muted">
              每道题目由 <code>---</code> 分隔，支持 YAML
              头信息（title、限制等）和嵌入式测试点。
            </p>
          </div>
          <el-upload
            drag
            action="#"
            :auto-upload="false"
            :show-file-list="true"
            :file-list="batchFileList"
            accept=".md"
            :on-change="handleBatchFile"
            :on-remove="
              () => {
                batchFileList = [];
                batchParsed = [];
              }
            "
            :limit="1"
          >
            <div class="upload-text">选择或拖入 .md 文件</div>
            <div class="muted">文件最大 32 MB</div>
          </el-upload>
          <el-button
            type="primary"
            :loading="batchParsing"
            :disabled="!batchFileList.length"
            style="margin-top: 14px"
            @click="previewBatch"
          >
            解析预览
          </el-button>
        </div>

        <div v-else class="batch-preview">
          <div class="batch-preview-header">
            <span
              >解析到 <strong>{{ batchParsed.length }}</strong> 道题目</span
            >
            <el-tag v-if="batchWarnings.length" type="warning" effect="plain">
              {{ batchWarnings.length }} 条提醒
            </el-tag>
          </div>
          <div v-if="batchWarnings.length" class="batch-warnings">
            <el-alert
              v-for="(w, i) in batchWarnings"
              :key="i"
              :title="w"
              type="warning"
              show-icon
              :closable="false"
              style="margin-bottom: 4px"
            />
          </div>
          <el-table
            :data="batchParsed"
            max-height="420"
            size="small"
            style="margin-top: 12px"
          >
            <el-table-column prop="label" label="题号" width="70" />
            <el-table-column prop="title" label="标题" min-width="180" />
            <el-table-column label="测试点" width="80">
              <template #default="{ row }">
                <el-tag
                  :type="(row.cases || []).length > 0 ? 'success' : 'danger'"
                  effect="plain"
                  size="small"
                >
                  {{ (row.cases || []).length || 0 }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="限制" width="160">
              <template #default="{ row }">
                <span class="muted" style="font-size: 12px">
                  {{ row.time_limit_ms || 1000 }}ms /
                  {{ row.memory_limit_mb || 256 }}MB /
                  {{ row.output_limit_kb || 1024 }}KB
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="score" label="分值" width="70" />
          </el-table>
          <div class="batch-actions">
            <el-button
              @click="
                batchParsed = [];
                batchFileList = [];
              "
              >重新选择</el-button
            >
            <el-button
              type="primary"
              :loading="batchImporting"
              @click="importBatch"
            >
              导入全部题目
            </el-button>
          </div>
        </div>
      </el-dialog>

      <!-- Markdown Problem Dialog -->
      <el-dialog
        v-model="markdownDialogVisible"
        title="新建 Markdown 题目"
        width="960px"
        destroy-on-close
        :close-on-click-modal="false"
      >
        <el-form label-width="92px" class="problem-form">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="题号">
                <el-input v-model="problemForm.label" maxlength="16" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="分值">
                <el-input-number
                  v-model="problemForm.score"
                  :min="1"
                  :max="1000"
                />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="标题">
            <el-input v-model="problemForm.title" placeholder="两数之和" />
          </el-form-item>
          <el-form-item label="标签">
            <ProblemTagSelector v-model="problemForm.tags" />
          </el-form-item>
          <el-form-item label="题面">
            <el-input
              v-model="problemForm.statement"
              type="textarea"
              :rows="8"
              placeholder="支持 Markdown、LaTeX 和图片。例如：![示意图](assets/example.png)"
            />
            <div class="statement-tools">
              <el-upload
                action="#"
                :auto-upload="false"
                :show-file-list="false"
                multiple
                accept="image/png,image/jpeg,image/gif,image/webp"
                :on-change="addProblemImage"
              >
                <el-button>插入图片</el-button>
              </el-upload>
              <span class="muted"
                >图片会自动写入题面 Markdown，单张不超过 5 MB。</span
              >
            </div>
            <div v-if="problemForm.assets.length" class="asset-row">
              <el-tag
                v-for="asset in problemForm.assets"
                :key="asset.path"
                closable
                @close="removeProblemImage(asset.path)"
              >
                {{ asset.name }}
              </el-tag>
            </div>
            <div class="statement-preview">
              <div class="muted">题面预览</div>
              <MarkdownRenderer
                :source="
                  problemForm.statement || '支持 **Markdown** 和 $a+b$。'
                "
                :asset-urls="problemAssetPreviewUrls"
              />
            </div>
          </el-form-item>
          <el-row :gutter="12">
            <el-col :span="8">
              <el-form-item label="时间限制">
                <el-input-number
                  v-model="problemForm.time_limit_ms"
                  :min="100"
                  :step="100"
                />
                <span class="unit">ms</span>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="内存限制">
                <el-input-number
                  v-model="problemForm.memory_limit_mb"
                  :min="16"
                  :step="16"
                />
                <span class="unit">MB</span>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="输出限制">
                <el-input-number
                  v-model="problemForm.output_limit_kb"
                  :min="1"
                  :step="64"
                />
                <span class="unit">KB</span>
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="隐藏测试点">
            <div class="test-file-panel">
              <el-upload
                :key="testPointUploadKey"
                drag
                action="#"
                multiple
                accept=".zip,.in,.out"
                :auto-upload="false"
                :file-list="testPointUploadFiles"
                :on-change="syncTestPointFiles"
                :on-remove="syncTestPointFiles"
              >
                <div class="upload-text">
                  选择或拖入 .zip / .in / .out 测试点文件
                </div>
                <div class="muted">
                  按文件名中的数字序号配对，例如 data1.in 与 answer1.out；ZIP
                  会在上传后解析，这些文件不会展示给考生。
                </div>
              </el-upload>
              <div v-if="testPointErrors.length" class="test-errors">
                <el-alert
                  v-for="error in testPointErrors"
                  :key="error"
                  type="error"
                  :title="error"
                  show-icon
                  :closable="false"
                />
              </div>
              <el-table
                v-if="problemForm.cases.length"
                :data="problemForm.cases"
                size="small"
                class="test-case-table"
              >
                <el-table-column prop="name" label="测试点" min-width="120" />
                <el-table-column label="输入文件" min-width="180">
                  <template #default="{ row }"
                    >{{ row.inputName }} ·
                    {{ formatBytes(row.inputSize) }}</template
                  >
                </el-table-column>
                <el-table-column label="输出文件" min-width="180">
                  <template #default="{ row }"
                    >{{ row.outputName }} ·
                    {{ formatBytes(row.outputSize) }}</template
                  >
                </el-table-column>
                <el-table-column prop="weight" label="权重" width="80" />
              </el-table>
              <p
                v-else-if="testPointUploadFiles.length"
                class="muted form-note"
              >
                已选择 ZIP 文件，上传后将由后端解析并按数字序号配对。
              </p>
              <p v-else class="muted form-note">
                请至少上传一组完整的 .in / .out 测试点文件，或上传一个测试点
                ZIP。
              </p>
            </div>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="markdownDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="creatingProblem"
            @click="createMarkdownProblem"
          >
            创建并加入考试
          </el-button>
        </template>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { PreparedProblem, Problem } from "../api/client";
import { apiErrorMessage } from "../api/errors";
import { listCourses } from "../api/modules/courses";
import { createExam } from "../api/modules/exams";
import {
  createProblem,
  listPreparedProblems,
  listProblems,
  parseProblemMarkdown,
} from "../api/modules/problems";
import ExamBasicInfoStep from "../components/exams/ExamBasicInfoStep.vue";
import ExamProblemSelectionStep from "../components/exams/ExamProblemSelectionStep.vue";
import ExamPublishReviewStep from "../components/exams/ExamPublishReviewStep.vue";
import ExamWizardHeader from "../components/exams/ExamWizardHeader.vue";
import MarkdownRenderer from "../components/MarkdownRenderer.vue";
import ProblemTagSelector from "../components/ProblemTagSelector.vue";
import {
  examTotalScore,
  nextProblemLabel,
  validateExamDraft,
} from "../features/exams/builder";
import {
  examDraftStorageKey,
  parseStoredExamDraft,
} from "../features/exams/draft";
import type {
  CourseSummary,
  ExamDraft,
  SelectedExamProblem,
} from "../features/exams/types";
import { problemDisplayCode, tagList } from "../features/problems/problemMeta";
import { useClassroomStore } from "../stores/classroom";

type ProblemAssetForm = {
  name: string;
  path: string;
  content_type: string;
  data: string;
  preview_url: string;
};
type ProblemCaseForm = {
  name: string;
  inputName: string;
  inputSize: number;
  outputName: string;
  outputSize: number;
  weight: number;
};
const router = useRouter();
const route = useRoute();
const classroom = useClassroomStore();
const courses = ref<CourseSummary[]>([]);
const problems = ref<Problem[]>([]);
const preparedProblems = ref<PreparedProblem[]>([]);
const selectedProblems = ref<SelectedExamProblem[]>([]);
const step = ref(0);
const saving = ref(false);
const creatingProblem = ref(false);
const markdownDialogVisible = ref(false);
const batchDialogVisible = ref(false);
const batchFileList = ref<any[]>([]);
const batchParsed = ref<any[]>([]);
const batchWarnings = ref<string[]>([]);
const batchParsing = ref(false);
const batchImporting = ref(false);
const testPointUploadFiles = ref<any[]>([]);
const testPointUploadKey = ref(0);
const testPointErrors = ref<string[]>([]);
const readingTestPoints = ref(false);
const savedAt = ref("");
const draftHydrated = ref(false);

const form = reactive<ExamDraft>({
  course_id: undefined,
  class_id: undefined,
  title: "",
  description: "",
  starts_at: null,
  ends_at: null,
  manual_review: false,
  ranking_visible: false,
  scoring_rule: "penalty",
});

const problemForm = reactive({
  label: "A",
  score: 100,
  title: "",
  statement: "",
  tags: [] as string[],
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  assets: [] as ProblemAssetForm[],
  cases: [] as ProblemCaseForm[],
});

const selectedTotalScore = computed(() =>
  examTotalScore(selectedProblems.value),
);
const problemAssetPreviewUrls = computed(() =>
  Object.fromEntries(
    problemForm.assets.map((asset) => [asset.path, asset.preview_url]),
  ),
);

const courseLabel = computed(() => {
  const course = courses.value.find((c: any) => c.id === form.course_id);
  return course ? `${course.code} ${course.name}` : "";
});

const classLabel = computed(() => {
  if (form.class_id === -1) return "全课程";
  const item = classroom.classes.find((c) => c.class_id === form.class_id);
  return item ? `${item.course_code} / ${item.class_name}` : "";
});

const isCourseWide = computed(() => form.class_id === -1);
const requestedCourseID = computed(
  () => Number(route.query.course_id) || undefined,
);
const scopedClasses = computed(() => {
  const courseID = form.course_id || requestedCourseID.value;
  return courseID
    ? classroom.classes.filter((item) => item.course_id === courseID)
    : classroom.classes;
});
const canProceedStep0 = computed(() => {
  const hasScope = isCourseWide.value ? !!form.course_id : !!form.class_id;
  return hasScope && form.title.trim();
});
const canContinue = computed(() => {
  if (step.value === 0) return Boolean(canProceedStep0.value);
  if (step.value === 1) return selectedProblems.value.length > 0;
  return !validateExamDraft(form, selectedProblems.value);
});

const classProblemOptions = computed(() =>
  problems.value.map((problem) => ({
    value: problem.id,
    label: `[题库] ${problemDisplayCode(problem)}. ${problem.title}`,
    title: problem.title,
    source: "题库",
    difficulty: problem.difficulty,
    tags: tagList(problem.tags),
  })),
);

const preparedProblemOptions = computed(() =>
  preparedProblems.value.map((item) => {
    const tags = tagList(item.problem?.tags);
    const suffix = [item.folder, item.difficulty, tags.join("/")]
      .filter(Boolean)
      .join(" · ");
    const code = item.problem ? problemDisplayCode(item.problem) : "未编号";
    return {
      value: item.problem_id,
      label: `[预备] ${code}. ${item.problem?.title || "未知题目"}${suffix ? `（${suffix}）` : ""}`,
      title: item.problem?.title,
      source: "预备",
      difficulty: item.problem?.difficulty || item.difficulty,
      tags,
    };
  }),
);

async function load() {
  const [courseItems, problemItems, preparedItems] = await Promise.all([
    listCourses(),
    listProblems(),
    listPreparedProblems(),
  ]);
  courses.value = courseItems;
  problems.value = problemItems;
  preparedProblems.value = preparedItems;
}

function syncCourseFromClass() {
  if (form.class_id === -1) {
    selectedProblems.value = [];
    return;
  }
  const item = classroom.classes.find(
    (entry) => entry.class_id === form.class_id,
  );
  form.course_id = item?.course_id;
  selectedProblems.value = [];
  loadClassProblems();
}

function onCourseWideSelect() {
  selectedProblems.value = [];
}

async function loadClassProblems() {
  problems.value = await listProblems();
}

function openMarkdownDialog() {
  problemForm.label = nextAvailableLabel();
  markdownDialogVisible.value = true;
}

function addSelectedProblem(source: "class" | "prepared", problemID: number) {
  const options =
    source === "prepared"
      ? preparedProblemOptions.value
      : classProblemOptions.value;
  const option = options.find((item) => item.value === problemID);
  if (!option) return;
  if (selectedProblems.value.some((item) => item.problem_id === option.value)) {
    ElMessage.warning("该题已经加入考试");
    return;
  }
  selectedProblems.value.push({
    problem_id: option.value,
    title: option.title || option.label,
    source: option.source,
    score: 100,
    label: nextAvailableLabel(),
    release_after_exam: source === "prepared",
  });
}

async function createMarkdownProblem() {
  if (!problemForm.label.trim() || !problemForm.title.trim()) {
    ElMessage.error("请填写题号和标题");
    return;
  }
  if (
    selectedProblems.value.some(
      (item) =>
        item.label.trim().toLowerCase() ===
        problemForm.label.trim().toLowerCase(),
    )
  ) {
    ElMessage.error("题号不能重复");
    return;
  }
  if (readingTestPoints.value) {
    ElMessage.warning("测试点文件仍在读取中，请稍后再试");
    return;
  }
  if (testPointErrors.value.length > 0) {
    ElMessage.error("请先修正测试点文件错误");
    return;
  }
  if (testPointUploadFiles.value.length === 0) {
    ElMessage.error(
      "请至少上传一组完整的 .in / .out 测试点文件，或上传一个测试点 ZIP",
    );
    return;
  }
  creatingProblem.value = true;
  try {
    const fd = new FormData();
    fd.append(
      "draft",
      JSON.stringify({
        title: problemForm.title,
        statement: problemForm.statement,
        tags: problemForm.tags,
        time_limit_ms: problemForm.time_limit_ms,
        memory_limit_mb: problemForm.memory_limit_mb,
        output_limit_kb: problemForm.output_limit_kb,
        assets: problemForm.assets.map(
          ({ name, path, content_type, data }) => ({
            name,
            path,
            content_type,
            data,
          }),
        ),
      }),
    );
    for (const item of testPointUploadFiles.value) {
      const file = item.raw as File | undefined;
      if (file) fd.append("tests", file, file.name);
    }
    const data = await createProblem(fd);
    selectedProblems.value.push({
      problem_id: data.id,
      title: data.title,
      source: "出题",
      score: problemForm.score,
      label: problemForm.label.trim(),
      release_after_exam: true,
    });
    ElMessage.success("题目已加入考试，考试结束后同步到题库");
    markdownDialogVisible.value = false;
    resetProblemForm();
  } catch (error: unknown) {
    ElMessage.error(apiErrorMessage(error, "创建题目失败"));
  } finally {
    creatingProblem.value = false;
  }
}

async function submitCreate() {
  const validationError = validateExamDraft(form, selectedProblems.value);
  if (validationError) {
    ElMessage.error(validationError);
    return;
  }
  saving.value = true;
  try {
    const data = await createExam({
      course_id: form.course_id,
      class_id: form.class_id && form.class_id > 0 ? form.class_id : null,
      title: form.title,
      description: form.description,
      starts_at: form.starts_at,
      ends_at: form.ends_at,
      manual_review: form.manual_review,
      ranking_visible: form.ranking_visible,
      scoring_rule: form.scoring_rule,
      problems: selectedProblems.value.map((item) => ({
        problem_id: item.problem_id,
        score: item.score,
        label: item.label.trim(),
        release_after_exam: Boolean(item.release_after_exam),
      })),
    });
    localStorage.removeItem(examDraftStorageKey);
    ElMessage.success("考试已创建");
    router.push(`/exams/${data.id}`);
  } catch (error: unknown) {
    ElMessage.error(apiErrorMessage(error, "创建考试失败"));
  } finally {
    saving.value = false;
  }
}

function nextAvailableLabel() {
  return nextProblemLabel(selectedProblems.value);
}

function openBatchDialog() {
  batchFileList.value = [];
  batchParsed.value = [];
  batchWarnings.value = [];
  batchDialogVisible.value = true;
}

function handleBatchFile(uploadFile: any) {
  batchFileList.value = [uploadFile];
  batchParsed.value = [];
}

async function previewBatch() {
  if (!batchFileList.value.length) return;
  const file = batchFileList.value[0].raw as File;
  if (!file) {
    ElMessage.error("请先选择文件");
    return;
  }
  batchParsing.value = true;
  try {
    const data = await parseProblemMarkdown(file);
    batchParsed.value = data.problems;
    batchWarnings.value = data.warnings;
    if (batchParsed.value.length === 0) {
      ElMessage.warning("未解析到任何题目，请检查文件格式");
    } else {
      ElMessage.success(`解析到 ${batchParsed.value.length} 道题目`);
    }
  } catch (error: unknown) {
    ElMessage.error(apiErrorMessage(error, "解析 Markdown 失败"));
  } finally {
    batchParsing.value = false;
  }
}

async function importBatch() {
  if (!batchParsed.value.length) return;
  batchImporting.value = true;
  let created = 0;
  try {
    for (const problem of batchParsed.value) {
      const cases = (problem.cases || []).map((c: any, i: number) => ({
        name: c.name || `case-${String(i + 1).padStart(2, "0")}`,
        input: c.input || "",
        output: c.output || "",
        weight: c.weight || 0,
      }));
      // If no embedded test cases, skip this problem
      if (cases.length === 0) {
        ElMessage.warning(`题目「${problem.title}」没有测试点，跳过`);
        continue;
      }
      const data = await createProblem({
        title: problem.title,
        statement: problem.statement || "",
        time_limit_ms: problem.time_limit_ms || 1000,
        memory_limit_mb: problem.memory_limit_mb || 256,
        output_limit_kb: problem.output_limit_kb || 1024,
        cases: cases,
      });
      selectedProblems.value.push({
        problem_id: data.id,
        title: data.title,
        source: "出题",
        score: problem.score || 100,
        label: problem.label || nextAvailableLabel(),
        release_after_exam: true,
      });
      created++;
    }
    if (created > 0) {
      ElMessage.success(`已导入 ${created} 道题目`);
    }
    batchDialogVisible.value = false;
    batchFileList.value = [];
    batchParsed.value = [];
  } catch (error: unknown) {
    ElMessage.error(apiErrorMessage(error, "批量导入题目失败"));
  } finally {
    batchImporting.value = false;
  }
}

function resetProblemForm() {
  problemForm.assets.forEach((asset) => URL.revokeObjectURL(asset.preview_url));
  problemForm.label = nextAvailableLabel();
  problemForm.score = 100;
  problemForm.title = "";
  problemForm.statement = "";
  problemForm.tags = [];
  problemForm.time_limit_ms = 1000;
  problemForm.memory_limit_mb = 256;
  problemForm.output_limit_kb = 1024;
  problemForm.assets.splice(0, problemForm.assets.length);
  problemForm.cases.splice(0, problemForm.cases.length);
  testPointUploadFiles.value = [];
  testPointUploadKey.value += 1;
  testPointErrors.value = [];
}

async function syncTestPointFiles(_uploadFile: any, uploadFiles: any[]) {
  testPointUploadFiles.value = [...uploadFiles];
  await rebuildTestCasesFromFiles(testPointUploadFiles.value);
}

async function rebuildTestCasesFromFiles(uploadFiles: any[]) {
  readingTestPoints.value = true;
  try {
    const errors: string[] = [];
    const groups = new Map<number, { input?: File; output?: File }>();
    const seen = new Set<string>();
    for (const item of uploadFiles) {
      const file = item.raw as File | undefined;
      if (!file) continue;
      if (file.name.toLowerCase().endsWith(".zip")) continue;
      const ext = file.name.toLowerCase().endsWith(".in")
        ? ".in"
        : file.name.toLowerCase().endsWith(".out")
          ? ".out"
          : "";
      if (!ext) {
        errors.push(`${file.name} 不是 .zip、.in 或 .out 文件`);
        continue;
      }
      const base = file.name.slice(0, -ext.length).trim();
      if (!base) {
        errors.push(`${file.name} 缺少测试点名称`);
        continue;
      }
      const seq = extractLastNumber(base);
      if (!seq) {
        errors.push(`${file.name} 缺少数字测试点序号`);
        continue;
      }
      const key = `${seq}${ext}`;
      if (seen.has(key)) {
        errors.push(`${file.name} 与已有第 ${seq} 个${ext}文件重复`);
        continue;
      }
      seen.add(key);
      const group = groups.get(seq) || {};
      if (ext === ".in") group.input = file;
      else group.output = file;
      groups.set(seq, group);
    }
    const cases: ProblemCaseForm[] = [];
    for (const [seq, group] of [...groups.entries()].sort(
      ([a], [b]) => a - b,
    )) {
      if (!group.input || !group.output) {
        errors.push(
          `第 ${seq} 个测试点缺少 ${group.input ? ".out" : ".in"} 配对文件`,
        );
        continue;
      }
      cases.push({
        name: `case-${String(cases.length + 1).padStart(2, "0")}`,
        inputName: group.input.name,
        inputSize: group.input.size,
        outputName: group.output.name,
        outputSize: group.output.size,
        weight: 1,
      });
    }
    problemForm.cases.splice(0, problemForm.cases.length, ...cases);
    testPointErrors.value = errors;
  } catch (err: any) {
    problemForm.cases.splice(0, problemForm.cases.length);
    testPointErrors.value = [err.message || "读取测试点文件失败"];
  } finally {
    readingTestPoints.value = false;
  }
}

function extractLastNumber(value: string) {
  const match = value.match(/(\d+)(?!.*\d)/);
  if (!match) return 0;
  return Number(match[1]);
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function addProblemImage(uploadFile: any) {
  const file = uploadFile.raw as File | undefined;
  if (!file) return;
  if (
    !["image/png", "image/jpeg", "image/gif", "image/webp"].includes(file.type)
  ) {
    ElMessage.error("仅支持 PNG、JPG、GIF、WebP 图片");
    return;
  }
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error("单张图片不能超过 5 MB");
    return;
  }
  const path = uniqueAssetPath(file.name);
  const reader = new FileReader();
  reader.onload = () => {
    problemForm.assets.push({
      name: file.name,
      path,
      content_type: file.type,
      data: String(reader.result),
      preview_url: URL.createObjectURL(file),
    });
    problemForm.statement = `${problemForm.statement.trimEnd()}\n\n![${file.name}](${path})\n`;
  };
  reader.readAsDataURL(file);
}

function removeProblemImage(path: string) {
  const index = problemForm.assets.findIndex((asset) => asset.path === path);
  if (index < 0) return;
  URL.revokeObjectURL(problemForm.assets[index].preview_url);
  problemForm.assets.splice(index, 1);
  problemForm.statement = problemForm.statement
    .replace(
      new RegExp(`!\\[[^\\]]*\\]\\(${escapeRegExp(path)}\\)\\n?`, "g"),
      "",
    )
    .trimEnd();
}

function uniqueAssetPath(name: string) {
  const safe = name
    .trim()
    .replace(/\s+/g, "-")
    .replace(/[^A-Za-z0-9._-]/g, "")
    .replace(/^\.+/, "");
  const fallback = `image-${Date.now()}.png`;
  const base = safe || fallback;
  let path = `assets/${base}`;
  let index = 1;
  while (problemForm.assets.some((asset) => asset.path === path)) {
    const dot = base.lastIndexOf(".");
    path =
      dot > 0
        ? `assets/${base.slice(0, dot)}-${index}${base.slice(dot)}`
        : `assets/${base}-${index}`;
    index += 1;
  }
  return path;
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function continueWizard() {
  if (step.value === 0) {
    if (!canProceedStep0.value) {
      ElMessage.warning("请先填写考试名称并选择可见范围");
      return;
    }
    step.value = 1;
    return;
  }
  if (step.value === 1) {
    if (!selectedProblems.value.length) {
      ElMessage.warning("请至少选择一道题目");
      return;
    }
    step.value = 2;
    return;
  }
  void submitCreate();
}

function saveDraft(notify = false) {
  if (!draftHydrated.value) return;
  const saved = new Date();
  localStorage.setItem(
    examDraftStorageKey,
    JSON.stringify({
      form,
      selectedProblems: selectedProblems.value,
      step: step.value,
      savedAt: saved.toISOString(),
    }),
  );
  savedAt.value = saved.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
  if (notify) ElMessage.success("草稿已保存到当前浏览器");
}

function restoreDraft() {
  const raw = localStorage.getItem(examDraftStorageKey);
  if (!raw) return false;
  try {
    const saved = parseStoredExamDraft(raw);
    if (!saved) return false;
    Object.assign(form, saved.form, {
      starts_at: saved.form.starts_at ? new Date(saved.form.starts_at) : null,
      ends_at: saved.form.ends_at ? new Date(saved.form.ends_at) : null,
    });
    selectedProblems.value = saved.selectedProblems;
    step.value = Math.min(2, Math.max(0, Number(saved.step) || 0));
    if (saved.savedAt) {
      savedAt.value = new Date(saved.savedAt).toLocaleTimeString("zh-CN", {
        hour: "2-digit",
        minute: "2-digit",
      });
    }
    return true;
  } catch {
    localStorage.removeItem(examDraftStorageKey);
    return false;
  }
}

onMounted(async () => {
  await classroom.load();
  await load();
  const restored = restoreDraft();
  if (!restored && requestedCourseID.value) {
    form.course_id = requestedCourseID.value;
    form.class_id = scopedClasses.value[0]?.class_id || -1;
  } else if (!restored) {
    form.class_id = classroom.classes[0]?.class_id || -1;
    syncCourseFromClass();
  }
  problemForm.label = nextAvailableLabel();
  draftHydrated.value = true;
});

watch([() => form, selectedProblems, step], () => saveDraft(false), {
  deep: true,
});

function returnToExams() {
  const courseID = form.course_id || requestedCourseID.value;
  router.push({
    path: "/admin/exams",
    query: courseID ? { course_id: courseID } : {},
  });
}
</script>

<style scoped>
.exam-authoring {
  display: flex;
  min-height: calc(100vh - 64px);
  color: #172033;
  background: #f7f9fc;
}

.authoring-workspace {
  min-width: 0;
  flex: 1;
}

@media (max-width: 900px) {
  .exam-authoring {
    flex-direction: column;
  }
}

.steps-bar {
  margin-bottom: 22px;
  padding: 18px 28px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
}

.step-panel {
  animation: step-in 0.22s ease-out;
}

@keyframes step-in {
  from {
    opacity: 0;
    transform: translateX(10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.exam-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  align-items: start;
  margin-bottom: 18px;
}

.info-panel {
  padding: 20px;
}

.info-form {
  padding-top: 4px;
}

.rule-list {
  margin: 0;
  padding-left: 18px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.8;
}

.problem-workspace {
  display: grid;
  grid-template-columns: minmax(320px, 1fr) minmax(380px, 1.2fr);
  gap: 14px;
  align-items: start;
  margin-bottom: 18px;
}

.source-select-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.create-problem-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 0;
}

.create-problem-info p {
  margin: 0 0 4px;
  font-size: 14px;
}

.form-note {
  margin: 10px 0 0;
}

.scoring-note {
  flex-basis: 100%;
  font-size: 12px;
  line-height: 1.5;
}

.selected-stats {
  display: flex;
  gap: 8px;
}

.empty-state {
  text-align: center;
  padding: 40px 16px;
}

.selected-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.selected-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 14px;
  background: var(--surface-strong);
  transition:
    border-color 0.2s,
    box-shadow 0.2s;
}

.selected-card:hover {
  border-color: var(--accent);
  box-shadow: 0 4px 14px rgba(10, 94, 166, 0.08);
}

.selected-card-body {
  display: flex;
  align-items: center;
  gap: 12px;
}

.selected-card-label {
  flex: 0 0 64px;
}

.label-input {
  width: 100%;
}

.selected-card-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.selected-card-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.release-note {
  font-size: 12px;
  white-space: nowrap;
}

.selected-card-score {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.selected-card-score :deep(.el-input-number) {
  width: 100px;
}

.selected-card-action {
  flex-shrink: 0;
}

.review-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  align-items: start;
  margin-bottom: 18px;
}

.step-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 8px;
}

.problem-form {
  max-height: 68vh;
  overflow-y: auto;
  padding-right: 4px;
}

.statement-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.asset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.statement-preview {
  margin-top: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.03);
}

:root.dark .statement-preview {
  background: rgba(255, 255, 255, 0.03);
}

.preview-samples,
.test-file-panel,
.test-errors {
  display: grid;
  gap: 10px;
  width: 100%;
}

.preview-samples {
  margin-top: 12px;
}

.preview-sample-pair {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.preview-sample {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface);
}

.sample-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}

.preview-sample pre {
  max-height: 220px;
  margin: 0;
  overflow: auto;
  padding: 10px;
  color: #e2e8f0;
  background: #0f172a;
}

.test-case-table {
  width: 100%;
}

.unit {
  margin-left: 8px;
  color: var(--muted);
}

@media (max-width: 980px) {
  .exam-info-grid,
  .problem-workspace,
  .review-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .steps-bar {
    padding: 14px 8px;
  }

  .selected-card-body {
    flex-wrap: wrap;
  }

  .selected-card-label {
    flex: 0 0 56px;
  }

  .selected-card-score {
    width: 100%;
    justify-content: flex-end;
  }

  .create-problem-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .preview-sample-pair {
    grid-template-columns: 1fr;
  }
}
.batch-upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.batch-upload-hint {
  text-align: center;
  margin-bottom: 16px;
}

.batch-upload-hint p {
  margin: 0 0 6px;
}

.batch-preview-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.batch-warnings {
  max-height: 150px;
  overflow-y: auto;
}

.batch-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}
</style>

<template>
  <div class="tag-selector-field">
    <div class="selected-preview" @click="open">
      <el-tag v-for="tag in modelValue" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
      <span v-if="!modelValue.length" class="placeholder">选择算法、年份和来源标签</span>
    </div>
    <el-button @click="open">选择标签</el-button>
  </div>

  <el-dialog
    v-model="visible"
    title="选择标签"
    width="min(980px, calc(100vw - 24px))"
    append-to-body
    destroy-on-close
    class="problem-tag-dialog"
  >
    <div class="tag-dialog-toolbar">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="算法" name="algorithm" />
        <el-tab-pane label="时间" name="year" />
        <el-tab-pane label="来源" name="source" />
      </el-tabs>
      <el-input v-if="activeTab === 'algorithm'" v-model="keyword" clearable placeholder="搜索全部算法标签" class="tag-search" />
    </div>

    <section class="picked-section">
      <span>已选标签（{{ draftTags.length }}）</span>
      <div class="picked-tags">
        <el-tag v-for="tag in draftTags" :key="tag" closable @close="removeTag(tag)">{{ tag }}</el-tag>
        <span v-if="!draftTags.length" class="muted">暂无已选标签</span>
      </div>
    </section>

    <div class="tag-scroll">
      <template v-if="activeTab === 'algorithm'">
        <section v-for="group in filteredGroups" :key="group.name" class="algorithm-group">
          <h3>{{ group.name }}</h3>
          <div class="tag-options">
            <button
              v-for="tag in group.tags"
              :key="tag"
              type="button"
              :class="{ selected: draftAlgorithms.includes(tag) }"
              @click="toggleAlgorithm(tag)"
            >
              {{ tag }}
            </button>
          </div>
        </section>
        <el-empty v-if="!filteredGroups.length" description="没有匹配的算法标签" />
      </template>

      <section v-else-if="activeTab === 'year'" class="single-select-section">
        <h3>选择题目年份</h3>
        <p>时间标签只保留一个年份，可随时重新选择。</p>
        <el-select v-model="draftYear" clearable filterable placeholder="选择年份">
          <el-option v-for="year in yearOptions" :key="year" :label="`${year} 年`" :value="String(year)" />
        </el-select>
      </section>

      <section v-else class="single-select-section">
        <h3>选择题目来源</h3>
        <p>可选择常用来源，也可填写校内赛、训练营等自定义来源。</p>
        <el-select v-model="draftSource" clearable filterable allow-create default-first-option placeholder="选择或输入来源">
          <el-option v-for="source in sourceOptions" :key="source" :label="source" :value="source" />
        </el-select>
      </section>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirm">确认</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: string[]): void }>()

const algorithmGroups = [
  { name: '语言入门', tags: ['语言入门', '顺序结构', '分支结构', '循环结构', '数组', '字符串（入门）', '结构体', '函数与递归'] },
  { name: '基础算法', tags: ['枚举', '模拟', '排序', '快速排序', '归并排序', '桶排序', '贪心', '分治', '二分查找', '二分答案', '前缀和', '差分', '双指针', '离散化', '位运算', '随机化'] },
  { name: '数学', tags: ['数学', '高精度', '数论', '质数筛', '最大公约数 GCD', '快速幂', '矩阵快速幂', '扩展欧几里得', '同余与逆元', '中国剩余定理', '组合数学', '容斥原理', '概率与期望', '线性代数', '博弈论'] },
  { name: '计算几何', tags: ['计算几何', '向量与叉积', '点线关系', '线段相交', '凸包', '旋转卡壳', '扫描线', '半平面交', '最近点对'] },
  { name: '搜索', tags: ['搜索', '深度优先搜索 DFS', '广度优先搜索 BFS', '回溯', '剪枝', '启发式搜索', 'A* 搜索', '双向搜索', '迭代加深', '记忆化搜索', 'Dancing Links'] },
  { name: '动态规划 DP', tags: ['动态规划 DP', '线性 DP', '背包 DP', '多重背包', '完全背包', '区间 DP', '树形 DP', '状态压缩 DP', '数位 DP', '概率 DP', '插头 DP', '斜率优化 DP', '单调队列优化 DP', 'DP 优化'] },
  { name: '图论', tags: ['图论', '图的遍历', '最短路', 'Dijkstra', 'Bellman-Ford', 'Floyd', '最小生成树', '拓扑排序', '强连通分量', '割点与桥', '双连通分量', '二分图', '二分图匹配', '网络流', '最小割', '费用流', '差分约束', '欧拉路径', '树上问题'] },
  { name: '树上算法', tags: ['树的直径', '最近公共祖先 LCA', '树上倍增', '树链剖分', '点分治', '树上差分', '虚树', 'DSU on Tree'] },
  { name: '数据结构', tags: ['数据结构', '栈', '单调栈', '队列', '单调队列', '链表', '堆', '并查集', '带权并查集', '树状数组', '线段树', '懒标记线段树', '平衡树', 'Treap', 'ST 表', '分块', '莫队算法', '可持久化数据结构'] },
  { name: '字符串', tags: ['字符串', 'KMP 算法', 'Z 函数', 'Trie 字典树', 'AC 自动机', '后缀数组 SA', '后缀自动机 SAM', '回文自动机 PAM', 'Manacher 算法', '最小表示法', '字符串哈希'] },
  { name: '进阶技巧', tags: ['倍增', '整体二分', 'CDQ 分治', '启发式合并', '离线算法', '在线算法', 'Meet in the Middle', '根号分治', '并行二分'] }
] as const

const sourceOptions = ['黄海学院 OJ', '校内赛', '课程作业', '训练营', '洛谷', 'Codeforces', 'AtCoder', 'ICPC', 'CCPC', 'NOIP', '蓝桥杯']
const algorithmOptions = new Set<string>(algorithmGroups.flatMap((group) => [...group.tags]))
const currentYear = new Date().getFullYear()
const yearOptions = Array.from({ length: 41 }, (_, index) => currentYear + 1 - index)
const visible = ref(false)
const activeTab = ref('algorithm')
const keyword = ref('')
const draftAlgorithms = ref<string[]>([])
const draftYear = ref('')
const draftSource = ref('')

const draftTags = computed(() => [
  ...draftAlgorithms.value,
  ...(draftYear.value ? [draftYear.value] : []),
  ...(draftSource.value ? [draftSource.value] : [])
])

const filteredGroups = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return algorithmGroups
  return algorithmGroups
    .map((group) => ({ ...group, tags: group.tags.filter((tag) => tag.toLowerCase().includes(text)) }))
    .filter((group) => group.tags.length)
})

function open() {
  const incoming = [...new Set(props.modelValue.map((tag) => tag.trim()).filter(Boolean))]
  draftYear.value = incoming.find((tag) => /^\d{4}$/.test(tag)) || ''
  draftSource.value = incoming.find((tag) => sourceOptions.includes(tag)) || incoming.find((tag) => tag !== draftYear.value && !algorithmOptions.has(tag)) || ''
  draftAlgorithms.value = incoming.filter((tag) => tag !== draftYear.value && tag !== draftSource.value)
  activeTab.value = 'algorithm'
  keyword.value = ''
  visible.value = true
}

function toggleAlgorithm(tag: string) {
  draftAlgorithms.value = draftAlgorithms.value.includes(tag)
    ? draftAlgorithms.value.filter((item) => item !== tag)
    : [...draftAlgorithms.value, tag]
}

function removeTag(tag: string) {
  if (tag === draftYear.value) draftYear.value = ''
  else if (tag === draftSource.value) draftSource.value = ''
  else draftAlgorithms.value = draftAlgorithms.value.filter((item) => item !== tag)
}

function confirm() {
  emit('update:modelValue', draftTags.value)
  visible.value = false
}

</script>

<style scoped>
.tag-selector-field { width: 100%; display: flex; align-items: stretch; gap: 8px; }
.selected-preview { min-height: 32px; display: flex; align-items: center; flex-wrap: wrap; gap: 6px; flex: 1; padding: 5px 10px; border: 1px solid var(--border); border-radius: 4px; background: var(--surface-strong); cursor: pointer; }
.placeholder { color: var(--muted); font-size: 13px; }
.tag-dialog-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 18px; border-bottom: 1px solid var(--border); }
.tag-dialog-toolbar :deep(.el-tabs__header) { margin: 0; }
.tag-search { width: min(340px, 46%); }
.picked-section { display: grid; gap: 12px; min-height: 104px; padding: 20px 0; border-bottom: 1px solid var(--border); }
.picked-section > span { font-weight: 700; }
.picked-tags { display: flex; align-items: flex-start; flex-wrap: wrap; gap: 8px; }
.tag-scroll { max-height: 48vh; overflow-y: auto; padding-right: 8px; }
.algorithm-group { padding: 20px 0; border-bottom: 1px solid var(--border); }
.algorithm-group h3, .single-select-section h3 { margin: 0 0 14px; }
.tag-options { display: flex; flex-wrap: wrap; gap: 10px; }
.tag-options button { padding: 7px 12px; color: var(--muted); border: 1px solid var(--border); border-radius: 5px; background: var(--surface-strong); cursor: pointer; transition: .16s ease; }
.tag-options button:hover, .tag-options button.selected { color: var(--accent); border-color: var(--accent); background: color-mix(in srgb, var(--accent) 9%, var(--surface-strong)); }
.single-select-section { min-height: 260px; padding: 28px 2px; }
.single-select-section p { margin: -4px 0 24px; color: var(--muted); }
.single-select-section .el-select { width: min(440px, 100%); }
@media (max-width: 680px) { .tag-selector-field, .tag-dialog-toolbar { align-items: stretch; flex-direction: column; } .tag-search { width: 100%; margin-bottom: 12px; } }
</style>

<template>
<section class="panel announcements" aria-label="赛时公告">
<div class="heading"><h3>赛时公告</h3><el-button :loading="loading" @click="load">刷新</el-button></div>
<el-alert v-if="error" type="error" :title="error" :closable="false" />
<ol v-if="items.length" aria-live="polite"><li v-for="item in items" :key="item.id"><small>{{ formatDateTime(item.created_at) }} · {{ item.problem_id ? `题目 #${item.problem_id}` : '全场公告' }}</small><p>{{ item.body }}</p></li></ol>
<p v-else-if="!loading && !error">暂无公告</p>
<el-form v-if="canPublish" label-position="top" @submit.prevent="publish">
<el-form-item label="关联题目 ID（选填）"><el-input v-model="problem" inputmode="numeric" /></el-form-item>
<el-form-item label="公告正文"><el-input v-model="body" type="textarea" :rows="3" maxlength="4000" show-word-limit /></el-form-item>
<p>发布后保留历史；更正请另发公告。</p><el-button type="primary" native-type="submit" :loading="saving" :disabled="!body.trim()">发布公告</el-button>
</el-form></section>
</template>
<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { generatedApi } from '../../api/client'
import { eventPath, type EventKind } from '../../api/modules/events'
import { apiErrorMessage } from '../../api/errors'
import { formatDateTime } from '../../features/time'
const props = defineProps<{ kind: EventKind; eventId: number; canPublish?: boolean }>()
const items = ref<{id: number; problem_id?: number; body: string; created_at: string}[]>([])
const body=ref(''), problem=ref(''), error=ref(''), saving=ref(false), loading=ref(false)
let generation=0
async function load() {
 const current=++generation; loading.value=true
 try { const {data}=await generatedApi.listEventAnnouncements<typeof items.value>({path:eventPath(props.kind,props.eventId)}); if(current===generation) {items.value=data;error.value=''} }
 catch(e) {if(current===generation) error.value=apiErrorMessage(e)}
 finally {if(current===generation) loading.value=false}
}
async function publish() {
 if(problem.value && !/^[1-9]\d*$/.test(problem.value)) {error.value='请输入有效题目 ID';return}
 saving.value=true
 try {await generatedApi.publishEventAnnouncement({path:eventPath(props.kind,props.eventId),body:{body:body.value,problem_id:problem.value ? Number(problem.value) : null}});body.value='';problem.value='';await load()}
 catch(e) {error.value=apiErrorMessage(e)} finally {saving.value=false}
}
let timer: ReturnType<typeof setInterval>
onMounted(()=>{timer=setInterval(()=>{if(!document.hidden && !loading.value) void load()},15000)})
onUnmounted(()=>{clearInterval(timer);generation++})
watch(()=>[props.kind,props.eventId],()=>{items.value=[];void load()},{immediate:true})
</script>
<style scoped>
.announcements {margin:16px 0;border-left:4px solid var(--el-color-warning)}
.heading {display:flex;align-items:center;justify-content:space-between;gap:12px}
ol {padding-left:24px;max-height:320px;overflow:auto}
li p {white-space:pre-wrap;overflow-wrap:anywhere;margin-top:6px}
</style>

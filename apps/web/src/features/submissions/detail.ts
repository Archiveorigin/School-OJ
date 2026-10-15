import { apiErrorMessage } from '../../api/errors'
import { onBeforeUnmount, ref, shallowRef } from 'vue'
import { ElMessage } from 'element-plus'
import { generatedApi, type components } from '../../api/client'

export function useSubmissionDetail() {
  const visible = ref(false)
  const detail = shallowRef<components['schemas']['SubmissionDetail'] | null>(null)
  let sequence = 0
  let controller: AbortController | null = null
  async function open(submission: { id: number } | number) {
    const request = ++sequence
    controller?.abort()
    controller = new AbortController()
    visible.value = false
    detail.value = null
    try {
      const { data } = await generatedApi.getSubmission({ path: { id: typeof submission === 'number' ? submission : submission.id }, signal: controller.signal })
      if (request !== sequence) return
      detail.value = data
      visible.value = true
    } catch (error: any) {
      if (request === sequence) ElMessage.error(apiErrorMessage(error, '提交详情加载失败'))
    }
  }
  onBeforeUnmount(() => { sequence++; controller?.abort() })
  return { visible, detail, open }
}

<template>
  <el-dialog v-model="scannerVisible" title="扫码加入课程" width="min(520px, calc(100vw - 28px))" align-center @closed="stopCamera">
    <div class="scanner-content">
      <div class="scanner-frame" :class="{ active: scanning }">
        <video ref="videoRef" muted playsinline aria-label="课程二维码扫描画面"></video>
        <div v-if="!scanning" class="scanner-placeholder">
          <img src="/course.jpg" alt="课程加入" />
          <strong>扫描课程二维码</strong>
          <span>可启动摄像头，或选择二维码图片识别</span>
        </div>
        <span v-else class="scan-line" aria-hidden="true"></span>
      </div>
      <div class="scanner-actions">
        <el-button type="primary" :loading="startingCamera" @click="startCamera">启动摄像头</el-button>
        <el-upload
          action="#"
          accept="image/*"
          :auto-upload="false"
          :show-file-list="false"
          :on-change="scanImage"
        >
          <el-button>选择二维码图片</el-button>
        </el-upload>
      </div>
      <el-input v-model="scanCode" clearable placeholder="识别结果，也可粘贴课程邀请码" @keyup.enter="joinScannedCourse">
        <template #prepend>邀请码</template>
      </el-input>
      <p class="muted scanner-tip">二维码识别需要浏览器支持；若设备不支持，可直接粘贴或输入邀请码。</p>
    </div>
    <template #footer>
      <el-button @click="scannerVisible = false">取消</el-button>
      <el-button type="primary" :loading="joining" :disabled="!scanCode.trim()" @click="joinScannedCourse">确认加入</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="inviteVisible" title="邀请码加入课程" width="min(460px, calc(100vw - 28px))" align-center>
    <div class="invite-content">
      <img src="/course.jpg" alt="课程邀请码" />
      <div>
        <p>输入教师提供的课程邀请码，也兼容班级邀请码。</p>
        <el-input v-model="inviteCode" size="large" clearable placeholder="课程或班级邀请码" @keyup.enter="joinInvitedCourse" />
      </div>
    </div>
    <template #footer>
      <el-button @click="inviteVisible = false">取消</el-button>
      <el-button type="primary" :loading="joining" :disabled="!inviteCode.trim()" @click="joinInvitedCourse">加入课程</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { nextTick, onBeforeUnmount, ref } from 'vue'
import { client } from '../api/client'

const emit = defineEmits<{
  (event: 'joined', code: string): void
}>()

const scannerVisible = ref(false)
const inviteVisible = ref(false)
const inviteCode = ref('')
const scanCode = ref('')
const joining = ref(false)
const scanning = ref(false)
const startingCamera = ref(false)
const videoRef = ref<HTMLVideoElement | null>(null)
let mediaStream: MediaStream | null = null
let scanFrame = 0
let detector: any = null

function openScanner() {
  scanCode.value = ''
  scannerVisible.value = true
}

function openInvite(code = '') {
  inviteCode.value = code
  inviteVisible.value = true
}

async function joinScannedCourse() {
  if (await joinCourse(scanCode.value)) scannerVisible.value = false
}

async function joinInvitedCourse() {
  if (await joinCourse(inviteCode.value)) inviteVisible.value = false
}

async function joinCourse(rawCode: string) {
  const code = extractJoinCode(rawCode)
  if (!code) {
    ElMessage.warning('请输入有效的课程邀请码')
    return false
  }
  joining.value = true
  try {
    try {
      await client.post('/courses/join', { join_code: code })
    } catch (courseError: any) {
      if (courseError.response?.status !== 404) throw courseError
      await client.post('/classes/join', { join_code: code })
    }
    stopCamera()
    inviteCode.value = ''
    scanCode.value = ''
    ElMessage.success('已加入课程')
    emit('joined', code)
    return true
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '邀请码无效')
    return false
  } finally {
    joining.value = false
  }
}

function extractJoinCode(value: string) {
  const raw = value.trim()
  if (!raw) return ''
  try {
    const url = new URL(raw, window.location.origin)
    return (url.searchParams.get('join_code') || url.searchParams.get('code') || raw).trim()
  } catch {
    return raw
  }
}

function barcodeDetector() {
  const Detector = (window as any).BarcodeDetector
  if (!Detector) return null
  if (!detector) detector = new Detector({ formats: ['qr_code'] })
  return detector
}

async function startCamera() {
  const activeDetector = barcodeDetector()
  if (!activeDetector) {
    ElMessage.warning('当前浏览器不支持二维码识别，请改用邀请码加入')
    return
  }
  if (!navigator.mediaDevices?.getUserMedia) {
    ElMessage.warning('当前设备无法调用摄像头')
    return
  }
  startingCamera.value = true
  stopCamera()
  try {
    mediaStream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'environment' } },
      audio: false
    })
    await nextTick()
    if (!videoRef.value) return
    videoRef.value.srcObject = mediaStream
    await videoRef.value.play()
    scanning.value = true
    scanVideoFrame(activeDetector)
  } catch (err: any) {
    ElMessage.error(err.name === 'NotAllowedError' ? '摄像头权限未开启' : '摄像头启动失败')
    stopCamera()
  } finally {
    startingCamera.value = false
  }
}

function scanVideoFrame(activeDetector: any) {
  cancelAnimationFrame(scanFrame)
  const scan = async () => {
    if (!scanning.value || !videoRef.value) return
    try {
      if (videoRef.value.readyState >= 2) {
        const codes = await activeDetector.detect(videoRef.value)
        if (codes.length) {
          scanCode.value = extractJoinCode(codes[0].rawValue || '')
          stopCamera()
          ElMessage.success('课程二维码识别成功')
          return
        }
      }
    } catch {
      // Ignore a transient undecodable frame and continue scanning.
    }
    scanFrame = requestAnimationFrame(scan)
  }
  scanFrame = requestAnimationFrame(scan)
}

async function scanImage(uploadFile: any) {
  const activeDetector = barcodeDetector()
  if (!activeDetector) {
    ElMessage.warning('当前浏览器不支持二维码图片识别，请改用邀请码加入')
    return
  }
  const file = uploadFile.raw as File | undefined
  if (!file) return
  try {
    const image = await createImageBitmap(file)
    const codes = await activeDetector.detect(image)
    image.close()
    if (!codes.length) {
      ElMessage.warning('图片中未识别到课程二维码')
      return
    }
    scanCode.value = extractJoinCode(codes[0].rawValue || '')
    ElMessage.success('课程二维码识别成功')
  } catch {
    ElMessage.error('二维码图片读取失败')
  }
}

function stopCamera() {
  cancelAnimationFrame(scanFrame)
  scanFrame = 0
  scanning.value = false
  mediaStream?.getTracks().forEach((track) => track.stop())
  mediaStream = null
  if (videoRef.value) videoRef.value.srcObject = null
}

defineExpose({ openScanner, openInvite })
onBeforeUnmount(stopCamera)
</script>

<style scoped>
.scanner-content {
  display: grid;
  gap: 14px;
}

.scanner-frame {
  position: relative;
  display: grid;
  min-height: 286px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 16px;
  background: #071827;
}

.scanner-frame video {
  display: none;
  width: 100%;
  height: 286px;
  object-fit: cover;
}

.scanner-frame.active video {
  display: block;
}

.scanner-placeholder {
  display: grid;
  justify-items: center;
  gap: 8px;
  color: #fff;
  text-align: center;
}

.scanner-placeholder img,
.invite-content img {
  width: 82px;
  height: 92px;
  object-fit: cover;
  border-radius: 12px;
  background: #fff;
}

.scanner-placeholder span,
.scanner-tip {
  color: #94a3b8;
  font-size: 12px;
}

.scan-line {
  position: absolute;
  left: 12%;
  right: 12%;
  top: 50%;
  height: 2px;
  background: #38bdf8;
  box-shadow: 0 0 14px #38bdf8;
  animation: scan 2s ease-in-out infinite;
}

.scanner-actions {
  display: flex;
  justify-content: center;
  gap: 10px;
}

.scanner-tip,
.invite-content p {
  margin: 0;
}

.invite-content {
  display: grid;
  grid-template-columns: 92px 1fr;
  gap: 18px;
  align-items: center;
}

.invite-content p {
  margin-bottom: 12px;
  color: var(--muted);
  line-height: 1.7;
}

@keyframes scan {
  0%, 100% { transform: translateY(-90px); opacity: .55; }
  50% { transform: translateY(90px); opacity: 1; }
}

@media (max-width: 520px) {
  .scanner-actions,
  .invite-content {
    align-items: stretch;
    grid-template-columns: 1fr;
    flex-direction: column;
  }

  .invite-content img {
    justify-self: center;
  }
}
</style>

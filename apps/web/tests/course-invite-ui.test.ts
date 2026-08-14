import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const joinDialogSource = readFileSync(
  resolve(process.cwd(), 'src/components/CourseJoinDialogs.vue'),
  'utf8',
)
const myCoursesSource = readFileSync(
  resolve(process.cwd(), 'src/views/courses/MyCourses.vue'),
  'utf8',
)
const courseWorkspaceSource = readFileSync(
  resolve(process.cwd(), 'src/views/courses/CourseWorkspace.vue'),
  'utf8',
)
const packageSource = readFileSync(resolve(process.cwd(), 'package.json'), 'utf8')

describe('course invitation UI', () => {
  it('offers one invite-code-only course join flow', () => {
    expect(joinDialogSource.match(/<el-dialog\b/g)).toHaveLength(1)
    expect(joinDialogSource).toContain('placeholder="请输入课程邀请码"')
    expect(joinDialogSource).toContain("client.post('/courses/join', { join_code: code })")
    expect(joinDialogSource).toContain('response.data?.course_id')
    expect(joinDialogSource).toContain('defineExpose({ openInvite })')
    expect(joinDialogSource).not.toContain('/classes/join')
    expect(joinDialogSource).not.toContain('openScanner')
    expect(joinDialogSource).not.toContain('BarcodeDetector')
    expect(joinDialogSource).not.toContain('二维码')
  })

  it('uses the supplied course-add asset in both course entry surfaces', () => {
    expect(existsSync(resolve(process.cwd(), 'src/assets/course-add.svg'))).toBe(true)
    expect(joinDialogSource).toContain("import courseAddIcon from '../assets/course-add.svg'")
    expect(myCoursesSource).toContain("import courseAddIcon from '../../assets/course-add.svg'")
    expect(courseWorkspaceSource).toContain("import courseAddIcon from '../../assets/course-add.svg'")
    expect(myCoursesSource).toContain('<img :src="courseAddIcon" alt="添加课程" />')
    expect(courseWorkspaceSource).toContain('<img :src="courseAddIcon" alt="" />')
    expect(myCoursesSource).not.toContain('/course.jpg')
    expect(courseWorkspaceSource).not.toContain('/course.jpg')
  })

  it('removes scanner, generated QR and legacy QR-link entry points', () => {
    expect(myCoursesSource).toContain('joinDialogs?.openInvite()')
    expect(myCoursesSource).not.toContain('openScanner')
    expect(myCoursesSource).not.toContain('join_code')
    expect(myCoursesSource).not.toContain('二维码')

    expect(courseWorkspaceSource).toContain('joinDialogs?.openInvite()')
    expect(courseWorkspaceSource).toContain('<CourseJoinDialogs')
    expect(courseWorkspaceSource).toContain('router.push(`/my/courses/${joinedCourseID}`)')
    expect(courseWorkspaceSource).not.toContain('QrcodeVue')
    expect(courseWorkspaceSource).not.toContain('qrValue')
    expect(courseWorkspaceSource).not.toContain('二维码')
    expect(packageSource).not.toContain('qrcode.vue')
  })
})

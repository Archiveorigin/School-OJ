import type { User } from '../api/client'

// Display hints come from the current server session. Missing/unknown grants
// never fall back to inferring privileges from a role name or local checkbox.
export function hasCapability(user: User | null | undefined, policy: string): boolean {
  return Boolean(user && !user.account_deleted && user.permissions?.includes(policy))
}

export const capabilityNames: Record<string, string> = {
  'user:manage': '用户管理', 'permission:manage': '权限授予与撤销', 'audit:view': '审计日志',
  'event:review': '考试与比赛审核', 'problem:review': '题目审核', 'problem:author': '发起题目变更',
  'course:teach': '教学工作台', 'course:create': '创建课程', 'course:manage:any': '跨课程管理',
  'team:manage:any': '跨团队管理', 'problem:manage:any': '题目数据管理',
  'submission:read:any': '全站提交查看', 'submission:results:any': '完整评测结果查看',
  'ticket:manage:any': '工单管理', 'event:read:any': '全站场次查看', 'exam:participate': '参加课程考试',
  'course:manage': '课程教学操作', 'course:admin': '课程成员与设置', 'team:read': '团队内容查看',
  'team:manage': '团队成员管理', 'team:permissions': '团队权限设置', 'team:content': '组织团队比赛与题单',
}

export function capabilityRoleName(role: string): string {
  return ({ admin: '管理员', teacher: '教师', student: '学生', problem_author: '出题者' } as Record<string, string>)[role] || role
}

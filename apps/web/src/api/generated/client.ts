/* eslint-disable */
// 此文件由 scripts/generate-openapi-client.mjs 根据 apps/api/openapi/openapi.json 生成。
// 请勿手工修改；修改契约后运行 npm run api:generate。

import type { AxiosRequestConfig, AxiosResponse } from 'axios'
import { http } from '../http'

export interface components {
  schemas: {
    "JsonValue": string | number | number | boolean | Array<unknown> | {
    [key: string]: unknown
  }
    "JsonObject": {
    [key: string]: unknown
  }
    "ErrorResponse": {
    "error": string
    "ticket_route"?: string
    "exam_id"?: number
  }
    "User": {
    "id": number
    "email": string
    "name": string
    "role": "student" | "teacher" | "admin"
    "student_no"?: string
    "avatar_url"?: string
    "can_author"?: boolean
    "email_verified"?: boolean
    "account_deleted"?: boolean
    "created_at"?: string
    "updated_at"?: string
  }
    "AuthResponse": {
    "token": string
    "user": components['schemas']["User"]
  }
    "LoginRequest": {
    "email": string
    "password": string
  }
    "EmailCodeRequest": {
    "email": string
    "purpose": "register" | "password_reset"
  }
    "RegisterRequest": {
    "email": string
    "name": string
    "password": string
    "code": string
  }
    "PasswordResetRequest": {
    "email": string
    "password": string
    "code": string
  }
    "ProfileUpdateRequest": {
    "name"?: string
    "avatar_url"?: string
  }
    "PasswordChangeRequest": {
    "current_password": string
    "new_password": string
  }
    "EmailRebindRequest": {
    "email": string
    "code": string
  }
    "Problem": {
    "id": number
    "owner_id"?: number
    "team_id"?: number
    "display_code"?: string
    "title": string
    "statement": string
    "tags"?: {
    [key: string]: unknown
  }
    "difficulty"?: string
    "time_limit_ms": number
    "memory_limit_mb": number
    "output_limit_kb": number
    "manifest"?: {
    [key: string]: unknown
  }
    "progress_status"?: "unattempted" | "attempted" | "accepted"
    "points"?: number
    "points_awarded"?: boolean
  }
    "ProblemCatalogItem": components['schemas']["Problem"] & {
    "pass_rate": number
    "accepted_count": number
    "evaluated_count": number
  }
    "ProblemCatalogResponse": {
    "items": Array<components['schemas']["ProblemCatalogItem"]>
    "total": number
    "page": number
    "page_size": number
    "available_tags": Array<string>
  }
    "PreparedProblem": {
    "id": number
    "problem_id": number
    "owner_id": number
    "folder"?: string
    "difficulty"?: string
    "source"?: string
    "notes"?: string
    "archived"?: boolean
    "published_at"?: string
    "problem": components['schemas']["Problem"]
  }
    "Submission": {
    "id": number
    "user_id": number
    "problem_id": number
    "problem_version_id"?: number
    "assignment_id"?: number
    "exam_id"?: number
    "team_contest_id"?: number
    "team_problem_set_id"?: number
    "language": "c" | "cpp" | "python" | "java"
    "source_code"?: string
    "status": string
    "score": number
    "time_ms": number
    "memory_kb": number
    "message": string
    "created_at": string
    "updated_at": string
  }
    "SubmissionCreateRequest": {
    "problem_id": number
    "language": "c" | "cpp" | "python" | "java"
    "source_code": string
    "assignment_id"?: number
    "exam_id"?: number
    "team_contest_id"?: number
    "team_problem_set_id"?: number
  }
    "ProblemChangeTicketRequest": {
    "action": "create" | "replace" | "archive"
    "problem_id"?: number
    "target_scope"?: "public" | "prepared" | "team_problem_set"
    "team_problem_set_id"?: number
    "description": string
    "attachment": string
    "operation_mode"?: "overwrite"
  }
    "ProblemChangeTicket": {
    "id": number
    "requester_id": number
    "requester"?: components['schemas']["User"]
    "problem_id"?: number
    "problem"?: components['schemas']["Problem"]
    "action": "create" | "replace" | "archive"
    "status": "pending" | "processing" | "completed" | "rejected" | "cancelled"
    "target_scope": "public" | "prepared" | "team_problem_set"
    "description": string
    "attachment_name"?: string
    "resolution_note"?: string
    "created_at"?: string
    "updated_at"?: string
  }
    "Team": {
    "id": number
    "name": string
    "slug": string
    "owner_id": number
    "visibility"?: "private" | "public"
    "join_mode"?: "invitation" | "application" | "open"
    "description"?: string
    "announcement"?: string
    "icon_url"?: string
    "member_count"?: number
    "my_role"?: "owner" | "admin" | "member"
  }
    "TeamContest": {
    "id": number
    "team_id": number
    "title": string
    "description"?: string
    "starts_at"?: string
    "ends_at"?: string
    "duration_minutes": number
    "scoring_rule"?: "oi" | "ioi" | "acm"
    "freeze_enabled"?: boolean
    "freeze_duration_minutes"?: number
    "state"?: "draft" | "published" | "running" | "closed"
    "created_by": number
  }
    "TeamProblemSet": {
    "id": number
    "team_id": number
    "title": string
    "description"?: string
    "created_by": number
    "problem_count"?: number
    "can_edit"?: boolean
    "can_delete"?: boolean
  }
    "AuthorApplication": {
    "id": number
    "user_id": number
    "motivation": string
    "status": "pending" | "approved" | "rejected"
    "review_note"?: string
    "user"?: components['schemas']["User"]
    "created_at"?: string
    "updated_at"?: string
  }
    "AuthorApplicationEnvelope": {
    "application"?: components['schemas']["AuthorApplication"]
  }
    "PermissionDefinition": {
    "key": string
    "name": string
    "description": string
    "scope": string
  }
  }
}

export const apiOperations = {
  "login": { method: "POST", path: "/auth/login" },
  "sendEmailCode": { method: "POST", path: "/auth/send-code" },
  "register": { method: "POST", path: "/auth/register" },
  "resetPassword": { method: "POST", path: "/auth/password-reset" },
  "listProblems": { method: "GET", path: "/problems" },
  "problemChangeTicketOnly": { method: "POST", path: "/problems" },
  "listProblemCatalog": { method: "GET", path: "/problems/catalog" },
  "getProblemAsset": { method: "GET", path: "/problems/{id}/assets/{asset_path}" },
  "getProblem": { method: "GET", path: "/problems/{id}" },
  "problemChangeTicketOnlyViaProblemsId": { method: "PUT", path: "/problems/{id}" },
  "problemChangeTicketOnlyDeleteProblem": { method: "DELETE", path: "/problems/{id}" },
  "me": { method: "GET", path: "/me" },
  "activeExam": { method: "GET", path: "/me/active-exam" },
  "getProfile": { method: "GET", path: "/profile" },
  "updateProfile": { method: "PUT", path: "/profile" },
  "deleteProfile": { method: "DELETE", path: "/profile" },
  "updateProfilePassword": { method: "POST", path: "/profile/password" },
  "sendProfileEmailCode": { method: "POST", path: "/profile/email-code" },
  "rebindEmail": { method: "POST", path: "/profile/email" },
  "getMyAuthorApplication": { method: "GET", path: "/author-applications/me" },
  "listAuthorApplications": { method: "GET", path: "/author-applications" },
  "createAuthorApplication": { method: "POST", path: "/author-applications" },
  "createFeedback": { method: "POST", path: "/feedback" },
  "listTeams": { method: "GET", path: "/teams" },
  "createTeam": { method: "POST", path: "/teams" },
  "getTeam": { method: "GET", path: "/teams/{id}" },
  "updateTeam": { method: "PUT", path: "/teams/{id}" },
  "joinTeam": { method: "POST", path: "/teams/{id}/join" },
  "leaveTeam": { method: "POST", path: "/teams/{id}/leave" },
  "listTeamMembers": { method: "GET", path: "/teams/{id}/members" },
  "updateTeamMember": { method: "PUT", path: "/teams/{id}/members/{user_id}" },
  "removeTeamMember": { method: "DELETE", path: "/teams/{id}/members/{user_id}" },
  "listTeamApplications": { method: "GET", path: "/teams/{id}/applications" },
  "reviewTeamApplication": { method: "PUT", path: "/teams/{id}/applications/{application_id}" },
  "listTeamContests": { method: "GET", path: "/teams/{id}/contests" },
  "createTeamContest": { method: "POST", path: "/teams/{id}/contests" },
  "getTeamContest": { method: "GET", path: "/teams/{id}/contests/{contest_id}" },
  "updateTeamContest": { method: "PUT", path: "/teams/{id}/contests/{contest_id}" },
  "deleteTeamContest": { method: "DELETE", path: "/teams/{id}/contests/{contest_id}" },
  "publishTeamContest": { method: "POST", path: "/teams/{id}/contests/{contest_id}/publish" },
  "addTeamContestProblem": { method: "POST", path: "/teams/{id}/contests/{contest_id}/problems" },
  "removeTeamContestProblem": { method: "DELETE", path: "/teams/{id}/contests/{contest_id}/problems/{problem_id}" },
  "listTeamContestSubmissions": { method: "GET", path: "/teams/{id}/contests/{contest_id}/submissions" },
  "createTeamContestSubmission": { method: "POST", path: "/teams/{id}/contests/{contest_id}/submissions" },
  "teamContestRanking": { method: "GET", path: "/teams/{id}/contests/{contest_id}/ranking" },
  "getTeamContestViaContestsContestId": { method: "GET", path: "/contests/{contest_id}" },
  "updateTeamContestViaContestsContestId": { method: "PUT", path: "/contests/{contest_id}" },
  "deleteTeamContestViaContestsContestId": { method: "DELETE", path: "/contests/{contest_id}" },
  "publishTeamContestViaContestsContestIdPublish": { method: "POST", path: "/contests/{contest_id}/publish" },
  "addTeamContestProblemViaContestsContestIdProblems": { method: "POST", path: "/contests/{contest_id}/problems" },
  "removeTeamContestProblemViaContestsContestIdProblemsProblemId": { method: "DELETE", path: "/contests/{contest_id}/problems/{problem_id}" },
  "listTeamContestSubmissionsViaContestsContestIdSubmissions": { method: "GET", path: "/contests/{contest_id}/submissions" },
  "createTeamContestSubmissionViaContestsContestIdSubmissions": { method: "POST", path: "/contests/{contest_id}/submissions" },
  "teamContestRankingViaContestsContestIdRanking": { method: "GET", path: "/contests/{contest_id}/ranking" },
  "listTeamProblemSets": { method: "GET", path: "/teams/{id}/problem-sets" },
  "createTeamProblemSet": { method: "POST", path: "/teams/{id}/problem-sets" },
  "getTeamProblemSet": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}" },
  "updateTeamProblemSet": { method: "PUT", path: "/teams/{id}/problem-sets/{set_id}" },
  "deleteTeamProblemSet": { method: "DELETE", path: "/teams/{id}/problem-sets/{set_id}" },
  "addTeamProblemSetProblem": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/problems" },
  "removeTeamProblemSetProblem": { method: "DELETE", path: "/teams/{id}/problem-sets/{set_id}/problems/{problem_id}" },
  "listTeamProblemSetSubmissions": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}/submissions" },
  "createTeamProblemSetSubmission": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/submissions" },
  "listTeamDiscussions": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}/discussions" },
  "createTeamDiscussion": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/discussions" },
  "getTeamProblemSetViaProblemSetsSetId": { method: "GET", path: "/problem-sets/{set_id}" },
  "updateTeamProblemSetViaProblemSetsSetId": { method: "PUT", path: "/problem-sets/{set_id}" },
  "deleteTeamProblemSetViaProblemSetsSetId": { method: "DELETE", path: "/problem-sets/{set_id}" },
  "addTeamProblemSetProblemViaProblemSetsSetIdProblems": { method: "POST", path: "/problem-sets/{set_id}/problems" },
  "removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId": { method: "DELETE", path: "/problem-sets/{set_id}/problems/{problem_id}" },
  "listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions": { method: "GET", path: "/problem-sets/{set_id}/submissions" },
  "createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions": { method: "POST", path: "/problem-sets/{set_id}/submissions" },
  "listTeamDiscussionsViaProblemSetsSetIdDiscussions": { method: "GET", path: "/problem-sets/{set_id}/discussions" },
  "createTeamDiscussionViaProblemSetsSetIdDiscussions": { method: "POST", path: "/problem-sets/{set_id}/discussions" },
  "listCourses": { method: "GET", path: "/courses" },
  "createCourse": { method: "POST", path: "/courses" },
  "updateCourse": { method: "PUT", path: "/courses/{id}" },
  "archiveCourse": { method: "DELETE", path: "/courses/{id}" },
  "archiveCourseViaCoursesIdArchive": { method: "POST", path: "/courses/{id}/archive" },
  "createClass": { method: "POST", path: "/courses/{id}/classes" },
  "listCourseMembers": { method: "GET", path: "/courses/{id}/members" },
  "addCourseMember": { method: "POST", path: "/courses/{id}/members" },
  "removeCourseMember": { method: "DELETE", path: "/courses/{id}/members/{user_id}" },
  "previewCourseByCode": { method: "GET", path: "/courses/preview" },
  "joinCourseByCode": { method: "POST", path: "/courses/join" },
  "leaveCourse": { method: "POST", path: "/courses/{id}/leave" },
  "listCourseStudents": { method: "GET", path: "/courses/{id}/students" },
  "listClasses": { method: "GET", path: "/classes" },
  "previewClassJoin": { method: "GET", path: "/classes/join-preview" },
  "myClasses": { method: "GET", path: "/me/classes" },
  "joinClassByCode": { method: "POST", path: "/classes/join" },
  "joinClass": { method: "POST", path: "/classes/{id}/join" },
  "leaveClass": { method: "POST", path: "/classes/{id}/leave" },
  "updateClass": { method: "PUT", path: "/classes/{id}" },
  "archiveClass": { method: "DELETE", path: "/classes/{id}" },
  "archiveClassViaClassesIdArchive": { method: "POST", path: "/classes/{id}/archive" },
  "listClassStudents": { method: "GET", path: "/classes/{id}/students" },
  "removeClassStudent": { method: "DELETE", path: "/classes/{id}/students/{user_id}" },
  "importClassStudents": { method: "POST", path: "/classes/{id}/students/import" },
  "problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex": { method: "PUT", path: "/problem-test-uploads/{upload_id}/{chunk_index}" },
  "parseMarkdownBatch": { method: "POST", path: "/problems/parse-markdown" },
  "problemChangeTicketOnlyViaProblemsUpload": { method: "POST", path: "/problems/upload" },
  "listProblemTests": { method: "GET", path: "/problems/{id}/tests" },
  "downloadProblemTests": { method: "GET", path: "/problems/{id}/tests/download" },
  "downloadProblemTestFile": { method: "GET", path: "/problems/{id}/tests/file/{file_path}" },
  "listPreparedProblems": { method: "GET", path: "/prepared-problems" },
  "problemChangeTicketOnlyViaPreparedProblems": { method: "POST", path: "/prepared-problems" },
  "problemChangeTicketOnlyViaPreparedProblemsUpload": { method: "POST", path: "/prepared-problems/upload" },
  "getPreparedProblem": { method: "GET", path: "/prepared-problems/{id}" },
  "problemChangeTicketOnlyViaPreparedProblemsId": { method: "PUT", path: "/prepared-problems/{id}" },
  "problemChangeTicketOnlyViaPreparedProblemsIdPublish": { method: "POST", path: "/prepared-problems/{id}/publish" },
  "listProblemChangeTickets": { method: "GET", path: "/problem-change-tickets" },
  "createProblemChangeTicket": { method: "POST", path: "/problem-change-tickets" },
  "listMyProblemChangeTickets": { method: "GET", path: "/problem-change-tickets/mine" },
  "listProblemChangeTicketEligibleProblems": { method: "GET", path: "/problem-change-tickets/eligible-problems" },
  "getProblemChangeTicket": { method: "GET", path: "/problem-change-tickets/{id}" },
  "downloadProblemChangeTicketAttachment": { method: "GET", path: "/problem-change-tickets/{id}/attachment" },
  "cancelProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/cancel" },
  "rejectProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/reject" },
  "applyProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/apply" },
  "listAssignments": { method: "GET", path: "/assignments" },
  "createAssignment": { method: "POST", path: "/assignments" },
  "getAssignment": { method: "GET", path: "/assignments/{id}" },
  "deleteAssignment": { method: "DELETE", path: "/assignments/{id}" },
  "assignmentReport": { method: "GET", path: "/assignments/{id}/report" },
  "listExams": { method: "GET", path: "/exams" },
  "createExam": { method: "POST", path: "/exams" },
  "getExam": { method: "GET", path: "/exams/{id}" },
  "deleteExam": { method: "DELETE", path: "/exams/{id}" },
  "finishExam": { method: "POST", path: "/exams/{id}/finish" },
  "exportExamReport": { method: "GET", path: "/exams/{id}/report/export" },
  "examReport": { method: "GET", path: "/exams/{id}/report" },
  "examRanking": { method: "GET", path: "/exams/{id}/ranking" },
  "judgeManualExamSubmission": { method: "POST", path: "/exams/{id}/submissions/{submission_id}/judge" },
  "gradeManualExamSubmission": { method: "PUT", path: "/exams/{id}/submissions/{submission_id}/grade" },
  "listSubmissions": { method: "GET", path: "/submissions" },
  "createSubmission": { method: "POST", path: "/submissions" },
  "latestSubmissions": { method: "GET", path: "/submissions/latest" },
  "getSubmission": { method: "GET", path: "/submissions/{id}" },
  "submissionEvents": { method: "GET", path: "/submissions/{id}/events" },
  "listPlagiarismJobs": { method: "GET", path: "/plagiarism/jobs" },
  "createPlagiarismJob": { method: "POST", path: "/plagiarism/jobs" },
  "listAuditLogs": { method: "GET", path: "/audit-logs" },
  "listUsers": { method: "GET", path: "/users" },
  "createUser": { method: "POST", path: "/users" },
  "listUserPermissionDefinitions": { method: "GET", path: "/permissions" },
  "getUser": { method: "GET", path: "/users/{id}" },
  "updateUser": { method: "PUT", path: "/users/{id}" },
  "deleteUser": { method: "DELETE", path: "/users/{id}" },
  "updateUserPermission": { method: "PUT", path: "/users/{id}/permissions/{permission}" },
  "resetUserPassword": { method: "POST", path: "/users/{id}/reset-password" },
  "reviewAuthorApplication": { method: "PUT", path: "/author-applications/{id}/review" },
  "listProblemAuthors": { method: "GET", path: "/problem-authors" },
  "removeProblemAuthor": { method: "DELETE", path: "/problem-authors/{id}" },
  "listMyProblemReviews": { method: "GET", path: "/problem-reviews/mine" },
  "listProblemReviews": { method: "GET", path: "/problem-reviews" },
  "problemChangeTicketOnlyViaProblemReviewsIdReview": { method: "PUT", path: "/problem-reviews/{id}/review" },
  "problemChangeTicketOnlyViaProblemReviewsIdWithdraw": { method: "PUT", path: "/problem-reviews/{id}/withdraw" },
} as const

export type ApiOperationId = keyof typeof apiOperations
export type ApiPathValue = string | number | boolean
export type ApiQueryValue = ApiPathValue | null | undefined | ApiPathValue[]

export interface ApiRequestOptions<TBody = unknown>
  extends Omit<AxiosRequestConfig, 'method' | 'url' | 'params' | 'data'> {
  path?: Record<string, ApiPathValue>
  query?: Record<string, ApiQueryValue>
  body?: TBody
}

function resolvePath(template: string, values: Record<string, ApiPathValue> = {}) {
  return template.replace(/\{([^}]+)\}/g, (_, key: string) => {
    const value = values[key]
    if (value === undefined || value === null || value === '') {
      throw new Error(`Missing OpenAPI path parameter: ${key}`)
    }
    return encodeURIComponent(String(value))
  })
}

export async function requestOperation<TResponse = unknown, TBody = unknown>(
  operationId: ApiOperationId,
  options: ApiRequestOptions<TBody> = {},
): Promise<AxiosResponse<TResponse>> {
  const definition = apiOperations[operationId]
  const { path, query, body, ...config } = options
  return http.request<TResponse>({
    ...config,
    method: definition.method,
    url: resolvePath(definition.path, path),
    params: query,
    data: body,
  })
}

export const generatedApi = {
  "login": (options: ApiRequestOptions<components['schemas']["LoginRequest"]> = {}) =>
    requestOperation<components['schemas']["AuthResponse"], components['schemas']["LoginRequest"]>("login", options),
  "sendEmailCode": (options: ApiRequestOptions<components['schemas']["EmailCodeRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["EmailCodeRequest"]>("sendEmailCode", options),
  "register": (options: ApiRequestOptions<components['schemas']["RegisterRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["RegisterRequest"]>("register", options),
  "resetPassword": (options: ApiRequestOptions<components['schemas']["PasswordResetRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["PasswordResetRequest"]>("resetPassword", options),
  "listProblems": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["Problem"]>, never>("listProblems", options),
  "problemChangeTicketOnly": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnly", options),
  "listProblemCatalog": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["ProblemCatalogResponse"], never>("listProblemCatalog", options),
  "getProblemAsset": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("getProblemAsset", options),
  "getProblem": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["Problem"], never>("getProblem", options),
  "problemChangeTicketOnlyViaProblemsId": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemsId", options),
  "problemChangeTicketOnlyDeleteProblem": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("problemChangeTicketOnlyDeleteProblem", options),
  "me": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["User"], never>("me", options),
  "activeExam": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("activeExam", options),
  "getProfile": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["User"], never>("getProfile", options),
  "updateProfile": (options: ApiRequestOptions<components['schemas']["ProfileUpdateRequest"]> = {}) =>
    requestOperation<components['schemas']["User"], components['schemas']["ProfileUpdateRequest"]>("updateProfile", options),
  "deleteProfile": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["User"], never>("deleteProfile", options),
  "updateProfilePassword": (options: ApiRequestOptions<components['schemas']["PasswordChangeRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["PasswordChangeRequest"]>("updateProfilePassword", options),
  "sendProfileEmailCode": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("sendProfileEmailCode", options),
  "rebindEmail": (options: ApiRequestOptions<components['schemas']["EmailRebindRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["EmailRebindRequest"]>("rebindEmail", options),
  "getMyAuthorApplication": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["AuthorApplicationEnvelope"], never>("getMyAuthorApplication", options),
  "listAuthorApplications": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["AuthorApplication"]>, never>("listAuthorApplications", options),
  "createAuthorApplication": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["AuthorApplication"], components['schemas']["JsonObject"]>("createAuthorApplication", options),
  "createFeedback": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createFeedback", options),
  "listTeams": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["Team"]>, never>("listTeams", options),
  "createTeam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["Team"], components['schemas']["JsonObject"]>("createTeam", options),
  "getTeam": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["Team"], never>("getTeam", options),
  "updateTeam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["Team"], components['schemas']["JsonObject"]>("updateTeam", options),
  "joinTeam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("joinTeam", options),
  "leaveTeam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("leaveTeam", options),
  "listTeamMembers": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamMembers", options),
  "updateTeamMember": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("updateTeamMember", options),
  "removeTeamMember": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeTeamMember", options),
  "listTeamApplications": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamApplications", options),
  "reviewTeamApplication": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("reviewTeamApplication", options),
  "listTeamContests": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["TeamContest"]>, never>("listTeamContests", options),
  "createTeamContest": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createTeamContest", options),
  "getTeamContest": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamContest"], never>("getTeamContest", options),
  "updateTeamContest": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamContest"], components['schemas']["JsonObject"]>("updateTeamContest", options),
  "deleteTeamContest": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamContest"], never>("deleteTeamContest", options),
  "publishTeamContest": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamContest"], components['schemas']["JsonObject"]>("publishTeamContest", options),
  "addTeamContestProblem": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("addTeamContestProblem", options),
  "removeTeamContestProblem": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeTeamContestProblem", options),
  "listTeamContestSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamContestSubmissions", options),
  "createTeamContestSubmission": (options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["SubmissionCreateRequest"]>("createTeamContestSubmission", options),
  "teamContestRanking": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("teamContestRanking", options),
  "getTeamContestViaContestsContestId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamContest"], never>("getTeamContestViaContestsContestId", options),
  "updateTeamContestViaContestsContestId": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamContest"], components['schemas']["JsonObject"]>("updateTeamContestViaContestsContestId", options),
  "deleteTeamContestViaContestsContestId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamContest"], never>("deleteTeamContestViaContestsContestId", options),
  "publishTeamContestViaContestsContestIdPublish": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamContest"], components['schemas']["JsonObject"]>("publishTeamContestViaContestsContestIdPublish", options),
  "addTeamContestProblemViaContestsContestIdProblems": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("addTeamContestProblemViaContestsContestIdProblems", options),
  "removeTeamContestProblemViaContestsContestIdProblemsProblemId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeTeamContestProblemViaContestsContestIdProblemsProblemId", options),
  "listTeamContestSubmissionsViaContestsContestIdSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamContestSubmissionsViaContestsContestIdSubmissions", options),
  "createTeamContestSubmissionViaContestsContestIdSubmissions": (options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["SubmissionCreateRequest"]>("createTeamContestSubmissionViaContestsContestIdSubmissions", options),
  "teamContestRankingViaContestsContestIdRanking": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("teamContestRankingViaContestsContestIdRanking", options),
  "listTeamProblemSets": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["TeamProblemSet"]>, never>("listTeamProblemSets", options),
  "createTeamProblemSet": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createTeamProblemSet", options),
  "getTeamProblemSet": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], never>("getTeamProblemSet", options),
  "updateTeamProblemSet": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], components['schemas']["JsonObject"]>("updateTeamProblemSet", options),
  "deleteTeamProblemSet": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], never>("deleteTeamProblemSet", options),
  "addTeamProblemSetProblem": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("addTeamProblemSetProblem", options),
  "removeTeamProblemSetProblem": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeTeamProblemSetProblem", options),
  "listTeamProblemSetSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamProblemSetSubmissions", options),
  "createTeamProblemSetSubmission": (options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["SubmissionCreateRequest"]>("createTeamProblemSetSubmission", options),
  "listTeamDiscussions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamDiscussions", options),
  "createTeamDiscussion": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createTeamDiscussion", options),
  "getTeamProblemSetViaProblemSetsSetId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], never>("getTeamProblemSetViaProblemSetsSetId", options),
  "updateTeamProblemSetViaProblemSetsSetId": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], components['schemas']["JsonObject"]>("updateTeamProblemSetViaProblemSetsSetId", options),
  "deleteTeamProblemSetViaProblemSetsSetId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["TeamProblemSet"], never>("deleteTeamProblemSetViaProblemSetsSetId", options),
  "addTeamProblemSetProblemViaProblemSetsSetIdProblems": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("addTeamProblemSetProblemViaProblemSetsSetIdProblems", options),
  "removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId", options),
  "listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions", options),
  "createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions": (options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["SubmissionCreateRequest"]>("createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions", options),
  "listTeamDiscussionsViaProblemSetsSetIdDiscussions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listTeamDiscussionsViaProblemSetsSetIdDiscussions", options),
  "createTeamDiscussionViaProblemSetsSetIdDiscussions": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createTeamDiscussionViaProblemSetsSetIdDiscussions", options),
  "listCourses": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listCourses", options),
  "createCourse": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createCourse", options),
  "updateCourse": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("updateCourse", options),
  "archiveCourse": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("archiveCourse", options),
  "archiveCourseViaCoursesIdArchive": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("archiveCourseViaCoursesIdArchive", options),
  "createClass": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createClass", options),
  "listCourseMembers": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listCourseMembers", options),
  "addCourseMember": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("addCourseMember", options),
  "removeCourseMember": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeCourseMember", options),
  "previewCourseByCode": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("previewCourseByCode", options),
  "joinCourseByCode": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("joinCourseByCode", options),
  "leaveCourse": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("leaveCourse", options),
  "listCourseStudents": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listCourseStudents", options),
  "listClasses": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listClasses", options),
  "previewClassJoin": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("previewClassJoin", options),
  "myClasses": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("myClasses", options),
  "joinClassByCode": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("joinClassByCode", options),
  "joinClass": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("joinClass", options),
  "leaveClass": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("leaveClass", options),
  "updateClass": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("updateClass", options),
  "archiveClass": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("archiveClass", options),
  "archiveClassViaClassesIdArchive": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("archiveClassViaClassesIdArchive", options),
  "listClassStudents": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listClassStudents", options),
  "removeClassStudent": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeClassStudent", options),
  "importClassStudents": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("importClassStudents", options),
  "problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex", options),
  "parseMarkdownBatch": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("parseMarkdownBatch", options),
  "problemChangeTicketOnlyViaProblemsUpload": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemsUpload", options),
  "listProblemTests": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listProblemTests", options),
  "downloadProblemTests": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("downloadProblemTests", options),
  "downloadProblemTestFile": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("downloadProblemTestFile", options),
  "listPreparedProblems": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["PreparedProblem"]>, never>("listPreparedProblems", options),
  "problemChangeTicketOnlyViaPreparedProblems": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblems", options),
  "problemChangeTicketOnlyViaPreparedProblemsUpload": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsUpload", options),
  "getPreparedProblem": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("getPreparedProblem", options),
  "problemChangeTicketOnlyViaPreparedProblemsId": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsId", options),
  "problemChangeTicketOnlyViaPreparedProblemsIdPublish": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsIdPublish", options),
  "listProblemChangeTickets": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["ProblemChangeTicket"]>, never>("listProblemChangeTickets", options),
  "createProblemChangeTicket": (options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["ProblemChangeTicketRequest"]>("createProblemChangeTicket", options),
  "listMyProblemChangeTickets": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["ProblemChangeTicket"]>, never>("listMyProblemChangeTickets", options),
  "listProblemChangeTicketEligibleProblems": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["Problem"]>, never>("listProblemChangeTicketEligibleProblems", options),
  "getProblemChangeTicket": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["ProblemChangeTicket"], never>("getProblemChangeTicket", options),
  "downloadProblemChangeTicketAttachment": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("downloadProblemChangeTicketAttachment", options),
  "cancelProblemChangeTicket": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["ProblemChangeTicket"], components['schemas']["JsonObject"]>("cancelProblemChangeTicket", options),
  "rejectProblemChangeTicket": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["ProblemChangeTicket"], components['schemas']["JsonObject"]>("rejectProblemChangeTicket", options),
  "applyProblemChangeTicket": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["ProblemChangeTicket"], components['schemas']["JsonObject"]>("applyProblemChangeTicket", options),
  "listAssignments": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listAssignments", options),
  "createAssignment": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createAssignment", options),
  "getAssignment": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("getAssignment", options),
  "deleteAssignment": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("deleteAssignment", options),
  "assignmentReport": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("assignmentReport", options),
  "listExams": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listExams", options),
  "createExam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createExam", options),
  "getExam": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("getExam", options),
  "deleteExam": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("deleteExam", options),
  "finishExam": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("finishExam", options),
  "exportExamReport": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("exportExamReport", options),
  "examReport": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("examReport", options),
  "examRanking": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("examRanking", options),
  "judgeManualExamSubmission": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("judgeManualExamSubmission", options),
  "gradeManualExamSubmission": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("gradeManualExamSubmission", options),
  "listSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["Submission"]>, never>("listSubmissions", options),
  "createSubmission": (options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<components['schemas']["Submission"], components['schemas']["SubmissionCreateRequest"]>("createSubmission", options),
  "latestSubmissions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["Submission"]>, never>("latestSubmissions", options),
  "getSubmission": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["Submission"], never>("getSubmission", options),
  "submissionEvents": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("submissionEvents", options),
  "listPlagiarismJobs": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listPlagiarismJobs", options),
  "createPlagiarismJob": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createPlagiarismJob", options),
  "listAuditLogs": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listAuditLogs", options),
  "listUsers": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["User"]>, never>("listUsers", options),
  "createUser": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("createUser", options),
  "listUserPermissionDefinitions": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<Array<components['schemas']["PermissionDefinition"]>, never>("listUserPermissionDefinitions", options),
  "getUser": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["User"], never>("getUser", options),
  "updateUser": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("updateUser", options),
  "deleteUser": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("deleteUser", options),
  "updateUserPermission": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("updateUserPermission", options),
  "resetUserPassword": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("resetUserPassword", options),
  "reviewAuthorApplication": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["AuthorApplication"], components['schemas']["JsonObject"]>("reviewAuthorApplication", options),
  "listProblemAuthors": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listProblemAuthors", options),
  "removeProblemAuthor": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("removeProblemAuthor", options),
  "listMyProblemReviews": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listMyProblemReviews", options),
  "listProblemReviews": (options: ApiRequestOptions<never> = {}) =>
    requestOperation<components['schemas']["JsonObject"], never>("listProblemReviews", options),
  "problemChangeTicketOnlyViaProblemReviewsIdReview": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemReviewsIdReview", options),
  "problemChangeTicketOnlyViaProblemReviewsIdWithdraw": (options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<components['schemas']["JsonObject"], components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemReviewsIdWithdraw", options),
}

// 兼容现有页面的 Axios 调用；新代码优先使用 generatedApi 的 operationId 方法。
export const client = http

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
    "code"?: string
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
    "is_public"?: boolean
    "team_contest_id"?: (number) | null
    "problem_set_id"?: (number) | null
    "language": "c" | "cpp" | "python" | "java"
    "source_code"?: string
    "status": string
    "score": number
    "time_ms": number
    "memory_kb": number
    "message": string
    "created_at": string
    "updated_at": string
    "user_name"?: string
    "problem_code"?: string
    "problem_title"?: string
    "manual_score"?: (number) | null
    "manual_graded_at"?: (string) | null
  }
    "SubmissionCreateRequest": {
    "problem_id": number
    "language": "c" | "cpp" | "python" | "java"
    "source_code": string
    "assignment_id"?: number
    "exam_id"?: number
    "is_public"?: boolean
  }
    "ProblemChangeTicketRequest": {
    "action": "create" | "replace" | "archive" | "suggest"
    "description": string
    "problem_id"?: number
    "target_scope"?: "public" | "prepared" | "team_problem_set"
    "team_problem_set_id"?: number
    "draft_title"?: string
    "draft_statement"?: string
    "revision"?: number
    "retain_attachment_ids"?: Array<number>
    "attachments"?: Array<string>
    "attachment"?: string
  }
    "ProblemChangeTicket": {
    "id": number
    "requester_id": number
    "requester": {
    "name": string
  }
    "problem"?: components['schemas']["TicketProblem"]
    "action": "create" | "replace" | "archive" | "suggest"
    "status": "pending" | "processing" | "completed" | "rejected" | "cancelled"
    "target_scope": "public" | "prepared" | "team_problem_set"
    "description": string
    "resolution_note"?: string
    "draft_title"?: string
    "draft_statement"?: string
    "revision": number
    "attachments"?: Array<components['schemas']["TicketAttachment"]>
    "attachment_name"?: string
    "attachment_count": number
    "impact_summary"?: {
    "future_exams"?: number
    "pinned_exams"?: number
    "future_contests"?: number
    "pinned_contests"?: number
    "historical_submissions"?: number
  }
    "problem_id"?: (number) | null
    "exam_id"?: (number) | null
    "contest_id"?: (number) | null
    "problem_version_id"?: (number) | null
    "current_version_id"?: (number) | null
    "applied_version_id"?: (number) | null
    "team_problem_set_id"?: (number) | null
    "can_edit": boolean
    "can_manage": boolean
    "can_apply": boolean
    "can_resolve": boolean
    "is_public": boolean
    "attachments_private": boolean
    "created_at": string
    "updated_at": string
    "processed_at"?: (string) | null
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
    "join_code"?: string
    "can_manage"?: boolean
    "can_edit_permissions"?: boolean
    "joined"?: boolean
    "application_status"?: string
    "contest_permission"?: "all" | "admin" | "owner"
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
    "icon_url"?: string
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
    "CaseSummary": ({
    "passed": number
    "total": number
  }) | null
    "SubmissionResult": {
    "case_name": string
    "status": string
    "time_ms": number
    "memory_kb": number
    "message": string
  }
    "SubmissionDetail": {
    "submission": components['schemas']["Submission"]
    "results": (Array<components['schemas']["SubmissionResult"]>) | null
    "case_summary": components['schemas']["CaseSummary"]
  }
    "Exam": {
    "id": number
    "course_id": number
    "title": string
    "description": string
    "starts_at": (string) | null
    "ends_at": (string) | null
    "scoring_rule": "oi" | "ioi" | "acm"
    "freeze_enabled": boolean
    "freeze_duration_minutes": number
    "settings": components['schemas']["JsonObject"]
  }
    "ExamProblem": {
    "problem_id": number
    "problem": components['schemas']["Problem"]
    "label": string
    "score": number
  }
    "ExamDetail": {
    "exam": components['schemas']["Exam"]
    "problems": Array<components['schemas']["ExamProblem"]>
    "now": string
    "closed": boolean
    "not_started": boolean
    "can_submit": boolean
    "can_manage": boolean
    "manual_review": boolean
    "ranking_visible": boolean
    "finished_at": (string) | null
  }
    "ExamEnterResponse": {
    "exam_id": number
    "entered_at": string
    "active": boolean
  }
    "ExamActivityEvent": {
    "id": string
    "event": "window_leave" | "window_return" | "devtools_shortcut"
    "client_at": string
  }
    "ExamActivityRequest": {
    "events": Array<components['schemas']["ExamActivityEvent"]>
  }
    "ExamActivity": {
    "id": number
    "exam_id": number
    "user_id": number
    "user_name": string
    "event_id": string
    "event": "enter" | "finish" | "login" | "window_leave" | "window_return" | "devtools_shortcut" | "access_blocked"
    "client_at": (string) | null
    "ip": string
    "meta": ({
    [key: string]: unknown
  }) | null
    "created_at": string
  }
    "ExamActivityPage": {
    "items": Array<components['schemas']["ExamActivity"]>
    "total": number
    "page": number
    "page_size": number
    "exam_title": string
  }
    "TeamJoinPreview": {
    "id": number
    "name": string
    "slug": string
    "icon_url": string
    "visibility": "private" | "public"
    "join_mode": "invitation" | "application" | "open"
    "joined": boolean
    "application_status": string
  }
    "TeamJoinRequest": {
    "join_code"?: string
    "message"?: string
  }
    "TeamJoinResponse": {
    "joined": boolean
    "application_status"?: string
  }
    "TicketAttachment": {
    "id": number
    "name": string
    "content_type": string
    "size": number
    "created_at"?: string
  }
    "TicketProblem": {
    "id": number
    "title": string
    "display_code"?: string
    "difficulty"?: string
  }
    "TicketDecision": {
    "revision": number
    "note"?: string
  }
    "TicketApply": {
    "revision": number
    "expected_version_id"?: number
    "package"?: string
    "operation_mode"?: "create" | "overwrite"
    "resolution_note"?: string
    "difficulty"?: string
    "tags"?: string
  }
    "TicketPage": {
    "items": Array<components['schemas']["ProblemChangeTicket"]>
    "total": number
    "page": number
    "page_size": number
  }
    "TeamProblemSetInput": {
    "title": string
    "description"?: string
    "icon_url"?: string
  }
    "TeamProblemSelectionRequest": {
    "problem_ids"?: Array<number>
    "problem_id"?: number
    "problem_code"?: string
    "label"?: string
    "sort_order"?: number
  }
  }
}

export const apiOperations = {
  "login": { method: "POST", path: "/auth/login", segments: [] },
  "sendEmailCode": { method: "POST", path: "/auth/send-code", segments: [] },
  "register": { method: "POST", path: "/auth/register", segments: [] },
  "resetPassword": { method: "POST", path: "/auth/password-reset", segments: [] },
  "listProblems": { method: "GET", path: "/problems", segments: [] },
  "problemChangeTicketOnly": { method: "POST", path: "/problems", segments: [] },
  "listProblemCatalog": { method: "GET", path: "/problems/catalog", segments: [] },
  "getProblemAsset": { method: "GET", path: "/problems/{id}/assets/{asset_path}", segments: ["asset_path"] },
  "getProblem": { method: "GET", path: "/problems/{id}", segments: [] },
  "problemChangeTicketOnlyViaProblemsId": { method: "PUT", path: "/problems/{id}", segments: [] },
  "problemChangeTicketOnlyDeleteProblem": { method: "DELETE", path: "/problems/{id}", segments: [] },
  "me": { method: "GET", path: "/me", segments: [] },
  "activeExam": { method: "GET", path: "/me/active-exam", segments: [] },
  "getProfile": { method: "GET", path: "/profile", segments: [] },
  "updateProfile": { method: "PUT", path: "/profile", segments: [] },
  "deleteProfile": { method: "DELETE", path: "/profile", segments: [] },
  "updateProfilePassword": { method: "POST", path: "/profile/password", segments: [] },
  "sendProfileEmailCode": { method: "POST", path: "/profile/email-code", segments: [] },
  "rebindEmail": { method: "POST", path: "/profile/email", segments: [] },
  "getMyAuthorApplication": { method: "GET", path: "/author-applications/me", segments: [] },
  "listAuthorApplications": { method: "GET", path: "/author-applications", segments: [] },
  "createAuthorApplication": { method: "POST", path: "/author-applications", segments: [] },
  "createFeedback": { method: "POST", path: "/feedback", segments: [] },
  "listTeams": { method: "GET", path: "/teams", segments: [] },
  "createTeam": { method: "POST", path: "/teams", segments: [] },
  "getTeam": { method: "GET", path: "/teams/{id}", segments: [] },
  "updateTeam": { method: "PUT", path: "/teams/{id}", segments: [] },
  "joinTeam": { method: "POST", path: "/teams/{id}/join", segments: [] },
  "leaveTeam": { method: "POST", path: "/teams/{id}/leave", segments: [] },
  "listTeamMembers": { method: "GET", path: "/teams/{id}/members", segments: [] },
  "updateTeamMember": { method: "PUT", path: "/teams/{id}/members/{user_id}", segments: [] },
  "removeTeamMember": { method: "DELETE", path: "/teams/{id}/members/{user_id}", segments: [] },
  "listTeamApplications": { method: "GET", path: "/teams/{id}/applications", segments: [] },
  "reviewTeamApplication": { method: "PUT", path: "/teams/{id}/applications/{application_id}", segments: [] },
  "listTeamContests": { method: "GET", path: "/teams/{id}/contests", segments: [] },
  "createTeamContest": { method: "POST", path: "/teams/{id}/contests", segments: [] },
  "getTeamContest": { method: "GET", path: "/teams/{id}/contests/{contest_id}", segments: [] },
  "updateTeamContest": { method: "PUT", path: "/teams/{id}/contests/{contest_id}", segments: [] },
  "deleteTeamContest": { method: "DELETE", path: "/teams/{id}/contests/{contest_id}", segments: [] },
  "publishTeamContest": { method: "POST", path: "/teams/{id}/contests/{contest_id}/publish", segments: [] },
  "addTeamContestProblem": { method: "POST", path: "/teams/{id}/contests/{contest_id}/problems", segments: [] },
  "removeTeamContestProblem": { method: "DELETE", path: "/teams/{id}/contests/{contest_id}/problems/{problem_id}", segments: [] },
  "listTeamContestSubmissions": { method: "GET", path: "/teams/{id}/contests/{contest_id}/submissions", segments: [] },
  "createTeamContestSubmission": { method: "POST", path: "/teams/{id}/contests/{contest_id}/submissions", segments: [] },
  "teamContestRanking": { method: "GET", path: "/teams/{id}/contests/{contest_id}/ranking", segments: [] },
  "getTeamContestViaContestsContestId": { method: "GET", path: "/contests/{contest_id}", segments: [] },
  "updateTeamContestViaContestsContestId": { method: "PUT", path: "/contests/{contest_id}", segments: [] },
  "deleteTeamContestViaContestsContestId": { method: "DELETE", path: "/contests/{contest_id}", segments: [] },
  "publishTeamContestViaContestsContestIdPublish": { method: "POST", path: "/contests/{contest_id}/publish", segments: [] },
  "addTeamContestProblemViaContestsContestIdProblems": { method: "POST", path: "/contests/{contest_id}/problems", segments: [] },
  "removeTeamContestProblemViaContestsContestIdProblemsProblemId": { method: "DELETE", path: "/contests/{contest_id}/problems/{problem_id}", segments: [] },
  "listTeamContestSubmissionsViaContestsContestIdSubmissions": { method: "GET", path: "/contests/{contest_id}/submissions", segments: [] },
  "createTeamContestSubmissionViaContestsContestIdSubmissions": { method: "POST", path: "/contests/{contest_id}/submissions", segments: [] },
  "teamContestRankingViaContestsContestIdRanking": { method: "GET", path: "/contests/{contest_id}/ranking", segments: [] },
  "listTeamProblemSets": { method: "GET", path: "/teams/{id}/problem-sets", segments: [] },
  "createTeamProblemSet": { method: "POST", path: "/teams/{id}/problem-sets", segments: [] },
  "getTeamProblemSet": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}", segments: [] },
  "updateTeamProblemSet": { method: "PUT", path: "/teams/{id}/problem-sets/{set_id}", segments: [] },
  "deleteTeamProblemSet": { method: "DELETE", path: "/teams/{id}/problem-sets/{set_id}", segments: [] },
  "addTeamProblemSetProblem": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/problems", segments: [] },
  "removeTeamProblemSetProblem": { method: "DELETE", path: "/teams/{id}/problem-sets/{set_id}/problems/{problem_id}", segments: [] },
  "listTeamProblemSetSubmissions": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}/submissions", segments: [] },
  "createTeamProblemSetSubmission": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/submissions", segments: [] },
  "listTeamDiscussions": { method: "GET", path: "/teams/{id}/problem-sets/{set_id}/discussions", segments: [] },
  "createTeamDiscussion": { method: "POST", path: "/teams/{id}/problem-sets/{set_id}/discussions", segments: [] },
  "getTeamProblemSetViaProblemSetsSetId": { method: "GET", path: "/problem-sets/{set_id}", segments: [] },
  "updateTeamProblemSetViaProblemSetsSetId": { method: "PUT", path: "/problem-sets/{set_id}", segments: [] },
  "deleteTeamProblemSetViaProblemSetsSetId": { method: "DELETE", path: "/problem-sets/{set_id}", segments: [] },
  "addTeamProblemSetProblemViaProblemSetsSetIdProblems": { method: "POST", path: "/problem-sets/{set_id}/problems", segments: [] },
  "removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId": { method: "DELETE", path: "/problem-sets/{set_id}/problems/{problem_id}", segments: [] },
  "listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions": { method: "GET", path: "/problem-sets/{set_id}/submissions", segments: [] },
  "createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions": { method: "POST", path: "/problem-sets/{set_id}/submissions", segments: [] },
  "listTeamDiscussionsViaProblemSetsSetIdDiscussions": { method: "GET", path: "/problem-sets/{set_id}/discussions", segments: [] },
  "createTeamDiscussionViaProblemSetsSetIdDiscussions": { method: "POST", path: "/problem-sets/{set_id}/discussions", segments: [] },
  "listCourses": { method: "GET", path: "/courses", segments: [] },
  "createCourse": { method: "POST", path: "/courses", segments: [] },
  "updateCourse": { method: "PUT", path: "/courses/{id}", segments: [] },
  "archiveCourse": { method: "DELETE", path: "/courses/{id}", segments: [] },
  "archiveCourseViaCoursesIdArchive": { method: "POST", path: "/courses/{id}/archive", segments: [] },
  "createClass": { method: "POST", path: "/courses/{id}/classes", segments: [] },
  "listCourseMembers": { method: "GET", path: "/courses/{id}/members", segments: [] },
  "addCourseMember": { method: "POST", path: "/courses/{id}/members", segments: [] },
  "removeCourseMember": { method: "DELETE", path: "/courses/{id}/members/{user_id}", segments: [] },
  "previewCourseByCode": { method: "GET", path: "/courses/preview", segments: [] },
  "joinCourseByCode": { method: "POST", path: "/courses/join", segments: [] },
  "leaveCourse": { method: "POST", path: "/courses/{id}/leave", segments: [] },
  "listCourseStudents": { method: "GET", path: "/courses/{id}/students", segments: [] },
  "listClasses": { method: "GET", path: "/classes", segments: [] },
  "previewClassJoin": { method: "GET", path: "/classes/join-preview", segments: [] },
  "myClasses": { method: "GET", path: "/me/classes", segments: [] },
  "joinClassByCode": { method: "POST", path: "/classes/join", segments: [] },
  "joinClass": { method: "POST", path: "/classes/{id}/join", segments: [] },
  "leaveClass": { method: "POST", path: "/classes/{id}/leave", segments: [] },
  "updateClass": { method: "PUT", path: "/classes/{id}", segments: [] },
  "archiveClass": { method: "DELETE", path: "/classes/{id}", segments: [] },
  "archiveClassViaClassesIdArchive": { method: "POST", path: "/classes/{id}/archive", segments: [] },
  "listClassStudents": { method: "GET", path: "/classes/{id}/students", segments: [] },
  "removeClassStudent": { method: "DELETE", path: "/classes/{id}/students/{user_id}", segments: [] },
  "importClassStudents": { method: "POST", path: "/classes/{id}/students/import", segments: [] },
  "problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex": { method: "PUT", path: "/problem-test-uploads/{upload_id}/{chunk_index}", segments: [] },
  "parseMarkdownBatch": { method: "POST", path: "/problems/parse-markdown", segments: [] },
  "problemChangeTicketOnlyViaProblemsUpload": { method: "POST", path: "/problems/upload", segments: [] },
  "listProblemTests": { method: "GET", path: "/problems/{id}/tests", segments: [] },
  "downloadProblemTests": { method: "GET", path: "/problems/{id}/tests/download", segments: [] },
  "downloadProblemTestFile": { method: "GET", path: "/problems/{id}/tests/file/{file_path}", segments: ["file_path"] },
  "listPreparedProblems": { method: "GET", path: "/prepared-problems", segments: [] },
  "problemChangeTicketOnlyViaPreparedProblems": { method: "POST", path: "/prepared-problems", segments: [] },
  "problemChangeTicketOnlyViaPreparedProblemsUpload": { method: "POST", path: "/prepared-problems/upload", segments: [] },
  "getPreparedProblem": { method: "GET", path: "/prepared-problems/{id}", segments: [] },
  "problemChangeTicketOnlyViaPreparedProblemsId": { method: "PUT", path: "/prepared-problems/{id}", segments: [] },
  "problemChangeTicketOnlyViaPreparedProblemsIdPublish": { method: "POST", path: "/prepared-problems/{id}/publish", segments: [] },
  "listProblemChangeTickets": { method: "GET", path: "/problem-change-tickets", segments: [] },
  "createProblemChangeTicket": { method: "POST", path: "/problem-change-tickets", segments: [] },
  "listMyProblemChangeTickets": { method: "GET", path: "/problem-change-tickets/mine", segments: [] },
  "listProblemChangeTicketEligibleProblems": { method: "GET", path: "/problem-change-tickets/eligible-problems", segments: [] },
  "getProblemChangeTicket": { method: "GET", path: "/problem-change-tickets/{id}", segments: [] },
  "updateProblemChangeTicket": { method: "PATCH", path: "/problem-change-tickets/{id}", segments: [] },
  "deleteProblemChangeTicket": { method: "DELETE", path: "/problem-change-tickets/{id}", segments: [] },
  "downloadProblemChangeTicketAttachment": { method: "GET", path: "/problem-change-tickets/{id}/attachment", segments: [] },
  "cancelProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/cancel", segments: [] },
  "rejectProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/reject", segments: [] },
  "applyProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/apply", segments: [] },
  "listAssignments": { method: "GET", path: "/assignments", segments: [] },
  "createAssignment": { method: "POST", path: "/assignments", segments: [] },
  "getAssignment": { method: "GET", path: "/assignments/{id}", segments: [] },
  "deleteAssignment": { method: "DELETE", path: "/assignments/{id}", segments: [] },
  "assignmentReport": { method: "GET", path: "/assignments/{id}/report", segments: [] },
  "listExams": { method: "GET", path: "/exams", segments: [] },
  "createExam": { method: "POST", path: "/exams", segments: [] },
  "getExam": { method: "GET", path: "/exams/{id}", segments: [] },
  "deleteExam": { method: "DELETE", path: "/exams/{id}", segments: [] },
  "finishExam": { method: "POST", path: "/exams/{id}/finish", segments: [] },
  "exportExamReport": { method: "GET", path: "/exams/{id}/report/export", segments: [] },
  "examReport": { method: "GET", path: "/exams/{id}/report", segments: [] },
  "examRanking": { method: "GET", path: "/exams/{id}/ranking", segments: [] },
  "judgeManualExamSubmission": { method: "POST", path: "/exams/{id}/submissions/{submission_id}/judge", segments: [] },
  "gradeManualExamSubmission": { method: "PUT", path: "/exams/{id}/submissions/{submission_id}/grade", segments: [] },
  "listSubmissions": { method: "GET", path: "/submissions", segments: [] },
  "createSubmission": { method: "POST", path: "/submissions", segments: [] },
  "latestSubmissions": { method: "GET", path: "/submissions/latest", segments: [] },
  "getSubmission": { method: "GET", path: "/submissions/{id}", segments: [] },
  "submissionEvents": { method: "GET", path: "/submissions/{id}/events", segments: [] },
  "listPlagiarismJobs": { method: "GET", path: "/plagiarism/jobs", segments: [] },
  "createPlagiarismJob": { method: "POST", path: "/plagiarism/jobs", segments: [] },
  "listAuditLogs": { method: "GET", path: "/audit-logs", segments: [] },
  "listUsers": { method: "GET", path: "/users", segments: [] },
  "createUser": { method: "POST", path: "/users", segments: [] },
  "listUserPermissionDefinitions": { method: "GET", path: "/permissions", segments: [] },
  "getUser": { method: "GET", path: "/users/{id}", segments: [] },
  "updateUser": { method: "PUT", path: "/users/{id}", segments: [] },
  "deleteUser": { method: "DELETE", path: "/users/{id}", segments: [] },
  "updateUserPermission": { method: "PUT", path: "/users/{id}/permissions/{permission}", segments: [] },
  "resetUserPassword": { method: "POST", path: "/users/{id}/reset-password", segments: [] },
  "reviewAuthorApplication": { method: "PUT", path: "/author-applications/{id}/review", segments: [] },
  "listProblemAuthors": { method: "GET", path: "/problem-authors", segments: [] },
  "removeProblemAuthor": { method: "DELETE", path: "/problem-authors/{id}", segments: [] },
  "listMyProblemReviews": { method: "GET", path: "/problem-reviews/mine", segments: [] },
  "listProblemReviews": { method: "GET", path: "/problem-reviews", segments: [] },
  "problemChangeTicketOnlyViaProblemReviewsIdReview": { method: "PUT", path: "/problem-reviews/{id}/review", segments: [] },
  "problemChangeTicketOnlyViaProblemReviewsIdWithdraw": { method: "PUT", path: "/problem-reviews/{id}/withdraw", segments: [] },
  "enterExam": { method: "POST", path: "/exams/{id}/enter", segments: [] },
  "listExamActivity": { method: "GET", path: "/exams/{id}/activity", segments: [] },
  "recordExamActivity": { method: "POST", path: "/exams/{id}/activity", segments: [] },
  "getExamProblemAsset": { method: "GET", path: "/exams/{id}/problems/{problem_id}/assets/{asset_path}", segments: ["asset_path"] },
  "previewTeamInvitation": { method: "POST", path: "/team-invitations/preview", segments: [] },
  "previewTeamJoin": { method: "GET", path: "/teams/{id}/join-preview", segments: [] },
  "resolveProblemChangeTicket": { method: "POST", path: "/problem-change-tickets/{id}/resolve", segments: [] },
  "downloadTicketAttachment": { method: "GET", path: "/problem-change-tickets/{id}/attachments/{attachment_id}", segments: [] },
  "listProblemFeedbackTickets": { method: "GET", path: "/problems/{id}/change-tickets", segments: [] },
  "createProblemFeedbackTicket": { method: "POST", path: "/problems/{id}/change-tickets", segments: [] },
  "getProblemFeedbackTicket": { method: "GET", path: "/problems/{id}/change-tickets/{ticket_id}", segments: [] },
  "updateProblemFeedbackTicket": { method: "PATCH", path: "/problems/{id}/change-tickets/{ticket_id}", segments: [] },
  "deleteProblemFeedbackTicket": { method: "DELETE", path: "/problems/{id}/change-tickets/{ticket_id}", segments: [] },
  "downloadProblemFeedbackTicketAttachment": { method: "GET", path: "/problems/{id}/change-tickets/{ticket_id}/attachments/{attachment_id}", segments: [] },
  "cancelProblemFeedbackTicket": { method: "POST", path: "/problems/{id}/change-tickets/{ticket_id}/cancel", segments: [] },
  "rejectProblemFeedbackTicket": { method: "POST", path: "/problems/{id}/change-tickets/{ticket_id}/reject", segments: [] },
  "resolveProblemFeedbackTicket": { method: "POST", path: "/problems/{id}/change-tickets/{ticket_id}/resolve", segments: [] },
  "listExamFeedbackTickets": { method: "GET", path: "/exams/{id}/change-tickets", segments: [] },
  "createExamFeedbackTicket": { method: "POST", path: "/exams/{id}/change-tickets", segments: [] },
  "getExamFeedbackTicket": { method: "GET", path: "/exams/{id}/change-tickets/{ticket_id}", segments: [] },
  "updateExamFeedbackTicket": { method: "PATCH", path: "/exams/{id}/change-tickets/{ticket_id}", segments: [] },
  "deleteExamFeedbackTicket": { method: "DELETE", path: "/exams/{id}/change-tickets/{ticket_id}", segments: [] },
  "downloadExamFeedbackTicketAttachment": { method: "GET", path: "/exams/{id}/change-tickets/{ticket_id}/attachments/{attachment_id}", segments: [] },
  "cancelExamFeedbackTicket": { method: "POST", path: "/exams/{id}/change-tickets/{ticket_id}/cancel", segments: [] },
  "rejectExamFeedbackTicket": { method: "POST", path: "/exams/{id}/change-tickets/{ticket_id}/reject", segments: [] },
  "resolveExamFeedbackTicket": { method: "POST", path: "/exams/{id}/change-tickets/{ticket_id}/resolve", segments: [] },
  "listContestFeedbackTickets": { method: "GET", path: "/contests/{contest_id}/change-tickets", segments: [] },
  "createContestFeedbackTicket": { method: "POST", path: "/contests/{contest_id}/change-tickets", segments: [] },
  "getContestFeedbackTicket": { method: "GET", path: "/contests/{contest_id}/change-tickets/{ticket_id}", segments: [] },
  "updateContestFeedbackTicket": { method: "PATCH", path: "/contests/{contest_id}/change-tickets/{ticket_id}", segments: [] },
  "deleteContestFeedbackTicket": { method: "DELETE", path: "/contests/{contest_id}/change-tickets/{ticket_id}", segments: [] },
  "downloadContestFeedbackTicketAttachment": { method: "GET", path: "/contests/{contest_id}/change-tickets/{ticket_id}/attachments/{attachment_id}", segments: [] },
  "cancelContestFeedbackTicket": { method: "POST", path: "/contests/{contest_id}/change-tickets/{ticket_id}/cancel", segments: [] },
  "rejectContestFeedbackTicket": { method: "POST", path: "/contests/{contest_id}/change-tickets/{ticket_id}/reject", segments: [] },
  "resolveContestFeedbackTicket": { method: "POST", path: "/contests/{contest_id}/change-tickets/{ticket_id}/resolve", segments: [] },
} as const

export type ApiOperationId = keyof typeof apiOperations
export type ApiPathValue = string | number | boolean
export type ApiQueryValue = ApiPathValue | null | undefined | ApiPathValue[]

export interface ApiRequestOptions<TBody = unknown>
  extends Omit<AxiosRequestConfig, 'method' | 'url' | 'params' | 'data'> {
  path?: Record<string, ApiPathValue>
  query?: Record<string, ApiQueryValue>
  body?: TBody | FormData | Blob
}

function resolvePath(template: string, values: Record<string, ApiPathValue> = {}, segments: readonly string[] = []) {
  return template.replace(/\{([^}]+)\}/g, (_, key: string) => {
    const value = values[key]
    if (value === undefined || value === null || value === '') {
      throw new Error(`Missing OpenAPI path parameter: ${key}`)
    }
    if (segments.includes(key)) {
      const parts = String(value).split('/')
      if (parts.some(part => part === '..' || part === '.' || !part)) throw new Error('Invalid asset path')
      return parts.map(part => encodeURIComponent(part)).join('/')
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
    url: resolvePath(definition.path, path, definition.segments),
    params: query,
    data: body,
  })
}

export const generatedApi = {
  "login": <TResponse = components['schemas']["AuthResponse"]>(options: ApiRequestOptions<components['schemas']["LoginRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["LoginRequest"]>("login", options),
  "sendEmailCode": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["EmailCodeRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["EmailCodeRequest"]>("sendEmailCode", options),
  "register": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["RegisterRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["RegisterRequest"]>("register", options),
  "resetPassword": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["PasswordResetRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["PasswordResetRequest"]>("resetPassword", options),
  "listProblems": <TResponse = Array<components['schemas']["Problem"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblems", options),
  "problemChangeTicketOnly": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnly", options),
  "listProblemCatalog": <TResponse = components['schemas']["ProblemCatalogResponse"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemCatalog", options),
  "getProblemAsset": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getProblemAsset", options),
  "getProblem": <TResponse = components['schemas']["Problem"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getProblem", options),
  "problemChangeTicketOnlyViaProblemsId": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemsId", options),
  "problemChangeTicketOnlyDeleteProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("problemChangeTicketOnlyDeleteProblem", options),
  "me": <TResponse = components['schemas']["User"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("me", options),
  "activeExam": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("activeExam", options),
  "getProfile": <TResponse = components['schemas']["User"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getProfile", options),
  "updateProfile": <TResponse = components['schemas']["User"]>(options: ApiRequestOptions<components['schemas']["ProfileUpdateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProfileUpdateRequest"]>("updateProfile", options),
  "deleteProfile": <TResponse = components['schemas']["User"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteProfile", options),
  "updateProfilePassword": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["PasswordChangeRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["PasswordChangeRequest"]>("updateProfilePassword", options),
  "sendProfileEmailCode": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("sendProfileEmailCode", options),
  "rebindEmail": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["EmailRebindRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["EmailRebindRequest"]>("rebindEmail", options),
  "getMyAuthorApplication": <TResponse = components['schemas']["AuthorApplicationEnvelope"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getMyAuthorApplication", options),
  "listAuthorApplications": <TResponse = Array<components['schemas']["AuthorApplication"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listAuthorApplications", options),
  "createAuthorApplication": <TResponse = components['schemas']["AuthorApplication"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createAuthorApplication", options),
  "createFeedback": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createFeedback", options),
  "listTeams": <TResponse = Array<components['schemas']["Team"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeams", options),
  "createTeam": <TResponse = components['schemas']["Team"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createTeam", options),
  "getTeam": <TResponse = components['schemas']["Team"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getTeam", options),
  "updateTeam": <TResponse = components['schemas']["Team"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateTeam", options),
  "joinTeam": <TResponse = components['schemas']["TeamJoinResponse"]>(options: ApiRequestOptions<components['schemas']["TeamJoinRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamJoinRequest"]>("joinTeam", options),
  "leaveTeam": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("leaveTeam", options),
  "listTeamMembers": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamMembers", options),
  "updateTeamMember": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateTeamMember", options),
  "removeTeamMember": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeTeamMember", options),
  "listTeamApplications": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamApplications", options),
  "reviewTeamApplication": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("reviewTeamApplication", options),
  "listTeamContests": <TResponse = Array<components['schemas']["TeamContest"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamContests", options),
  "createTeamContest": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createTeamContest", options),
  "getTeamContest": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getTeamContest", options),
  "updateTeamContest": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateTeamContest", options),
  "deleteTeamContest": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteTeamContest", options),
  "publishTeamContest": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("publishTeamContest", options),
  "addTeamContestProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("addTeamContestProblem", options),
  "removeTeamContestProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeTeamContestProblem", options),
  "listTeamContestSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamContestSubmissions", options),
  "createTeamContestSubmission": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["SubmissionCreateRequest"]>("createTeamContestSubmission", options),
  "teamContestRanking": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("teamContestRanking", options),
  "getTeamContestViaContestsContestId": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getTeamContestViaContestsContestId", options),
  "updateTeamContestViaContestsContestId": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateTeamContestViaContestsContestId", options),
  "deleteTeamContestViaContestsContestId": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteTeamContestViaContestsContestId", options),
  "publishTeamContestViaContestsContestIdPublish": <TResponse = components['schemas']["TeamContest"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("publishTeamContestViaContestsContestIdPublish", options),
  "addTeamContestProblemViaContestsContestIdProblems": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("addTeamContestProblemViaContestsContestIdProblems", options),
  "removeTeamContestProblemViaContestsContestIdProblemsProblemId": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeTeamContestProblemViaContestsContestIdProblemsProblemId", options),
  "listTeamContestSubmissionsViaContestsContestIdSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamContestSubmissionsViaContestsContestIdSubmissions", options),
  "createTeamContestSubmissionViaContestsContestIdSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["SubmissionCreateRequest"]>("createTeamContestSubmissionViaContestsContestIdSubmissions", options),
  "teamContestRankingViaContestsContestIdRanking": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("teamContestRankingViaContestsContestIdRanking", options),
  "listTeamProblemSets": <TResponse = Array<components['schemas']["TeamProblemSet"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamProblemSets", options),
  "createTeamProblemSet": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["TeamProblemSetInput"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamProblemSetInput"]>("createTeamProblemSet", options),
  "getTeamProblemSet": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getTeamProblemSet", options),
  "updateTeamProblemSet": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<components['schemas']["TeamProblemSetInput"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamProblemSetInput"]>("updateTeamProblemSet", options),
  "deleteTeamProblemSet": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteTeamProblemSet", options),
  "addTeamProblemSetProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["TeamProblemSelectionRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamProblemSelectionRequest"]>("addTeamProblemSetProblem", options),
  "removeTeamProblemSetProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeTeamProblemSetProblem", options),
  "listTeamProblemSetSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamProblemSetSubmissions", options),
  "createTeamProblemSetSubmission": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["SubmissionCreateRequest"]>("createTeamProblemSetSubmission", options),
  "listTeamDiscussions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamDiscussions", options),
  "createTeamDiscussion": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createTeamDiscussion", options),
  "getTeamProblemSetViaProblemSetsSetId": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getTeamProblemSetViaProblemSetsSetId", options),
  "updateTeamProblemSetViaProblemSetsSetId": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<components['schemas']["TeamProblemSetInput"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamProblemSetInput"]>("updateTeamProblemSetViaProblemSetsSetId", options),
  "deleteTeamProblemSetViaProblemSetsSetId": <TResponse = components['schemas']["TeamProblemSet"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteTeamProblemSetViaProblemSetsSetId", options),
  "addTeamProblemSetProblemViaProblemSetsSetIdProblems": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["TeamProblemSelectionRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TeamProblemSelectionRequest"]>("addTeamProblemSetProblemViaProblemSetsSetIdProblems", options),
  "removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId", options),
  "listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions", options),
  "createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["SubmissionCreateRequest"]>("createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions", options),
  "listTeamDiscussionsViaProblemSetsSetIdDiscussions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listTeamDiscussionsViaProblemSetsSetIdDiscussions", options),
  "createTeamDiscussionViaProblemSetsSetIdDiscussions": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createTeamDiscussionViaProblemSetsSetIdDiscussions", options),
  "listCourses": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listCourses", options),
  "createCourse": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createCourse", options),
  "updateCourse": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateCourse", options),
  "archiveCourse": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("archiveCourse", options),
  "archiveCourseViaCoursesIdArchive": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("archiveCourseViaCoursesIdArchive", options),
  "createClass": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createClass", options),
  "listCourseMembers": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listCourseMembers", options),
  "addCourseMember": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("addCourseMember", options),
  "removeCourseMember": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeCourseMember", options),
  "previewCourseByCode": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("previewCourseByCode", options),
  "joinCourseByCode": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("joinCourseByCode", options),
  "leaveCourse": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("leaveCourse", options),
  "listCourseStudents": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listCourseStudents", options),
  "listClasses": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listClasses", options),
  "previewClassJoin": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("previewClassJoin", options),
  "myClasses": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("myClasses", options),
  "joinClassByCode": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("joinClassByCode", options),
  "joinClass": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("joinClass", options),
  "leaveClass": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("leaveClass", options),
  "updateClass": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateClass", options),
  "archiveClass": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("archiveClass", options),
  "archiveClassViaClassesIdArchive": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("archiveClassViaClassesIdArchive", options),
  "listClassStudents": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listClassStudents", options),
  "removeClassStudent": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeClassStudent", options),
  "importClassStudents": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("importClassStudents", options),
  "problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemTestUploadsUploadIdChunkIndex", options),
  "parseMarkdownBatch": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("parseMarkdownBatch", options),
  "problemChangeTicketOnlyViaProblemsUpload": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemsUpload", options),
  "listProblemTests": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemTests", options),
  "downloadProblemTests": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadProblemTests", options),
  "downloadProblemTestFile": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadProblemTestFile", options),
  "listPreparedProblems": <TResponse = Array<components['schemas']["PreparedProblem"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listPreparedProblems", options),
  "problemChangeTicketOnlyViaPreparedProblems": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblems", options),
  "problemChangeTicketOnlyViaPreparedProblemsUpload": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsUpload", options),
  "getPreparedProblem": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getPreparedProblem", options),
  "problemChangeTicketOnlyViaPreparedProblemsId": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsId", options),
  "problemChangeTicketOnlyViaPreparedProblemsIdPublish": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaPreparedProblemsIdPublish", options),
  "listProblemChangeTickets": <TResponse = components['schemas']["TicketPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemChangeTickets", options),
  "createProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("createProblemChangeTicket", options),
  "listMyProblemChangeTickets": <TResponse = components['schemas']["TicketPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listMyProblemChangeTickets", options),
  "listProblemChangeTicketEligibleProblems": <TResponse = Array<components['schemas']["TicketProblem"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemChangeTicketEligibleProblems", options),
  "getProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getProblemChangeTicket", options),
  "updateProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("updateProblemChangeTicket", options),
  "deleteProblemChangeTicket": <TResponse = {
    "deleted"?: boolean
  }>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("deleteProblemChangeTicket", options),
  "downloadProblemChangeTicketAttachment": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadProblemChangeTicketAttachment", options),
  "cancelProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("cancelProblemChangeTicket", options),
  "rejectProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("rejectProblemChangeTicket", options),
  "applyProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketApply"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketApply"]>("applyProblemChangeTicket", options),
  "listAssignments": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listAssignments", options),
  "createAssignment": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createAssignment", options),
  "getAssignment": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getAssignment", options),
  "deleteAssignment": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteAssignment", options),
  "assignmentReport": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("assignmentReport", options),
  "listExams": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listExams", options),
  "createExam": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createExam", options),
  "getExam": <TResponse = components['schemas']["ExamDetail"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getExam", options),
  "deleteExam": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteExam", options),
  "finishExam": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("finishExam", options),
  "exportExamReport": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("exportExamReport", options),
  "examReport": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("examReport", options),
  "examRanking": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("examRanking", options),
  "judgeManualExamSubmission": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("judgeManualExamSubmission", options),
  "gradeManualExamSubmission": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("gradeManualExamSubmission", options),
  "listSubmissions": <TResponse = Array<components['schemas']["Submission"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listSubmissions", options),
  "createSubmission": <TResponse = components['schemas']["Submission"]>(options: ApiRequestOptions<components['schemas']["SubmissionCreateRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["SubmissionCreateRequest"]>("createSubmission", options),
  "latestSubmissions": <TResponse = Array<components['schemas']["Submission"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("latestSubmissions", options),
  "getSubmission": <TResponse = components['schemas']["SubmissionDetail"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getSubmission", options),
  "submissionEvents": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("submissionEvents", options),
  "listPlagiarismJobs": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listPlagiarismJobs", options),
  "createPlagiarismJob": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createPlagiarismJob", options),
  "listAuditLogs": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listAuditLogs", options),
  "listUsers": <TResponse = Array<components['schemas']["User"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listUsers", options),
  "createUser": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("createUser", options),
  "listUserPermissionDefinitions": <TResponse = Array<components['schemas']["PermissionDefinition"]>>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listUserPermissionDefinitions", options),
  "getUser": <TResponse = components['schemas']["User"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getUser", options),
  "updateUser": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateUser", options),
  "deleteUser": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("deleteUser", options),
  "updateUserPermission": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("updateUserPermission", options),
  "resetUserPassword": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("resetUserPassword", options),
  "reviewAuthorApplication": <TResponse = components['schemas']["AuthorApplication"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("reviewAuthorApplication", options),
  "listProblemAuthors": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemAuthors", options),
  "removeProblemAuthor": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("removeProblemAuthor", options),
  "listMyProblemReviews": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listMyProblemReviews", options),
  "listProblemReviews": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemReviews", options),
  "problemChangeTicketOnlyViaProblemReviewsIdReview": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemReviewsIdReview", options),
  "problemChangeTicketOnlyViaProblemReviewsIdWithdraw": <TResponse = components['schemas']["JsonObject"]>(options: ApiRequestOptions<components['schemas']["JsonObject"]> = {}) =>
    requestOperation<TResponse, components['schemas']["JsonObject"]>("problemChangeTicketOnlyViaProblemReviewsIdWithdraw", options),
  "enterExam": <TResponse = components['schemas']["ExamEnterResponse"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("enterExam", options),
  "listExamActivity": <TResponse = components['schemas']["ExamActivityPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listExamActivity", options),
  "recordExamActivity": <TResponse = {
    "recorded": boolean
  }>(options: ApiRequestOptions<components['schemas']["ExamActivityRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ExamActivityRequest"]>("recordExamActivity", options),
  "getExamProblemAsset": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getExamProblemAsset", options),
  "previewTeamInvitation": <TResponse = components['schemas']["TeamJoinPreview"]>(options: ApiRequestOptions<{
    "join_code": string
  }> = {}) =>
    requestOperation<TResponse, {
    "join_code": string
  }>("previewTeamInvitation", options),
  "previewTeamJoin": <TResponse = components['schemas']["TeamJoinPreview"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("previewTeamJoin", options),
  "resolveProblemChangeTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("resolveProblemChangeTicket", options),
  "downloadTicketAttachment": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadTicketAttachment", options),
  "listProblemFeedbackTickets": <TResponse = components['schemas']["TicketPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listProblemFeedbackTickets", options),
  "createProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("createProblemFeedbackTicket", options),
  "getProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getProblemFeedbackTicket", options),
  "updateProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("updateProblemFeedbackTicket", options),
  "deleteProblemFeedbackTicket": <TResponse = {
    "deleted"?: boolean
  }>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("deleteProblemFeedbackTicket", options),
  "downloadProblemFeedbackTicketAttachment": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadProblemFeedbackTicketAttachment", options),
  "cancelProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("cancelProblemFeedbackTicket", options),
  "rejectProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("rejectProblemFeedbackTicket", options),
  "resolveProblemFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("resolveProblemFeedbackTicket", options),
  "listExamFeedbackTickets": <TResponse = components['schemas']["TicketPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listExamFeedbackTickets", options),
  "createExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("createExamFeedbackTicket", options),
  "getExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getExamFeedbackTicket", options),
  "updateExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("updateExamFeedbackTicket", options),
  "deleteExamFeedbackTicket": <TResponse = {
    "deleted"?: boolean
  }>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("deleteExamFeedbackTicket", options),
  "downloadExamFeedbackTicketAttachment": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadExamFeedbackTicketAttachment", options),
  "cancelExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("cancelExamFeedbackTicket", options),
  "rejectExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("rejectExamFeedbackTicket", options),
  "resolveExamFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("resolveExamFeedbackTicket", options),
  "listContestFeedbackTickets": <TResponse = components['schemas']["TicketPage"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("listContestFeedbackTickets", options),
  "createContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("createContestFeedbackTicket", options),
  "getContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("getContestFeedbackTicket", options),
  "updateContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["ProblemChangeTicketRequest"]> = {}) =>
    requestOperation<TResponse, components['schemas']["ProblemChangeTicketRequest"]>("updateContestFeedbackTicket", options),
  "deleteContestFeedbackTicket": <TResponse = {
    "deleted"?: boolean
  }>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("deleteContestFeedbackTicket", options),
  "downloadContestFeedbackTicketAttachment": <TResponse = unknown>(options: ApiRequestOptions<never> = {}) =>
    requestOperation<TResponse, never>("downloadContestFeedbackTicketAttachment", options),
  "cancelContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("cancelContestFeedbackTicket", options),
  "rejectContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("rejectContestFeedbackTicket", options),
  "resolveContestFeedbackTicket": <TResponse = components['schemas']["ProblemChangeTicket"]>(options: ApiRequestOptions<components['schemas']["TicketDecision"]> = {}) =>
    requestOperation<TResponse, components['schemas']["TicketDecision"]>("resolveContestFeedbackTicket", options),
}

// 兼容现有页面的 Axios 调用；新代码优先使用 generatedApi 的 operationId 方法。
export const client = http

package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

// S3 transport fixture exercises real HTTP uploads/downloads without a Docker dependency.
// Real MinIO/container acceptance remains a separate release check.
type ticketObjectFixture struct {
	mu      sync.Mutex
	objects map[string][]byte
	fail    bool
}

func setupTicketObjects(t *testing.T, f *examTestFixture) *ticketObjectFixture {
	t.Helper()
	store := &ticketObjectFixture{objects: map[string][]byte{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store.mu.Lock()
		defer store.mu.Unlock()
		switch r.Method {
		case "PUT":
			if store.fail {
				w.WriteHeader(403)
				return
			}
			var reader io.Reader = r.Body
			if strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") || r.Header.Get("X-Amz-Decoded-Content-Length") != "" {
				reader = httputil.NewChunkedReader(r.Body)
			}
			body, err := io.ReadAll(reader)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			store.objects[r.URL.Path] = body
			sum := md5.Sum(body)
			w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:])+`"`)
			w.WriteHeader(200)
		case "DELETE":
			delete(store.objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			body, ok := store.objects[r.URL.Path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", "application/octet-stream")
			if r.Method != "HEAD" {
				_, _ = w.Write(body)
			}
		}
	}))
	t.Cleanup(server.Close)
	var err error
	f.s.MinIO, err = minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{Creds: credentials.NewStaticV4("test", "testsecret", ""), Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	f.s.Cfg.MinIOBucket = "ticket-test"
	f.router = f.s.Router()
	return store
}

func newTicketFixture(t *testing.T) *examTestFixture {
	t.Helper()
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires OJ_TEST_REDIS_ADDR")
	}
	clearTicketRate(t, f)
	return f
}
func clearTicketRate(t *testing.T, f *examTestFixture) {
	t.Helper()
	ctx := context.Background()
	keys, err := f.s.Redis.Keys(ctx, "oj:rate:problem-change-tickets:create:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) > 0 {
		f.s.Redis.Del(ctx, keys...)
	}
}

type ticketWire struct {
	ID               uint                             `json:"id"`
	Revision         int                              `json:"revision"`
	ProblemID        *uint                            `json:"problem_id"`
	ProblemVersionID *uint                            `json:"problem_version_id"`
	CurrentVersionID *uint                            `json:"current_version_id"`
	Action           string                           `json:"action"`
	Status           string                           `json:"status"`
	IsPublic         bool                             `json:"is_public"`
	Attachments      []models.ProblemTicketAttachment `json:"attachments"`
}

func readTicket(t *testing.T, r *httptest.ResponseRecorder) ticketWire {
	t.Helper()
	var v ticketWire
	if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func ticketDraft(action string, problemID uint) gin.H {
	v := gin.H{"action": action, "description": "题目输入范围与描述存在歧义，请核对修正"}
	if problemID != 0 {
		v["problem_id"] = problemID
	}
	return v
}
func ticketMultipart(t *testing.T, f *examTestFixture, method, path string, user models.User, draft any, files map[string][]byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if draft != nil {
		raw, _ := json.Marshal(draft)
		_ = writer.WriteField("draft", string(raw))
	}
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	for name, value := range files {
		key := "attachments"
		if strings.HasSuffix(path, "/apply") {
			key = "package"
		}
		part, err := writer.CreateFormFile(key, name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(value)
	}
	_ = writer.Close()
	request := httptest.NewRequest(method, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	token, _ := middleware.SignToken(f.s.Cfg.JWTSecret, user)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	return response
}
func ticketPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.RGBA{40, 120, 210, 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestTicketPublicPermissionsAndLifecyclePostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	base := fmt.Sprintf("/api/problems/%d/change-tickets", f.problem.ID)
	assertStatus(t, f.request("POST", base, nil, ticketDraft("replace", f.problem.ID)), 401)
	for _, user := range []models.User{f.student, f.peer, f.teacher, f.admin} {
		r := f.request("POST", base, &user, ticketDraft("replace", f.problem.ID))
		assertStatus(t, r, 201)
	}
	for _, action := range []string{"create", "archive"} {
		assertStatus(t, f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft(action, f.problem.ID)), 400)
	}
	r := ticketMultipart(t, f, "POST", base, f.student, ticketDraft("suggest", f.problem.ID), map[string][]byte{"1.in": []byte("secret test input"), "figure.png": ticketPNG(t)}, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	if len(ticket.Attachments) != 2 || !ticket.IsPublic {
		t.Fatal(r.Body.String())
	}
	detail := fmt.Sprintf("%s/%d", base, ticket.ID)
	for _, user := range []*models.User{nil, &f.peer} {
		r = f.request("GET", detail, user, nil)
		assertStatus(t, r, 200)
		raw := r.Body.String()
		for _, secret := range []string{"email", "student_no", "attachment_name", "1.in", "manifest", "package_object", "password_hash"} {
			if strings.Contains(raw, secret) {
				t.Fatalf("public DTO exposed %s: %s", secret, raw)
			}
		}
	}
	filePath := fmt.Sprintf("%s/attachments/%d", detail, ticket.Attachments[0].ID)
	assertStatus(t, f.request("GET", filePath, &f.peer, nil), 403)
	assertStatus(t, f.request("GET", filePath, &f.student, nil), 200)
	assertStatus(t, f.request("PATCH", detail, &f.peer, gin.H{"revision": 1}), 403)
	edit := ticketDraft("suggest", f.problem.ID)
	edit["revision"] = ticket.Revision
	edit["description"] = "补充反馈细节，说明对应边界情况"
	edit["retain_attachment_ids"] = []uint{ticket.Attachments[0].ID}
	r = f.request("PATCH", detail, &f.student, edit)
	assertStatus(t, r, 200)
	updated := readTicket(t, r)
	if updated.Revision != 2 || len(updated.Attachments) != 1 {
		t.Fatal(r.Body.String())
	}
	assertStatus(t, f.request("PATCH", detail, &f.student, edit), 409)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.teacher, gin.H{"revision": 2, "note": "确认问题已经处理完毕"}), 403)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.admin, gin.H{"revision": 2, "note": ""}), 400)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.admin, gin.H{"revision": 2, "note": "确认问题已经处理完毕"}), 200)
	assertStatus(t, f.request("DELETE", detail, &f.student, gin.H{"revision": 3}), 409)
	r = f.request("POST", base, &f.student, ticketDraft("replace", f.problem.ID))
	assertStatus(t, r, 201)
	ticket = readTicket(t, r)
	detail = fmt.Sprintf("%s/%d", base, ticket.ID)
	assertStatus(t, f.request("DELETE", detail, &f.student, gin.H{"revision": ticket.Revision}), 200)
	assertStatus(t, f.request("GET", detail, &f.student, nil), 404)
	var persisted models.ProblemChangeTicket
	if f.s.DB.First(&persisted, ticket.ID).Error != nil || persisted.DeletedAt == nil {
		t.Fatal("ticket was not soft deleted")
	}
}

func TestTicketPrivateAttachmentsAndFailuresPostgres(t *testing.T) {
	f := newTicketFixture(t)
	store := setupTicketObjects(t, f)
	base := fmt.Sprintf("/api/problems/%d/change-tickets", f.problem.ID)
	r := ticketMultipart(t, f, "POST", base, f.student, ticketDraft("replace", f.problem.ID), map[string][]byte{"picture.png": ticketPNG(t), "malicious.png": []byte("<script>alert(1)</script>")}, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	for _, file := range ticket.Attachments {
		path := fmt.Sprintf("%s/%d/attachments/%d", base, ticket.ID, file.ID)
		status := 200
		if file.Name == "malicious.png" {
			status = 400
		}
		response := f.request("GET", path+"?preview=1", &f.student, nil)
		assertStatus(t, response, status)
		if status == 200 && response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("private image was cacheable")
		}
		assertStatus(t, f.request("GET", path, &f.peer, nil), 403)
	}
	legacy := fmt.Sprintf("/api/problem-change-tickets/%d/attachment", ticket.ID)
	assertStatus(t, f.request("GET", legacy, &f.student, nil), 200)
	f.s.DB.Model(&models.ProblemTicketAttachment{}).Where("ticket_id = ?", ticket.ID).Update("size", 0)
	legacyEdit := ticketDraft("replace", f.problem.ID)
	legacyEdit["revision"] = 1
	legacyEdit["retain_attachment_ids"] = []uint{ticket.Attachments[0].ID, ticket.Attachments[1].ID}
	maximumFile := bytes.Repeat([]byte("x"), int(problemTicketAttachmentLimit))
	assertStatus(t, ticketMultipart(t, f, "PATCH", fmt.Sprintf("%s/%d", base, ticket.ID), f.student, legacyEdit, map[string][]byte{"1.in": maximumFile, "2.in": maximumFile, "3.in": maximumFile}, nil), 400)
	assertStatus(t, f.request("PATCH", fmt.Sprintf("%s/%d", base, ticket.ID), &f.student, legacyEdit), 200)
	edit := ticketDraft("replace", f.problem.ID)
	edit["revision"] = 1
	edit["retain_attachment_ids"] = []uint{999999}
	assertStatus(t, f.request("PATCH", fmt.Sprintf("%s/%d", base, ticket.ID), &f.student, edit), 400)
	assertStatus(t, ticketMultipart(t, f, "POST", base, f.student, ticketDraft("replace", f.problem.ID), map[string][]byte{"empty.in": {}}, nil), 400)
	files := map[string][]byte{}
	for i := 0; i < 21; i++ {
		files[fmt.Sprintf("%d.in", i)] = []byte("1")
	}
	assertStatus(t, ticketMultipart(t, f, "POST", base, f.student, ticketDraft("replace", f.problem.ID), files, nil), 400)
	var count int64
	f.s.DB.Model(&models.ProblemChangeTicket{}).Count(&count)
	store.mu.Lock()
	store.fail = true
	store.mu.Unlock()
	assertStatus(t, ticketMultipart(t, f, "POST", base, f.student, ticketDraft("replace", f.problem.ID), map[string][]byte{"1.in": []byte("1")}, nil), 503)
	var after int64
	f.s.DB.Model(&models.ProblemChangeTicket{}).Count(&after)
	if count != after {
		t.Fatal("failed upload created a ticket")
	}
}

func TestTicketExamScopeAndVersionPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	base := fmt.Sprintf("/api/exams/%d", f.exam.ID)
	ticketBase := base + "/change-tickets"
	assertStatus(t, f.request("POST", ticketBase, &f.student, ticketDraft("replace", f.problem.ID)), 400)
	assertStatus(t, f.request("POST", base+"/enter", &f.student, nil), 200)
	forgedDraft := ticketDraft("suggest", f.problem.ID)
	forgedDraft["draft_statement"] = "[离开考试](https://example.test)"
	assertStatus(t, f.request("POST", ticketBase, &f.student, forgedDraft), 400)
	v2 := f.version
	v2.ID = 0
	v2.Version = 2
	v2.Title = "UNRELEASED_VERSION_SECRET"
	f.s.DB.Create(&v2)
	f.s.DB.Model(&f.problem).Updates(map[string]any{"current_version_id": v2.ID, "title": v2.Title})
	req := ticketDraft("replace", f.problem.ID)
	req["problem_version_id"] = v2.ID
	req["target_scope"] = "public"
	r := ticketMultipart(t, f, "POST", ticketBase, f.student, req, map[string][]byte{"1.out": []byte("private output")}, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	if ticket.IsPublic || ticket.ProblemVersionID == nil || *ticket.ProblemVersionID != f.version.ID || strings.Contains(r.Body.String(), v2.Title) {
		t.Fatal("event version or visibility escaped")
	}
	detail := fmt.Sprintf("%s/%d", ticketBase, ticket.ID)
	for _, path := range []string{"/api/problem-change-tickets/mine", fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID), fmt.Sprintf("/api/problem-change-tickets/%d/attachments/%d", ticket.ID, ticket.Attachments[0].ID), fmt.Sprintf("/api/problems/%d/change-tickets", f.problem.ID), fmt.Sprintf("/api/exams/%d/change-tickets/%d", f.other.ID, ticket.ID)} {
		assertStatus(t, f.request("GET", path, &f.student, nil), 403)
	}
	assertStatus(t, f.request("GET", detail, &f.peer, nil), 403)
	assertStatus(t, f.request("GET", detail, &f.teacher, nil), 200)
	wrong := ticketDraft("suggest", 999999)
	assertStatus(t, f.request("POST", ticketBase, &f.student, wrong), 400)
	forged := ticketDraft("suggest", f.problem.ID)
	forged["exam_id"] = f.other.ID
	assertStatus(t, f.request("POST", ticketBase, &f.student, forged), 400)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.teacher, gin.H{"revision": 1, "note": "不允许跳过题库执行"}), 400)
	assertStatus(t, f.request("POST", detail+"/reject", &f.teacher, gin.H{"revision": 1, "note": "核对后暂不需要修改本题"}), 200)
	r = f.request("POST", ticketBase, &f.student, ticketDraft("suggest", 0))
	assertStatus(t, r, 201)
	overall := readTicket(t, r)
	assertStatus(t, f.request("POST", fmt.Sprintf("%s/%d/resolve", ticketBase, overall.ID), &f.teacher, gin.H{"revision": 1, "note": "已经说明考试规则"}), 200)
	assertStatus(t, f.request("POST", base+"/finish", &f.student, nil), 200)
	r = f.request("GET", fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID), &f.student, nil)
	assertStatus(t, r, 200)
	if strings.Contains(r.Body.String(), v2.Title) {
		t.Fatal("post-exam feedback leaked latest version")
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/problems/%d/change-tickets", f.problem.ID), &f.peer, nil), 200)
}

func TestTicketContestManagersAndMembershipPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	team := models.Team{Name: "工单比赛团队", Slug: "ticket-contest", OwnerID: f.teacher.ID, Visibility: "private", ContestPermission: "all"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	for _, u := range []models.User{f.teacher, f.student, f.peer} {
		role := models.TeamRoleMember
		if u.ID == f.teacher.ID {
			role = models.TeamRoleOwner
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: u.ID, Role: role})
	}
	start := time.Now().Add(-time.Minute)
	contest := models.TeamContest{TeamID: team.ID, CreatedBy: f.teacher.ID, Title: "工单比赛", StartsAt: &start, PublishedAt: &start, DurationMinutes: 60, ScoringRule: "ioi", State: "published"}
	f.s.DB.Create(&contest)
	f.s.DB.Create(&models.TeamContestProblem{ContestID: contest.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "A"})
	base := fmt.Sprintf("/api/contests/%d/change-tickets", contest.ID)
	r := f.request("POST", base, &f.student, ticketDraft("suggest", f.problem.ID))
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	detail := fmt.Sprintf("%s/%d", base, ticket.ID)
	assertStatus(t, f.request("GET", detail, &f.peer, nil), 403)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.peer, gin.H{"revision": 1, "note": "普通成员不可处理别人工单"}), 403)
	assertStatus(t, f.request("POST", detail+"/resolve", &f.teacher, gin.H{"revision": 1, "note": "比赛管理者已确认建议"}), 200)
	future := time.Now().Add(time.Hour)
	f.s.DB.Model(&contest).Update("starts_at", future)
	assertStatus(t, f.request("POST", base, &f.student, ticketDraft("replace", f.problem.ID)), 400)
	f.s.DB.Model(&contest).Update("starts_at", start)
	f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.student.ID).Delete(&models.TeamMembership{})
	assertStatus(t, f.request("GET", detail, &f.student, nil), 403)
}

func TestTicketApplyVersionsAndConcurrencyPostgres(t *testing.T) {
	f := newTicketFixture(t)
	objects := setupTicketObjects(t, f)
	future := time.Now().Add(time.Hour)
	f.s.DB.Model(&f.other).Update("starts_at", future)
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(3)", Status: models.StatusAccepted}
	f.s.DB.Create(&sub)
	body, _, err := services.BuildProblemPackage(services.ProblemPackageDraft{Slug: "ticket-test", Title: "修正后的题目", Statement: "## 新题面", TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024, Checker: services.CheckerManifest{Type: "exact"}, Cases: []services.ProblemCaseDraft{{Name: "one", Input: "1 2", Output: "3", Weight: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	r := f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("replace", f.problem.ID))
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	path := fmt.Sprintf("/api/problem-change-tickets/%d/apply", ticket.ID)
	fields := map[string]string{"revision": "1", "expected_version_id": fmt.Sprint(f.version.ID), "operation_mode": "overwrite"}
	assertStatus(t, ticketMultipart(t, f, "POST", path, f.admin, nil, map[string][]byte{"broken.zip": []byte("invalid")}, fields), 400)
	assertStatus(t, ticketMultipart(t, f, "POST", path, f.admin, nil, map[string][]byte{"final.zip": body}, fields), 200)
	assertStatus(t, ticketMultipart(t, f, "POST", path, f.admin, nil, map[string][]byte{"final.zip": body}, fields), 409)
	var problem models.Problem
	f.s.DB.First(&problem, f.problem.ID)
	if problem.CurrentVersionID == nil || *problem.CurrentVersionID == f.version.ID {
		t.Fatal("new version missing")
	}
	var link models.ExamProblem
	f.s.DB.Where("exam_id = ?", f.exam.ID).First(&link)
	if link.ProblemVersionID != f.version.ID {
		t.Fatal("active exam version changed")
	}
	link = models.ExamProblem{}
	f.s.DB.Where("exam_id = ?", f.other.ID).First(&link)
	if link.ProblemVersionID != *problem.CurrentVersionID {
		t.Fatal("future exam did not update")
	}
	var savedSub models.Submission
	f.s.DB.First(&savedSub, sub.ID)
	if savedSub.ProblemVersionID != f.version.ID {
		t.Fatal("historical submission changed")
	}
	r = f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("replace", f.problem.ID))
	assertStatus(t, r, 201)
	ticket = readTicket(t, r)
	path = fmt.Sprintf("/api/problem-change-tickets/%d/apply", ticket.ID)
	objects.mu.Lock()
	beforeObjects := len(objects.objects)
	objects.mu.Unlock()
	assertStatus(t, ticketMultipart(t, f, "POST", path, f.admin, nil, map[string][]byte{"final.zip": body}, fields), 409)
	objects.mu.Lock()
	afterObjects := len(objects.objects)
	objects.mu.Unlock()
	if beforeObjects != afterObjects {
		t.Fatal("version conflict leaked uploaded package")
	}
	r = f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("suggest", f.problem.ID))
	assertStatus(t, r, 201)
	ticket = readTicket(t, r)
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for _, action := range []string{"resolve", "reject"} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			codes <- f.request("POST", fmt.Sprintf("/api/problem-change-tickets/%d/%s", ticket.ID, action), &f.admin, gin.H{"revision": 1, "note": "并发审核确认结果"}).Code
		}(action)
	}
	wg.Wait()
	close(codes)
	success, conflicts := 0, 0
	for code := range codes {
		if code == 200 {
			success++
		}
		if code == 409 {
			conflicts++
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("concurrent decisions were not serialized")
	}
}

func TestTicketCreateArchiveAndFailedAttachmentCleanupPostgres(t *testing.T) {
	f := newTicketFixture(t)
	objects := setupTicketObjects(t, f)
	f.s.DB.Model(&f.student).Update("can_author", true)
	f.student.CanAuthor = true
	req := ticketDraft("create", 0)
	req["draft_title"], req["draft_statement"], req["target_scope"] = "新题草稿", "## 题面\n计算两数之和。", "prepared"
	r := f.request("POST", "/api/problem-change-tickets", &f.student, req)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	var count int64
	f.s.DB.Model(&models.Problem{}).Count(&count)
	if count != 1 {
		t.Fatal("draft entered problem bank")
	}
	body, _, err := services.BuildProblemPackage(services.ProblemPackageDraft{Slug: "final", Title: "审核后的正式标题", Statement: "正式题面", TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024, Checker: services.CheckerManifest{Type: "exact"}, Cases: []services.ProblemCaseDraft{{Name: "one", Input: "1 2", Output: "3", Weight: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	r = ticketMultipart(t, f, "POST", fmt.Sprintf("/api/problem-change-tickets/%d/apply", ticket.ID), f.admin, nil, map[string][]byte{"final.zip": body}, map[string]string{"revision": "1"})
	assertStatus(t, r, 200)
	completed := readTicket(t, r)
	if completed.ProblemID == nil || completed.Status != "completed" {
		t.Fatal("new problem was not applied")
	}
	var problem models.Problem
	f.s.DB.First(&problem, *completed.ProblemID)
	if problem.Title != "审核后的正式标题" || f.s.isProblemPublic(problem.ID) {
		t.Fatal("draft used as package or prepared scope published")
	}
	r = f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("archive", problem.ID))
	assertStatus(t, r, 201)
	archive := readTicket(t, r)
	r = ticketMultipart(t, f, "POST", fmt.Sprintf("/api/problem-change-tickets/%d/apply", archive.ID), f.admin, nil, nil, map[string]string{"revision": "1", "operation_mode": "overwrite", "expected_version_id": fmt.Sprint(*problem.CurrentVersionID)})
	assertStatus(t, r, 200)
	f.s.DB.First(&problem, problem.ID)
	if problem.ArchivedAt == nil || problem.DeletedAt != nil {
		t.Fatal("archive must preserve problem")
	}
	f.s.DB.Model(&models.ProblemVersion{}).Where("problem_id = ?", problem.ID).Count(&count)
	if count != 1 {
		t.Fatal("archive lost version")
	}
	objects.mu.Lock()
	before := len(objects.objects)
	objects.mu.Unlock()
	if err := f.s.DB.Exec("ALTER TABLE problem_change_tickets ADD CONSTRAINT injected_ticket_failure CHECK (description <> 'inject database failure')").Error; err != nil {
		t.Fatal(err)
	}
	failing := ticketDraft("suggest", f.problem.ID)
	failing["description"] = "inject database failure"
	assertStatus(t, ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, failing, map[string][]byte{"proof.in": []byte("1 2")}, nil), 400)
	objects.mu.Lock()
	after := len(objects.objects)
	objects.mu.Unlock()
	if after != before {
		t.Fatal("failed transaction leaked attachment")
	}
	// The aggregate limit is enforced on actual multipart input, even when each file is valid.
	large := bytes.Repeat([]byte("x"), 25<<20)
	assertStatus(t, ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, ticketDraft("suggest", f.problem.ID), map[string][]byte{"1.in": large, "2.in": large, "3.in": large, "4.in": large}, nil), 400)
}

func TestTicketNewProblemPreviewAndPrivateScopePostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	f.s.DB.Model(&f.student).Update("can_author", true)
	f.student.CanAuthor = true
	req := ticketDraft("create", 0)
	assertStatus(t, f.request("POST", "/api/problem-change-tickets", &f.student, req), 400)
	req["draft_title"] = "新题预览"
	req["draft_statement"] = "## 题目描述\n求 $a+b$。"
	req["target_scope"] = "prepared"
	r := f.request("POST", "/api/problem-change-tickets", &f.student, req)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	if ticket.IsPublic {
		t.Fatal("draft is public")
	}
	assertStatus(t, f.request("GET", fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID), &f.peer, nil), 403)
	// Changing a prepared problem must never publish it through an ordinary feedback ticket.
	f.s.DB.Create(&models.PreparedProblem{ProblemID: f.problem.ID, OwnerID: f.teacher.ID, Difficulty: "入门"})
	r = f.request("POST", "/api/problem-change-tickets", &f.admin, ticketDraft("replace", f.problem.ID))
	assertStatus(t, r, 201)
	var stored models.ProblemChangeTicket
	f.s.DB.First(&stored, readTicket(t, r).ID)
	if stored.TargetScope != "prepared" {
		t.Fatal("feedback can request publication")
	}
}

func TestTicketDescriptionAndAttachmentLimitsPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	f.s.DB.Model(&f.student).Update("can_author", true)
	f.student.CanAuthor = true
	for _, description := range []string{"修改样例", "  补充说明  ", "修正😀题"} {
		req := ticketDraft("suggest", f.problem.ID)
		req["description"] = description
		assertStatus(t, f.request("POST", "/api/problem-change-tickets", &f.student, req), 201)
	}
	req := ticketDraft("create", 0)
	req["draft_title"], req["draft_statement"], req["description"] = "五十附件", "题目描述", "增加测试"
	files := map[string][]byte{}
	for i := 1; i <= 25; i++ {
		files[fmt.Sprintf("%d.in", i)] = []byte("1 2\n")
		files[fmt.Sprintf("%d.out", i)] = []byte("3\n")
	}
	r := ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, req, files, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	if len(ticket.Attachments) != 50 {
		t.Fatalf("want 50 attachments, got %d", len(ticket.Attachments))
	}
	files["extra.in"] = []byte("1")
	assertStatus(t, ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, req, files, nil), 400)
	ids := make([]uint, 0, 50)
	for _, file := range ticket.Attachments {
		ids = append(ids, file.ID)
	}
	url := fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID)
	req["revision"], req["retain_attachment_ids"] = 1, ids
	assertStatus(t, ticketMultipart(t, f, "PATCH", url, f.student, req, map[string][]byte{"extra.in": []byte("1")}, nil), 400)
	req["retain_attachment_ids"] = ids[:49]
	r = ticketMultipart(t, f, "PATCH", url, f.student, req, map[string][]byte{"extra.in": []byte("1")}, nil)
	assertStatus(t, r, 200)
	if len(readTicket(t, r).Attachments) != 50 {
		t.Fatal("replacement did not retain 50 files")
	}
	req["revision"], req["description"] = 2, "😀😀😀"
	assertStatus(t, f.request("PATCH", url, &f.student, req), 400)
	feedback := ticketDraft("suggest", f.problem.ID)
	small := map[string][]byte{}
	for i := 0; i < 21; i++ {
		small[fmt.Sprintf("%d.in", i)] = []byte("1")
	}
	assertStatus(t, ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, feedback, small, nil), 400)
}

func TestTicketBrowserFixture(t *testing.T) {
	if os.Getenv("OJ_TICKET_BROWSER_FIXTURE_ADDR") == "" {
		t.Skip("opt-in browser fixture")
	}
	if os.Getenv("OJ_TICKET_BROWSER_FIXTURE_ADDR") != "127.0.0.1:18087" {
		t.Fatal("fixture must bind approved loopback port")
	}
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	f.s.DB.Model(&f.teacher).Update("can_author", true)
	f.s.Cfg.CORSOrigins = []string{"http://127.0.0.1:4177"}
	f.router = f.s.Router()
	f.teacher.CanAuthor = true
	team := models.Team{Name: "工单验收团队", Slug: "ticket-qa", OwnerID: f.teacher.ID, Visibility: "private", ContestPermission: "all", JoinMode: "open"}
	f.s.DB.Create(&team)
	for _, user := range []models.User{f.teacher, f.student, f.peer} {
		role := models.TeamRoleMember
		if user.ID == f.teacher.ID {
			role = models.TeamRoleOwner
		}
		f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: user.ID, Role: role})
	}
	set := models.TeamProblemSet{TeamID: team.ID, CreatedBy: f.teacher.ID, Title: "反馈验收题单"}
	if err := f.s.DB.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.TeamProblemSetProblem{ProblemSetID: set.ID, ProblemID: f.problem.ID, Label: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range []models.User{f.student, f.peer} {
		for _, public := range []bool{false, true} {
			sub := models.Submission{UserID: user.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Language: "python", SourceCode: "print(3)", Status: models.StatusAccepted, IsPublic: public}
			if err := f.s.DB.Create(&sub).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	start := time.Now().Add(-time.Hour)
	contest := models.TeamContest{TeamID: team.ID, CreatedBy: f.teacher.ID, Title: "工单反馈比赛", Description: "检查反馈只在本场内流转。", StartsAt: &start, PublishedAt: &start, DurationMinutes: 240, State: "published", ScoringRule: "ioi"}
	f.s.DB.Create(&contest)
	f.s.DB.Create(&models.TeamContestProblem{ContestID: contest.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Label: "B"})
	for _, user := range []models.User{f.student, f.peer} {
		f.s.DB.Create(&models.TeamContestParticipant{ContestID: contest.ID, UserID: user.ID})
	}
	f.s.DB.Model(&f.exam).Update("settings", datatypes.JSONMap{"lock_exit": true})
	// Use an independent extra exam so each viewport can enter and finish its own attempt.
	t.Logf("TICKET_FIXTURE_READY problem=%d exam=%d contest=%d", f.problem.ID, f.exam.ID, contest.ID)
	if err := http.ListenAndServe("127.0.0.1:18087", f.router); err != nil {
		t.Fatal(err)
	}
}

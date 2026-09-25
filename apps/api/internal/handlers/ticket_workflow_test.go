package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

func workflowDraft() gin.H {
	return gin.H{"action": "create", "description": "提交可直接入库的完整材料", "target_scope": "public", "draft_title": "无输入题目", "draft_statement": "## 题目描述\n输出数字一。\n\n![示意图](attachments/figure.png)\n\n## 输入格式\n无输入。\n\n## 输出格式\n输出 1。\n\n## 样例\n### 样例 1\n#### 输入\n```text\n\n```\n#### 输出\n```text\n1\n```"}
}

func workflowReview(t *testing.T, f *examTestFixture, id uint) ticketPackageReview {
	t.Helper()
	r := f.request("GET", fmt.Sprintf("/api/problem-change-tickets/%d", id), &f.admin, nil)
	assertStatus(t, r, 200)
	var view struct {
		Review ticketPackageReview `json:"review"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Review.Revision == 0 {
		t.Fatalf("missing review defaults: %s", r.Body.String())
	}
	return view.Review
}

func workflowApply(t *testing.T, f *examTestFixture, id uint, review ticketPackageReview) int {
	t.Helper()
	body, _ := json.Marshal(review)
	fields := map[string]string{"revision": fmt.Sprint(review.Revision), "operation_mode": "overwrite", "review": string(body)}
	if review.ExpectedVersionID != nil {
		fields["expected_version_id"] = fmt.Sprint(*review.ExpectedVersionID)
	}
	r := ticketMultipart(t, f, "POST", fmt.Sprintf("/api/problem-change-tickets/%d/apply", id), f.admin, nil, nil, fields)
	if r.Code != 200 && r.Code != 409 {
		t.Fatalf("apply status %d: %s", r.Code, r.Body.String())
	}
	return r.Code
}

func TestTicketMaterialsReturnResubmitAndApplyPostgres(t *testing.T) {
	f := newTicketFixture(t)
	store := setupTicketObjects(t, f)
	f.student.CanAuthor = true
	f.s.DB.Model(&f.student).Update("can_author", true)
	draft := workflowDraft()
	r := ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, draft, map[string][]byte{"empty.in": {}, "figure.png": ticketPNG(t)}, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	base := fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID)
	review := workflowReview(t, f, ticket.ID)
	assertStatus(t, f.request("POST", base+"/preview", &f.admin, review), 400)
	assertStatus(t, f.request("POST", base+"/preview", &f.student, review), 403)
	assertStatus(t, f.request("POST", base+"/return", &f.peer, gin.H{"revision": 1, "note": "补充输出文件"}), 403)
	assertStatus(t, f.request("POST", base+"/return", &f.admin, gin.H{"revision": 1, "note": ""}), 400)
	assertStatus(t, f.request("POST", base+"/return", &f.admin, gin.H{"revision": 1, "note": "请补充 empty.out"}), 200)
	if workflowApply(t, f, ticket.ID, review) != 409 {
		t.Fatal("returned ticket applied")
	}
	draft["revision"] = 2
	ids := []uint{}
	for _, a := range ticket.Attachments {
		ids = append(ids, a.ID)
	}
	draft["retain_attachment_ids"] = ids
	assertStatus(t, ticketMultipart(t, f, "PATCH", base, f.student, draft, map[string][]byte{"FIGURE.PNG": ticketPNG(t)}, nil), 400)
	r = ticketMultipart(t, f, "PATCH", base, f.student, draft, map[string][]byte{"empty.out": []byte("1\n")}, nil)
	assertStatus(t, r, 200)
	if v := readTicket(t, r); v.Status != "pending" || v.Revision != 3 {
		t.Fatalf("not resubmitted: %+v", v)
	}
	if !strings.Contains(r.Body.String(), "请补充 empty.out") || !strings.Contains(r.Body.String(), "resubmit") {
		t.Fatal("review history lost")
	}
	review = workflowReview(t, f, ticket.ID)
	store.mu.Lock()
	before := len(store.objects)
	store.mu.Unlock()
	r = f.request("POST", base+"/preview", &f.admin, review)
	assertStatus(t, r, 200)
	if !strings.Contains(r.Body.String(), `"case_count":1`) || !strings.Contains(r.Body.String(), "assets/figure.png") {
		t.Fatalf("bad preview: %s", r.Body.String())
	}
	store.mu.Lock()
	after := len(store.objects)
	store.mu.Unlock()
	if before != after {
		t.Fatal("preview wrote objects")
	}
	if workflowApply(t, f, ticket.ID, review) != 200 {
		t.Fatal("material apply failed")
	}
	if workflowApply(t, f, ticket.ID, review) != 409 {
		t.Fatal("duplicate apply accepted")
	}
	var saved models.ProblemChangeTicket
	f.s.DB.First(&saved, ticket.ID)
	var problem models.Problem
	f.s.DB.First(&problem, *saved.ProblemID)
	if problem.OwnerID != f.student.ID || !strings.Contains(problem.Statement, "assets/figure.png") || !f.s.isProblemPublic(problem.ID) {
		t.Fatal("ownership, image or scope lost")
	}
	if bytes.Contains(saved.History, []byte("figure.png")) {
		t.Fatal("history should record decisions, not duplicate data")
	}
	// Changing only the statement preserves test bytes, weights and image resources.
	r = f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("replace", problem.ID))
	assertStatus(t, r, 201)
	modify := readTicket(t, r)
	review = workflowReview(t, f, modify.ID)
	if review.ReplaceTests || len(review.AttachmentIDs) != 0 {
		t.Fatal("replacement tests must be opt-in")
	}
	review.Statement += "\n补充说明。"
	if workflowApply(t, f, modify.ID, review) != 200 {
		t.Fatal("statement-only update failed")
	}
	f.s.DB.First(&problem, problem.ID)
	store.mu.Lock()
	body := append([]byte(nil), store.objects["/ticket-test/"+problem.PackageObject]...)
	store.mu.Unlock()
	pkg, err := services.ParseProblemPackage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Manifest.Cases) != 1 || pkg.Manifest.Cases[0].Weight != 1 || len(pkg.Assets) != 1 {
		t.Fatal("statement edit lost tests or images")
	}
	if !strings.HasSuffix(pkg.Manifest.Statement, "补充说明。") {
		t.Fatal("statement edit missing")
	}
}

func TestTicketRetargetAndFinalRejectionPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	f.student.CanAuthor = true
	f.s.DB.Model(&f.student).Update("can_author", true)
	req := workflowDraft()
	req["target_scope"] = "prepared"
	r := f.request("POST", "/api/problem-change-tickets", &f.student, req)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	base := fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID)
	req["revision"] = 1
	req["target_scope"] = "team_problem_set"
	req["team_problem_set_id"] = 999999
	assertStatus(t, f.request("PATCH", base, &f.student, req), 404)
	req["target_scope"] = "public"
	delete(req, "team_problem_set_id")
	r = f.request("PATCH", base, &f.student, req)
	assertStatus(t, r, 200)
	if !strings.Contains(r.Body.String(), `"target_scope":"public"`) {
		t.Fatal("target scope ignored")
	}
	assertStatus(t, f.request("POST", base+"/reject", &f.admin, gin.H{"revision": 2, "note": "最终不予采纳"}), 200)
	req["revision"] = 3
	assertStatus(t, f.request("PATCH", base, &f.student, req), 409)
}

func TestTicketMaterialPairingAndImages(t *testing.T) {
	files := []ticketMaterial{{"groupA/sample.in", nil}, {"groupA/sample.out", []byte("1")}, {"groupB/sample.in", []byte("2")}, {"groupB/sample.out", nil}, {"images/figure.png", ticketPNG(t)}}
	cases, assets, aliases, err := ticketMaterialParts(files, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 || cases[0].Input != "" || cases[1].Output != "" || len(assets) != 1 {
		t.Fatal("path pairing or empty test lost")
	}
	for _, source := range []string{"![image](images/figure.png)", "![image](attachments/figure.png)", "![image][diagram]\n\n[diagram]: images/figure.png"} {
		result, err := rewriteTicketImages(source, aliases)
		if err != nil || !strings.Contains(result, "assets/figure.png") {
			t.Fatalf("%s => %s, %v", source, result, err)
		}
	}
	if _, err := rewriteTicketImages("![missing](missing.png)", aliases); err == nil {
		t.Fatal("missing image silently accepted")
	}
	if _, _, _, err := ticketMaterialParts(files, false); err == nil {
		t.Fatal("test replacement not explicit")
	}
	if _, _, _, err := ticketMaterialParts(files[:1], true); err == nil {
		t.Fatal("unpaired input accepted")
	}
	code := "```text\n![example](missing.png)\n```\n`![example](missing.png)`\n    ![example](missing.png)\n"
	if result, err := rewriteTicketImages(code, aliases); err != nil || result != code {
		t.Fatalf("modified code example: %s %v", result, err)
	}
	aliases["图 1.png"] = "assets/图 1.png"
	if result, err := rewriteTicketImages("![图](<images/图 1.png>)", aliases); err != nil || !strings.Contains(result, "assets/%E5%9B%BE%201.png") {
		t.Fatalf("encoded image lost: %s %v", result, err)
	}
}

func TestTicketOrphanCanBeClosedWithoutStrandingAuthorPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	r := f.request("POST", "/api/problem-change-tickets", &f.student, ticketDraft("replace", f.problem.ID))
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	base := fmt.Sprintf("/api/problem-change-tickets/%d", ticket.ID)
	if err := f.s.DB.Model(&f.problem).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatal(err)
	}
	assertStatus(t, f.request("GET", base, &f.admin, nil), 200)
	assertStatus(t, f.request("GET", base, &f.student, nil), 403)
	assertStatus(t, f.request("POST", base+"/return", &f.admin, gin.H{"revision": 1, "note": "不可退回到已失效目标"}), 409)
	assertStatus(t, f.request("POST", base+"/reject", &f.admin, gin.H{"revision": 1, "note": "目标题目已删除，关闭此申请"}), 200)
}

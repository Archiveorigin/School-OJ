package handlers

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/http"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/streams"
	"strings"
	"testing"
)

func TestSubmissionOutboxRecoversRedisFailurePostgres(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires isolated Redis")
	}
	ctx := context.Background()
	// This fixture uses its own Redis logical database and deletes only its stream.
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:56379", DB: 15})
	if n := rdb.Exists(ctx, streams.SubmissionStream).Val(); n != 0 {
		t.Fatal("outbox test stream is already in use")
	}
	t.Cleanup(func() { rdb.Del(ctx, streams.SubmissionStream); _ = rdb.Close() })
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, Status: models.StatusQueued, Language: "python3", SourceCode: "print(3)"}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	if err := services.AddSubmissionOutbox(f.s.DB, sub.ID); err != nil {
		t.Fatal(err)
	}
	// A closed transport deterministically simulates Redis becoming unreachable;
	// the database and recovery Redis are real, not mocks.
	offline := redis.NewClient(&redis.Options{Addr: "127.0.0.1:56379", DB: 15})
	_ = offline.Close()
	dispatcher := services.SubmissionOutboxDispatcher{DB: f.s.DB, Redis: offline}
	if n, err := dispatcher.DispatchOnce(ctx); err != nil || n != 0 {
		t.Fatalf("failed publish: %d %v", n, err)
	}
	var pending int64
	f.s.DB.Table("submission_outbox").Where("submission_id = ? AND published_at IS NULL AND locked_until IS NULL AND attempts = 1 AND last_error <> ''", sub.ID).Count(&pending)
	if pending != 1 {
		t.Fatal("Redis outage lost the committed outbox event")
	}
	f.s.DB.Exec("UPDATE submission_outbox SET available_at = CURRENT_TIMESTAMP WHERE submission_id = ?", sub.ID)
	dispatcher.Redis = rdb
	if n, err := dispatcher.DispatchOnce(ctx); err != nil || n != 1 {
		t.Fatalf("recovery: %d %v", n, err)
	}
	if n, err := dispatcher.DispatchOnce(ctx); err != nil || n != 0 {
		t.Fatalf("duplicate publish after success: %d %v", n, err)
	}
	if length := rdb.XLen(ctx, streams.SubmissionStream).Val(); length != 1 {
		t.Fatalf("stream length = %d", length)
	}
	// A dispatcher crash after claiming leaves a lease, which becomes retryable.
	if err := services.AddSubmissionOutbox(f.s.DB, sub.ID); err != nil {
		t.Fatal(err)
	}
	f.s.DB.Exec("UPDATE submission_outbox SET locked_until = CURRENT_TIMESTAMP + INTERVAL '1 hour' WHERE published_at IS NULL")
	if n, err := dispatcher.DispatchOnce(ctx); err != nil || n != 0 {
		t.Fatalf("live claim was stolen: %d %v", n, err)
	}
	f.s.DB.Exec("UPDATE submission_outbox SET locked_until = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE published_at IS NULL")
	if n, err := dispatcher.DispatchOnce(ctx); err != nil || n != 1 {
		t.Fatalf("expired claim not recovered: %d %v", n, err)
	}
}

func TestAPIPrefixesRemainEquivalent(t *testing.T) {
	routes := (Server{}).Router().Routes()
	legacy, versioned := map[string]string{}, map[string]string{}
	for _, route := range routes {
		if strings.HasPrefix(route.Path, "/api/v1/") {
			versioned[route.Method+strings.TrimPrefix(route.Path, "/api/v1")] = route.Handler
		} else if strings.HasPrefix(route.Path, "/api/") {
			legacy[route.Method+strings.TrimPrefix(route.Path, "/api")] = route.Handler
		}
	}
	if len(legacy) == 0 || len(legacy) != len(versioned) {
		t.Fatalf("API counts: %d / %d", len(legacy), len(versioned))
	}
	for key, handler := range legacy {
		if versioned[key] != handler {
			t.Errorf("API compatibility changed for %s", key)
		}
	}
}

func TestTeamInvitationLookupPostgres(t *testing.T) {
	f := newExamFixture(t)
	if f.s.Redis == nil {
		t.Skip("requires dedicated local Redis")
	}
	team := models.Team{Name: "私有邀请团队", Slug: "invitation-only", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "invitation", JoinCode: "ABCDEF012345", Announcement: "private-internal-content"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	for _, prefix := range []string{"/api", "/api/v1"} {
		response := f.request("POST", prefix+"/team-invitations/preview", &f.student, map[string]any{"join_code": "  abcdef012345  "})
		assertStatus(t, response, http.StatusOK)
		for _, secret := range []string{"join_code", team.JoinCode, team.Announcement, "owner_id", "member_count"} {
			if strings.Contains(response.Body.String(), secret) {
				t.Fatalf("preview leaked %s", secret)
			}
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("invitation preview may be cached")
		}
		assertStatus(t, f.request("POST", prefix+"/team-invitations/preview", &f.student, map[string]any{"join_code": "000000000000"}), 403)
	}
	if _, joined := f.s.teamMembership(team.ID, f.student.ID); joined {
		t.Fatal("preview auto-joined")
	}
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/teams/%d/join", team.ID), &f.student, map[string]any{"join_code": "ABCDEF012345"}), 200)
	f.s.DB.Model(&team).Update("join_mode", "application")
	assertStatus(t, f.request("POST", "/api/team-invitations/preview", &f.peer, map[string]any{"join_code": "ABCDEF012345"}), 403)
	assertStatus(t, f.request("POST", fmt.Sprintf("/api/teams/%d/join", team.ID), &f.peer, map[string]any{}), 202)
	assertStatus(t, f.request("POST", "/api/team-invitations/preview", nil, map[string]any{"join_code": team.JoinCode}), 401)
	_ = f.s.Redis.Close()
	assertStatus(t, f.request("POST", "/api/v1/team-invitations/preview", &f.peer, map[string]any{"join_code": team.JoinCode}), 503)
}

func TestTeamJoinAtomicRollbackAndRolesPostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "团队事务验收", Slug: "atomic-team", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "application", JoinCode: "ABCDEF543210"}
	if err := f.s.DB.Create(&team).Error; err != nil {
		t.Fatal(err)
	}
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	if _, err := f.s.teamAccess().Join(context.Background(), team.ID, f.student.ID, "", "申请"); err != nil {
		t.Fatal(err)
	}
	f.s.DB.Model(&team).Update("join_mode", "invitation")
	if err := f.s.DB.Exec("ALTER TABLE team_join_applications ADD CONSTRAINT test_reject_approval CHECK(status <> 'approved')").Error; err != nil {
		t.Fatal(err)
	}
	if joined, err := f.s.teamAccess().Join(context.Background(), team.ID, f.student.ID, team.JoinCode, ""); err == nil || joined {
		t.Fatal("partial membership commit")
	}
	if _, joined := f.s.teamMembership(team.ID, f.student.ID); joined {
		t.Fatal("failed transaction created member")
	}
	f.s.DB.Exec("ALTER TABLE team_join_applications DROP CONSTRAINT test_reject_approval")
	if joined, err := f.s.teamAccess().Join(context.Background(), team.ID, f.student.ID, team.JoinCode, ""); err != nil || !joined {
		t.Fatalf("recovered join: %v", err)
	}
	var application models.TeamJoinApplication
	f.s.DB.Where("team_id = ? AND user_id = ?", team.ID, f.student.ID).First(&application)
	if application.Status != "approved" {
		t.Fatal("old pending application blocks invitation join")
	}
	if err := f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.peer.ID, Role: "teacher"}).Error; err == nil {
		t.Fatal("fourth role accepted")
	}
	if err := f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.peer.ID, Role: models.TeamRoleOwner}).Error; err == nil {
		t.Fatal("two owners accepted")
	}
	if err := f.s.teamAccess().ChangeMembership(context.Background(), f.teacher.ID, team.ID, f.student.ID, "role", models.TeamRoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.Create(&models.TeamJoinApplication{TeamID: team.ID, UserID: f.student.ID, Status: "pending"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.teamAccess().Join(context.Background(), team.ID, f.student.ID, team.JoinCode, ""); err != nil {
		t.Fatal(err)
	}
	var stalePending int64
	f.s.DB.Model(&models.TeamJoinApplication{}).Where("team_id = ? AND user_id = ? AND status = 'pending'", team.ID, f.student.ID).Count(&stalePending)
	if stalePending != 0 {
		t.Fatal("duplicate successful join left stale pending applications")
	}
	membership, _ := f.s.teamMembership(team.ID, f.student.ID)
	if membership.Role != models.TeamRoleAdmin {
		t.Fatal("rejoin overwrote administrator")
	}
	if err := f.s.teamAccess().ChangeMembership(context.Background(), f.student.ID, team.ID, f.teacher.ID, "remove", ""); err == nil {
		t.Fatal("administrator removed creator")
	}
}

func TestTeamSubmissionDetailsRespectTeamScopePostgres(t *testing.T) {
	f := newExamFixture(t)
	team := models.Team{Name: "提交权限", Slug: "submission-access", OwnerID: f.teacher.ID, Visibility: "private", JoinMode: "open", ContestPermission: "admin"}
	f.s.DB.Create(&team)
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.teacher.ID, Role: models.TeamRoleOwner})
	f.s.DB.Create(&models.TeamMembership{TeamID: team.ID, UserID: f.peer.ID, Role: models.TeamRoleAdmin})
	set := models.TeamProblemSet{TeamID: team.ID, Title: "私有题单", CreatedBy: f.teacher.ID}
	f.s.DB.Create(&set)
	sub := models.Submission{UserID: f.student.ID, ProblemID: f.problem.ID, ProblemVersionID: f.version.ID, ProblemSetID: &set.ID, Language: "python", SourceCode: "print(1)", Status: models.StatusAccepted}
	if err := f.s.DB.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf("/api/submissions/%d", sub.ID)
	assertStatus(t, f.request("GET", url, &f.teacher, nil), 200)
	assertStatus(t, f.request("GET", url, &f.peer, nil), 200)
	f.s.DB.Model(&models.TeamMembership{}).Where("team_id = ? AND user_id = ?", team.ID, f.peer.ID).Update("role", models.TeamRoleMember)
	assertStatus(t, f.request("GET", url, &f.peer, nil), 403)
	assertStatus(t, f.request("GET", url, &f.student, nil), 200)
}

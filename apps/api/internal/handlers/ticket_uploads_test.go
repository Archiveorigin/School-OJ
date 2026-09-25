package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
)

func TestTicketStreamingZIPAndSpoolCleanupPostgres(t *testing.T) {
	f := newTicketFixture(t)
	setupTicketObjects(t, f)
	f.s.DB.Model(&f.student).Update("can_author", true)
	f.student.CanAuthor = true
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	req := ticketDraft("create", 0)
	req["draft_title"] = "测试包"
	req["draft_statement"] = "题目说明"
	names := []string{}
	for i := 0; i < 100; i++ {
		names = append(names, fmt.Sprintf("tests/%d.in", i), fmt.Sprintf("tests/%d.out", i))
	}
	files := map[string][]byte{"cases.zip": testTicketZIP(t, names), "large.in": bytes.Repeat([]byte("1"), 5<<20), "large.out": []byte("1")}
	r := ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, req, files, nil)
	assertStatus(t, r, 201)
	ticket := readTicket(t, r)
	if len(ticket.Attachments) != 3 {
		t.Fatal("ZIP should remain one attachment")
	}
	entries, err := os.ReadDir(temp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("multipart spool leaked: %v %v", entries, err)
	}
	files["cases.zip"] = testTicketZIP(t, []string{"1.in"})
	assertStatus(t, ticketMultipart(t, f, "POST", "/api/problem-change-tickets", f.student, req, files, nil), 400)
	entries, err = os.ReadDir(temp)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed parse spool leaked: %v %v", entries, err)
	}
}

func testTicketZIP(t *testing.T, names []string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range names {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("1\n"))
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestInspectTicketZIP(t *testing.T) {
	names := []string{}
	for i := 0; i < 100; i++ {
		names = append(names, fmt.Sprintf("tests/%d.in", i), fmt.Sprintf("tests/%d.out", i))
	}
	body := testTicketZIP(t, names)
	if n, err := inspectTicketZIP(context.Background(), bytes.NewReader(body), int64(len(body))); err != nil || n != 400 {
		t.Fatalf("100 pairs: size=%d err=%v", n, err)
	}
	for _, names := range [][]string{{"1.in"}, {"../1.in", "../1.out"}, {"1.in", "1.out", "1.in"}, {"readme.md"}} {
		body = testTicketZIP(t, names)
		if _, err := inspectTicketZIP(context.Background(), bytes.NewReader(body), int64(len(body))); err == nil {
			t.Fatalf("accepted invalid ZIP: %v", names)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspectTicketZIP(ctx, bytes.NewReader(body), int64(len(body))); err == nil {
		t.Fatal("ignored cancellation")
	}
}

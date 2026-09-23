package db

import (
	"school-oj/apps/api/internal/models"
	"testing"
)

func TestTicketMigration027LegacyRollbackAndRestartPostgres(t *testing.T) {
	database := migrationTestDB(t)
	// Restore the pre-027 shape on an ORM baseline, as used by existing installations.
	if err := database.AutoMigrate(models.AllModels()...); err != nil {
		t.Fatal(err)
	}
	if err := ensureMigrationLedger(database); err != nil {
		t.Fatal(err)
	}
	files, err := embeddedMigrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	if err := baselineLegacyMigrations(database, files); err != nil {
		t.Fatal(err)
	}
	var prior []migrationFile
	var upgrade migrationFile
	for _, file := range files {
		if file.Version < 27 {
			prior = append(prior, file)
		}
		if file.Version == 27 {
			upgrade = file
		}
	}
	if err := applyVersionedMigrations(database, prior); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`DROP TABLE problem_ticket_attachments;
 ALTER TABLE problem_change_tickets DROP COLUMN exam_id, DROP COLUMN contest_id,
 DROP COLUMN problem_version_id, DROP COLUMN draft_title, DROP COLUMN draft_statement,
 DROP COLUMN is_public, DROP COLUMN revision, DROP COLUMN deleted_at;`).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "legacy author", Email: "legacy@example.test", Role: models.RoleTeacher}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`INSERT INTO problem_change_tickets(requester_id, action, status, target_scope, description, attachment_name, attachment_object) VALUES (?, 'create', 'pending', 'prepared', 'legacy request', 'old.zip', 'private/old.zip')`, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	broken := upgrade
	broken.SQL += "; SELECT * FROM injected_missing_table"
	if err := applyVersionedMigrations(database, []migrationFile{broken}); err == nil {
		t.Fatal("expected rollback")
	}
	if database.Migrator().HasColumn("problem_change_tickets", "revision") || database.Migrator().HasTable("problem_ticket_attachments") {
		t.Fatal("027 DDL was not rolled back")
	}
	for i := 0; i < 2; i++ {
		if err := Migrate(database); err != nil {
			t.Fatal(err)
		}
	}
	var ticket models.ProblemChangeTicket
	if err := database.Preload("Attachments").First(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	if ticket.IsPublic || ticket.Revision != 1 || len(ticket.Attachments) != 1 || ticket.Attachments[0].Object != "private/old.zip" {
		t.Fatalf("legacy migration: %+v", ticket)
	}
	var count int64
	database.Table("schema_migrations").Where("version = 27").Count(&count)
	if count != 1 {
		t.Fatalf("027 ledger count %d", count)
	}
	var oldIndex bool
	database.Raw("SELECT to_regclass('idx_problem_change_tickets_active_problem') IS NOT NULL").Scan(&oldIndex)
	if oldIndex {
		t.Fatal("old unique pending ticket restriction remains")
	}
}

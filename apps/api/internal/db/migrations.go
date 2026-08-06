package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	appmigrations "school-oj/apps/api/migrations"

	"gorm.io/gorm"
)

const legacyMigrationVersion = 19

type migrationFile struct {
	Name     string
	Version  int
	SQL      string
	Checksum string
}

// Migrate is the single startup migration entry point. Existing installations
// are baselined through the legacy migrations, while new installations execute
// the complete ordered SQL history. GORM remains a compatibility pass for old
// installations whose historical schema was produced by AutoMigrate.
func Migrate(gdb *gorm.DB) error {
	hadSchema := gdb.Migrator().HasTable("users")
	if err := ensureMigrationLedger(gdb); err != nil {
		return err
	}
	if err := gdb.Exec("SELECT pg_advisory_lock(?)", int64(734625109)).Error; err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer gdb.Exec("SELECT pg_advisory_unlock(?)", int64(734625109))

	files, err := embeddedMigrationFiles()
	if err != nil {
		return err
	}
	if hadSchema {
		if err := baselineLegacyMigrations(gdb, files); err != nil {
			return err
		}
	}
	if err := applyVersionedMigrations(gdb, files); err != nil {
		return err
	}
	if err := runModelCompatibilityMigrations(gdb); err != nil {
		return fmt.Errorf("model compatibility migration: %w", err)
	}
	return nil
}

func ensureMigrationLedger(gdb *gorm.DB) error {
	return gdb.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  checksum VARCHAR(64) NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`).Error
}

func embeddedMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(appmigrations.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version %q: %w", entry.Name(), err)
		}
		content, err := appmigrations.Files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		digest := sha256.Sum256(content)
		files = append(files, migrationFile{Name: entry.Name(), Version: version, SQL: string(content), Checksum: hex.EncodeToString(digest[:])})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	for index := 1; index < len(files); index++ {
		if files[index-1].Version == files[index].Version {
			return nil, fmt.Errorf("duplicate migration version %d", files[index].Version)
		}
	}
	return files, nil
}

func baselineLegacyMigrations(gdb *gorm.DB, files []migrationFile) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, file := range files {
			if file.Version > legacyMigrationVersion {
				break
			}
			if err := tx.Exec(`
INSERT INTO schema_migrations(version, name, checksum, applied_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (version) DO NOTHING`, file.Version, file.Name, file.Checksum, time.Now()).Error; err != nil {
				return fmt.Errorf("baseline migration %s: %w", file.Name, err)
			}
		}
		return nil
	})
}

func applyVersionedMigrations(gdb *gorm.DB, files []migrationFile) error {
	for _, file := range files {
		var existing struct {
			Name     string
			Checksum string
		}
		err := gdb.Table("schema_migrations").Select("name", "checksum").Where("version = ?", file.Version).Take(&existing).Error
		if err == nil {
			if existing.Name != file.Name || existing.Checksum != file.Checksum {
				return fmt.Errorf("migration %03d changed after it was applied", file.Version)
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("inspect migration %s: %w", file.Name, err)
		}
		if err := gdb.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(file.SQL).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)`, file.Version, file.Name, file.Checksum).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", file.Name, err)
		}
	}
	return nil
}

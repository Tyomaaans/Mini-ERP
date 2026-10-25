package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"mini-erp/migrations"
)

func Migrate(ctx context.Context, db *gorm.DB, down bool) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	connection, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "SELECT pg_advisory_lock(73410201)"); err != nil {
		return err
	}
	defer connection.ExecContext(context.Background(), "SELECT pg_advisory_unlock(73410201)")
	if _, err := connection.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY, name TEXT NOT NULL, checksum VARCHAR(64) NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	files, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}
	var names []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			names = append(names, file.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version, err := strconv.ParseInt(strings.SplitN(name, "_", 2)[0], 10, 64)
		if err != nil {
			return err
		}
		contents, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(contents)
		checksum := hex.EncodeToString(hash[:])
		var stored string
		err = connection.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version = $1", version).Scan(&stored)
		if err == nil && stored != checksum {
			return fmt.Errorf("migration %s was modified after being applied", name)
		}
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if down || err == nil {
			continue
		}
		if err := applyMigration(ctx, connection, version, name, checksum, string(contents), false); err != nil {
			return err
		}
	}
	if !down {
		return nil
	}
	var version int64
	var name string
	err = connection.QueryRowContext(ctx, "SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version, &name)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	contents, err := migrations.Files.ReadFile(strings.TrimSuffix(name, ".up.sql") + ".down.sql")
	if err != nil {
		return err
	}
	return applyMigration(ctx, connection, version, name, "", string(contents), true)
}

func applyMigration(ctx context.Context, connection *sql.Conn, version int64, name, checksum, contents string, down bool) error {
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, contents); err != nil {
		return fmt.Errorf("migration %s: %w", name, err)
	}
	if down {
		_, err = tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", version)
	} else {
		_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)", version, name, checksum)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

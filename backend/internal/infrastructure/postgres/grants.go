package postgres

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

func GrantApplication(ctx context.Context, db *gorm.DB, role string) error {
	if role == "" {
		return nil
	}
	identifier := `"` + strings.ReplaceAll(role, `"`, `""`) + `"`
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{
			"GRANT USAGE ON SCHEMA public TO " + identifier,
			"GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO " + identifier,
			"REVOKE ALL ON schema_migrations FROM " + identifier,
			"REVOKE UPDATE, DELETE ON stock_movements, audit_logs FROM " + identifier,
		} {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

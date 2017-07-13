package store

import (
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) Exec(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection) (int, error) {
	return migrate.Exec(db.(*sqlx.DB).DB, dialect, m, dir)
}

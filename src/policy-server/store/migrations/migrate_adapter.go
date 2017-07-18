package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/rubenv/sql-migrate"
	"errors"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) Exec(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection) (int, error) {
	if db, ok := db.(*sqlx.DB); ok {
		return migrate.Exec(db.DB, dialect, m, dir)
	}

	return 0, errors.New("unable to adapt for db migration")
}

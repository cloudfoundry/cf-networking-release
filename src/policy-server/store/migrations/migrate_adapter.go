package migrations

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/rubenv/sql-migrate"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, max int) (int, error) {
	if db, ok := db.(*sqlx.DB); ok {
		return migrate.ExecMax(db.DB, dialect, m, dir, max) // tested through integration
	}

	return 0, errors.New("unable to adapt for db migration")
}

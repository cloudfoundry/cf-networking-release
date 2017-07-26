package migrations

import (
	"errors"
	"time"

	"github.com/cf-container-networking/sql-migrate"
	"github.com/jmoiron/sqlx"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, max int) (int, error) {
	if dir == migrate.Down {
		return 0, errors.New("down migration not supported")
	}
	if db, ok := db.(*sqlx.DB); ok {
		return migrate.ExecMaxWithLock(db.DB, dialect, m, dir, max, 1*time.Minute) // tested through integration
	}

	return 0, errors.New("unable to adapt for db migration")
}

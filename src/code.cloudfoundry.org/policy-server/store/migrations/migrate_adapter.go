package migrations

import (
	"errors"

	migrate "github.com/rubenv/sql-migrate"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, max int) (int, error) {
	if dir == migrate.Down {
		return 0, errors.New("down migration not supported")
	}

	return migrate.ExecMax(db.RawConnection().DB, dialect, m, dir, max) // tested through integration
}

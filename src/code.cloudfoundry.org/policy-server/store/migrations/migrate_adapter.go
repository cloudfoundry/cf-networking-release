package migrations

import (
	"errors"
	"time"

	migrate "github.com/cf-container-networking/sql-migrate"
)

type MigrateAdapter struct {
}

func (ma *MigrateAdapter) ExecMax(db MigrationDb, dialect string, m migrate.MigrationSource, dir migrate.MigrationDirection, max int) (int, error) {
	if dir == migrate.Down {
		return 0, errors.New("down migration not supported")
	}

	return migrate.ExecMaxWithLock(db.RawConnection().DB, dialect, m, dir, max, 1*time.Minute) // tested through integration
}

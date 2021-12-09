package store_test

//counterfeiter:generate -o fakes/sql_result.go --fake-name SqlResult . result
type result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

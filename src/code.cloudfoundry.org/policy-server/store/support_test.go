package store_test

//go:generate counterfeiter -o fakes/sql_result.go --fake-name SqlResult . result
type result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

package helpers

import (
	"fmt"
	"strings"
)

const (
	MySQL    = "mysql"
	Postgres = "postgres"
)

func QuestionMarks(count int) string {
	if count == 0 {
		return ""
	}
	return strings.Repeat("?, ", count-1) + "?"
}

func RebindForSQLDialect(query, dialect string) string {
	if dialect == MySQL {
		return query
	}
	if dialect != Postgres {
		panic(fmt.Sprintf("Unrecognized DB dialect '%s'", dialect))
	}

	strParts := strings.Split(query, "?")
	for i := 1; i < len(strParts); i++ {
		strParts[i-1] = fmt.Sprintf("%s$%d", strParts[i-1], i)
	}
	return strings.Join(strParts, "")
}

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
	return MarksWithSeparator(count, "?", ", ")
}

func MarksWithSeparator(count int, mark string, separator string) string {
	if count == 0 {
		return ""
	}
	return strings.Repeat(mark+separator, count-1) + mark
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

func RebindForSQLDialectAndMark(query, dialect, mark string) string {
	if dialect != Postgres && dialect != MySQL {
		panic(fmt.Sprintf("Unrecognized DB dialect '%s'", dialect))
	}

	if dialect == MySQL {
		return strings.ReplaceAll(query, mark, "?")
	}

	strParts := strings.Split(query, mark)
	for i := 1; i < len(strParts); i++ {
		strParts[i-1] = fmt.Sprintf("%s$%d", strParts[i-1], i)
	}
	return strings.Join(strParts, "")
}

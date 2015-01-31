package sillyquill_rt

import "bytes"
import "fmt"

func BuildInsertQuery(
	tableName string,
	loadColumnNames []string,
	saveColumnNames []string) string {
	var buf bytes.Buffer
	(&buf).WriteString("INSERT INTO ")
	(&buf).WriteString(tableName)
	(&buf).WriteRune('(')
	for _, v := range saveColumnNames {
		(&buf).WriteRune('"')
		(&buf).WriteString(v)
		(&buf).WriteRune('"')
		(&buf).WriteRune(',')
	}
	(&buf).Truncate(buf.Len() - 1)
	(&buf).WriteString(") VALUES(")

	for i := range saveColumnNames {
		fmt.Fprintf(&buf, "$%d,", i+1)
	}
	(&buf).Truncate(buf.Len() - 1)
	(&buf).WriteString(") RETURNING ")
	for _, v := range loadColumnNames {
		(&buf).WriteRune('"')
		(&buf).WriteString(v)
		(&buf).WriteRune('"')
		(&buf).WriteRune(',')
	}
	(&buf).Truncate(buf.Len() - 1)

	return buf.String()
}

func BuildUpdateQuery(
	tableName string,
	columns []string) string {
	var buf bytes.Buffer
	(&buf).WriteString("UPDATE ")
	(&buf).WriteString(tableName)
	(&buf).WriteString(" SET ")
	for i, v := range columns {
		fmt.Fprintf(&buf, "%q=$%d,", v, i+1)
	}
	(&buf).Truncate(buf.Len() - 1)

	return buf.String()
}

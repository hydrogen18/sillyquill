package sillyquill_rt

import "bytes"
import "fmt"

func BuildInsertQuery(
	w *bytes.Buffer,
	tableName string,
	loadColumnNames []string,
	saveColumnNames []string) {
	fmt.Fprint(w, "INSERT INTO ")
	fmt.Fprint(w, tableName)
	fmt.Fprint(w, "(")
	for _, v := range saveColumnNames {
		fmt.Fprint(w, `"`, v, `",`)
	}
	w.Truncate(w.Len() - 1)
	fmt.Fprint(w, ") VALUES(")

	for i := range saveColumnNames {
		fmt.Fprintf(w, "$%d,", i+1)
	}
	w.Truncate(w.Len() - 1)
	fmt.Fprint(w, ") RETURNING ")
	for _, v := range loadColumnNames {
		fmt.Fprint(w, `"`, v, `",`)
	}
	w.Truncate(w.Len() - 1)
}

func BuildUpdateQuery(
	w *bytes.Buffer,
	tableName string,
	columns []string) {

	w.WriteString("UPDATE ")
	w.WriteString(tableName)
	w.WriteString(" SET ")
	for i, v := range columns {
		fmt.Fprintf(w, "%q=$%d,", v, i+1)
	}
	w.Truncate(w.Len() - 1)

}

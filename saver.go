package main

//import "fmt"

type ColumnSaver struct {
	TheColumnType       *ColumnType
	TheColumnizedStruct *ColumnizedStruct
}

func NewColumnSaverFor(s *ColumnizedStruct,
	columnInterfaces *ColumnType) *ColumnSaver {
	this := new(ColumnSaver)
	this.TheColumnType = columnInterfaces
	this.TheColumnizedStruct = s

	return this
}

func (this *ColumnSaver) Imports() []string {
	return []string{"github.com/hydrogen18/sillyquill/rt",
		"database/sql",
		"bytes",
	}
}

func (this *ColumnSaver) Emit(pw *panicWriter) error {
	pw.fprintLn("func (this *%s) updateColumnsWhere(db *sql.DB,where %s,columns ...%s) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.TheColumnType.ListTypeName,
		this.TheColumnType.InterfaceName,
	)
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn(`(&buf).WriteString("UPDATE %s SET ")`, this.TheColumnizedStruct.TableName)
	pw.fprintLn("for i, v := range columns {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q=$%%d",v.Name(),i+1)`)
	pw.deindent()
	pw.fprintLn("}") // end for
	pw.fprintLn(`(&buf).WriteString(" WHERE ")`)
	pw.fprintLn("(&buf).WriteString(where.andEqualClauseOf(len(columns)+1))")
	pw.fprintLn("var args []interface{}")
	pw.fprintLn("args = %s(columns).ValuesOf(this)", this.TheColumnType.ListTypeName)
	pw.fprintLn("args = append(args,where.ValuesOf(this)...)")
	pw.fprintLn("result, err := db.Exec((&buf).String(),args...)")
	//TODO chcek result.RowsAffected
	pw.fprintLn("_ = result")
	pw.fprintLn("return err")
	pw.fprintLn("}")
	return nil
}

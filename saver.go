package main

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
		"fmt",
	}
}

func (this *ColumnSaver) Suffix() string {
	return "_saver"
}

func (this *ColumnSaver) Emit(pw *panicWriter) error {
	//--Emit a low level wrapper for UPDATE
	pw.fprintLn("func (this *%s) updateColumnsWhere(db *sql.DB,where %s,columns ...%s) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.TheColumnType.ListTypeName,
		this.TheColumnType.InterfaceName,
	)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn("%s.BuildUpdateQuery(&buf, %q,%s(columns).Names())",
		sillyquil_runtime_pkg_name,
		this.TheColumnizedStruct.TableName,
		this.TheColumnType.ListTypeName)
	pw.fprintLn(`(&buf).WriteString(" WHERE ")`)
	pw.fprintLn(`%s.BuildAndEqualClause(&buf,len(columns)+1,where.Names())`,
		sillyquil_runtime_pkg_name)

	pw.fprintLn("var args []interface{}")
	pw.fprintLn("args = %s(columns).ValuesOf(this)", this.TheColumnType.ListTypeName)
	pw.fprintLn("args = append(args,where.ValuesOf(this)...)")

	pw.fprintLn("result, err := db.Exec((&buf).String(),args...)")
	//check result.RowsAffected
	pw.fprintLn("if err == nil {")
	pw.indent()
	pw.fprintLn("var rowsAffected int64")
	pw.fprintLn("rowsAffected, err = result.RowsAffected()")
	pw.fprintLn("if err == nil && rowsAffected != 1 {")
	pw.indent()
	pw.fprintLn("return %s.RowDoesNotExistError{Instance: this}", sillyquil_runtime_pkg_name)
	pw.deindent()
	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")

	//Emit a low level wrapper for INSERT
	pw.fprintLn("func (this *%s) insertColumns(db *sql.DB, columnsToLoad %s, columnsToSave %s) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.TheColumnType.ListTypeName,
		this.TheColumnType.ListTypeName,
	)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn("%s.BuildInsertQuery(&buf,%q,columnsToLoad.Names(),columnsToSave.Names())",
		sillyquil_runtime_pkg_name,
		this.TheColumnizedStruct.TableName)
	pw.fprintLn("args := columnsToSave.ValuesOf(this)")
	pw.fprintLn("result := db.QueryRow((&buf).String(),args...)")
	pw.fprintLn("return this.loadWithColumns(columnsToLoad,result)")
	pw.deindent()
	pw.fprintLn("}")

	//Emit a low level wrapper for find-or-create
	pw.fprintLn("func (this *%s) findOrCreateColumnsWhere(db *sql.DB, where, columnsToSave, columnsToLoad %s) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.TheColumnType.ListTypeName,
	)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")

	pw.fprintLn(`(&buf).WriteString("With extant_row AS (SELECT ")`)
	pw.fprintLn("for _, v := range columnsToLoad {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q,",v.Name())`)
	pw.deindent()
	pw.fprintLn("}") // end for

	pw.fprintLn("(&buf).Truncate(buf.Len()-1)")
	pw.fprintLn(`(&buf).WriteString(" FROM %s WHERE ")`,
		this.TheColumnizedStruct.TableName)
	pw.fprintLn(`%s.BuildAndEqualClause(&buf, 1, where.Names())`,
		sillyquil_runtime_pkg_name)
	pw.fprintLn(`(&buf).WriteString("), new_row as ( INSERT INTO %s (")`,
		this.TheColumnizedStruct.TableName)
	pw.fprintLn("for _, v := range columnsToSave {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q,",v.Name())`)
	pw.deindent()
	pw.fprintLn("}") // end for

	pw.fprintLn("(&buf).Truncate(buf.Len()-1)")
	pw.fprintLn(`(&buf).WriteString(") SELECT ")`)
	pw.fprintLn("for i, v := range columnsToSave {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"$%%d as %%q,",len(where)+i+1,v.Name())`)
	pw.deindent()
	pw.fprintLn("}") // end for
	pw.fprintLn("(&buf).Truncate(buf.Len()-1)")
	pw.fprintLn(`(&buf).WriteString(" WHERE NOT EXISTS ( ")`)
	pw.fprintLn(`(&buf).WriteString("SELECT 1 from extant_row LIMIT 1 ) ")`)
	pw.fprintLn(`(&buf).WriteString("RETURNING ")`)
	pw.fprintLn("for _, v := range columnsToLoad {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q,",v.Name())`)
	pw.deindent()
	pw.fprintLn("}") // end for
	pw.fprintLn("(&buf).Truncate(buf.Len()-1)")
	pw.fprintLn(`(&buf).WriteString(") SELECT * from extant_row UNION ALL ")`)
	pw.fprintLn(`(&buf).WriteString("SELECT * from new_row  ")`)

	pw.fprintLn("args := where.ValuesOf(this)")
	pw.fprintLn("args = append(args, columnsToSave.ValuesOf(this)...)")
	pw.fprintLn("result := db.QueryRow((&buf).String(),args...)")
	//TODO check and wrap sql.ErrNoRows
	pw.fprintLn("return this.loadWithColumns(columnsToLoad,result)")
	pw.deindent()
	pw.fprintLn("}")

	return nil
}

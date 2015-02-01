package main

import "fmt"

type ColumnLoader struct {
	ColumnAnalyzerFunctionName  string
	LoadManyFunctionName        string
	LoadWithColumnsReceiverName string

	TheColumnType       *ColumnType
	TheColumnizedStruct *ColumnizedStruct
}

func NewColumnLoaderFor(s *ColumnizedStruct,
	columnInterfaces *ColumnType) *ColumnLoader {
	this := new(ColumnLoader)
	this.ColumnAnalyzerFunctionName = fmt.Sprintf("analyzeColumnsFor%s",
		s.PluralModelName)

	this.LoadWithColumnsReceiverName = fmt.Sprintf("loadWithColumns")
	this.LoadManyFunctionName = fmt.Sprintf("LoadMany%s",
		s.PluralModelName)
	this.TheColumnType = columnInterfaces
	this.TheColumnizedStruct = s

	return this
}

func (this *ColumnLoader) Suffix() string {
	return "_loader"
}

func (this *ColumnLoader) Imports() []string {
	return []string{
		"github.com/hydrogen18/sillyquill/rt",
		"database/sql",
		"bytes",
		"fmt",
	}
}

func (this *ColumnLoader) Emit(pw *panicWriter) error {
	//--Emit a function that maps column names to a list of
	//instances of the Column Interface
	pw.fprintLn("func %s(names []string) (%s, error) {",
		this.ColumnAnalyzerFunctionName,
		this.TheColumnType.ListTypeName,
	)
	pw.indent()
	pw.fprintLn("result := make(%s, len(names))",
		this.TheColumnType.ListTypeName)

	pw.fprintLn("for i, name := range names {")
	pw.indent()
	pw.fprintLn("switch(name) {")

	for _, defn := range this.TheColumnType.Defns {
		pw.fprintLn("case %q:",
			defn.ColumnName)
		pw.indent()
		pw.fprintLn("result[i] = %s",
			defn.InstanceName)
		pw.deindent()
	}

	pw.fprintLn("default:")

	pw.indent()
	pw.fprintLn("return nil, %s.UnknownColumnError{Index: i, Name: name}", sillyquil_runtime_pkg_name)

	pw.deindent()
	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return result,nil")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//--Emit a receiver that loads a list of columns
	//from a Scanner like sql.Rows
	pw.fprintLn("func (this *%s) %s(columns %s, scanner sillyquill_rt.Scanner) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.LoadWithColumnsReceiverName,
		this.TheColumnType.ListTypeName,
	)
	pw.indent()

	pw.fprintLn("args := columns.PointersTo(this)")
	pw.fprintLn("err := scanner.Scan(args...)")
	pw.fprintLn("if err != nil { ")
	pw.indent()
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("columns.SetLoaded(this,true)")
	pw.fprintLn("return nil")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit a fuction that loads a list of the model type
	//from a type like sql.Rows
	pw.fprintLn("func %s(rows %s.Rows) (%s,error) {",
		this.LoadManyFunctionName,
		sillyquil_runtime_pkg_name,
		this.TheColumnizedStruct.ListTypeName,
	)
	pw.indent()
	pw.fprintLn("columnNames, err := rows.Columns()")
	pw.returnIf("err != nil", "nil,err")

	pw.fprintLn("columns, err := %s(columnNames)",
		this.ColumnAnalyzerFunctionName)
	pw.returnIf("err != nil", "nil, err")
	pw.fprintLn("var result %s",
		this.TheColumnizedStruct.ListTypeName)
	pw.fprintLn("for rows.Next() {")
	pw.indent()
	pw.fprintLn("var m %s",
		this.TheColumnizedStruct.SingularModelName,
	)
	pw.fprintLn("err = (&m).loadWithColumns(columns,rows)")
	pw.fprintLn("if err != nil {")
	pw.indent()
	pw.fprintLn("rows.Close()")
	pw.fprintLn("return nil, err")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("result = append(result,m)")
	pw.deindent()
	pw.fprintLn("}")
	pw.returnIf("rows.Err() != nil", "nil, rows.Err()")
	pw.fprintLn("return result, nil")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit a receiver that loads a list of columns
	//based on another set of columns in the instance
	pw.fprintLn("func (this *%s) loadColumnsWhere(db *sql.DB, where %s,columns ...%s) error {",
		this.TheColumnizedStruct.SingularModelName,
		this.TheColumnType.ListTypeName,
		this.TheColumnType.InterfaceName)

	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn(`(&buf).WriteString("Select ")`)
	pw.fprintLn("for _, column := range columns {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q,",column.Name())`)
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("(&buf).Truncate((&buf).Len() - 1)")
	pw.fprintLn("(&buf).WriteString(` from %q where `)", this.TheColumnizedStruct.TableName)

	pw.fprintLn(`%s.BuildAndEqualClause(&buf,1,where.Names())`,
		sillyquil_runtime_pkg_name)

	pw.fprintLn("row := db.QueryRow(buf.String(),where.ValuesOf(this)...)")
	pw.fprintLn("return this.loadWithColumns(columns,row)")
	pw.deindent()
	pw.fprintLn("}")

	return nil
}

package main

import "fmt"
import "io"
import "github.com/spiceworks/spicelog"
import "reflect"
import "time"
import "strings"

type ModelEmitter struct {
	TableNameToCodeName  func(string) string
	ColumnNameToCodeName func(string) string
	ColumnToDataType     func(Column) []interface{}
	Tab                  string
	Package              string
}

func NewModelEmitter() *ModelEmitter {
	return &ModelEmitter{
		TableNameToCodeName:  UnderscoresToCamelCase,
		ColumnNameToCodeName: UnderscoresToCamelCase,
		ColumnToDataType:     columnToDataType,
		Tab:                  "    ",
	}
}

func UnderscoresToCamelCase(v string) string {
	var output []rune

	var in_name bool
	for _, c := range v {
		if c == '_' && in_name {
			in_name = false
		} else if !in_name {
			c = []rune(strings.ToUpper(string(c)))[0]
			in_name = true
			output = append(output, c)
		} else {
			output = append(output, c)
		}
	}

	return string(output)
}

type panicWriter struct {
	io.Writer
	level int
	tab   string
}

func (this *panicWriter) Write(data []byte) (int, error) {
	n, err := this.Writer.Write(data)
	if err != nil {
		panic(err)
	}

	return n, nil
}

func (this *panicWriter) indent() {
	this.level++
}
func (this *panicWriter) deindent() {
	this.level--
	if this.level == -1 {
		panic("negative indent")
	}
}

func (this *panicWriter) fprintLn(fmtStr string, args ...interface{}) {
	for i := this.level; i != 0; i-- {
		fmt.Fprint(this, this.tab)
	}
	fmt.Fprintf(this, fmtStr, args...)
	fmt.Fprint(this, "\n")
}

func (this *panicWriter) returnIf(ifClause, returnClause string) {
	this.fprintLn("if %s {", ifClause)
	this.indent()
	this.fprintLn("return %s", returnClause)
	this.deindent()
	this.fprintLn("}")
}

func (pw *panicWriter) printDataType(dt []interface{}) error {
	rk, ok := dt[0].(reflect.Kind)
	if !ok {
		return fmt.Errorf("First element not %T", rk)
	}
	i := 1

	if rk == reflect.Ptr {
		fmt.Fprintf(pw, "*")

		rk, ok = dt[i].(reflect.Kind)
		i++
		if !ok {
			return fmt.Errorf("First element not %T", rk)
		}
	}

	switch rk {
	case reflect.Int32,
		reflect.Int64,
		reflect.Bool,
		reflect.String,
		reflect.Float64:
		fmt.Fprintf(pw, "%v", rk)
	case reflect.Struct:
		fmt.Fprintf(pw, "%T", dt[i])
		i++
	case reflect.Slice:
		fmt.Fprintf(pw, "[]%v", dt[i])
		i++
	default:
		return fmt.Errorf("Not convertable %v", dt)
	}

	return nil

}

func tempModelNamer(v string) (string, string) {
	pluralName := UnderscoresToCamelCase(v)

	singularName := pluralName[0 : len(pluralName)-1]

	return pluralName, singularName
}

func (this *ModelEmitter) Emit(table Table, w io.Writer) error {
	pw := &panicWriter{
		Writer: w,
		level:  0,
		tab:    this.Tab,
	}
	var emitters []CodeEmitter
	columnizedStruct, err := NewColumnizedStruct(table,
		tempModelNamer,
		this.ColumnNameToCodeName,
		this.ColumnToDataType)

	if err != nil {
		return err
	}
	emitters = append(emitters, columnizedStruct)

	columnType := NewColumnType(columnizedStruct)
	columnizedStruct.TheColumnType = columnType

	emitters = append(emitters, columnType)

	columnLoader := NewColumnLoaderFor(columnizedStruct,
		columnType)

	emitters = append(emitters, columnLoader)

	pw.fprintLn("package %s", this.Package)

	imports := make(map[string]int)

	for _, emitter := range emitters {
		emitterImports := emitter.Imports()
		for _, v := range emitterImports {
			imports[v] = 0
		}
	}
	pw.fprintLn("")

	for v, _ := range imports {
		pw.fprintLn("import %q", v)
	}
	pw.fprintLn("")

	for _, emitter := range emitters {
		err := emitter.Emit(pw)
		if err != nil {
			return err
		}
	}

	return nil

	uniqueColumns, err := table.Unique()
	if err != nil {
		return err
	}

	isIdentifiable := len(uniqueColumns) != 0

	primaryKeyColumns, err := table.PrimaryKey()
	if err != nil {
		return err
	}

	isIdentifiable = isIdentifiable || len(primaryKeyColumns) != 0

	pw.fprintLn("package %s", this.Package)
	pw.fprintLn("")
	pw.fprintLn(`import "database/sql"`)
	pw.fprintLn(`import "bytes"`)
	pw.fprintLn(`import "fmt"`)

	pw.fprintLn(`import "github.com/hydrogen18/sillyquill/rt"`)

	columns, err := table.Columns()
	if err != nil {
		return err
	}

	//TODO - figure out how to remove hack ?
	for _, column := range columns {
		if column.DataType() == SqlTimestamp ||
			column.DataType() == SqlDate {
			pw.fprintLn(`import "time"`)
			break
		}
	}
	pw.fprintLn("")

	fmt.Fprintf(pw, "//Table Name:%s\n", table.Name())
	modelName := this.TableNameToCodeName(table.Name())
	singluarModelName := modelName[0 : len(modelName)-1]
	fmt.Fprintf(pw, "type %s struct {\n", singluarModelName)

	spicelog.V(6).Infof("columns:%v", columns)

	for _, column := range columns {
		columnName := this.ColumnNameToCodeName(column.Name())
		fmt.Fprintf(pw, "%s%s ",
			this.Tab,
			columnName,
		)

		dt := this.ColumnToDataType(column)

		if len(dt) == 0 {
			return fmt.Errorf("No data type for %q", column.Name())
		}

		pw.printDataType(dt)

		fmt.Fprintf(pw, " //Column Name:%s\n", column.Name())
	}

	fmt.Fprintf(pw, "\n")
	fmt.Fprintf(pw, "%sisLoaded struct {\n", this.Tab)
	for _, column := range columns {
		columnName := this.ColumnNameToCodeName(column.Name())
		fmt.Fprintf(pw, "%s%s%s bool\n",
			this.Tab,
			this.Tab,
			columnName)

	}

	fmt.Fprintf(pw, "%s}\n", this.Tab)
	fmt.Fprintf(pw, "}\n")
	fmt.Fprintf(pw, "\n")

	//---

	pw.fprintLn("func (this *%s) String() string {", singluarModelName)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn(`fmt.Fprintf(&buf,"%s{")`, singluarModelName)
	for i, column := range columns {
		columnName := this.ColumnNameToCodeName(column.Name())

		pw.fprintLn("if this.isLoaded.%s {", columnName)
		pw.indent()

		dt := this.ColumnToDataType(column)
		if dt[0] != reflect.Ptr {
			pw.fprintLn(`fmt.Fprintf(&buf,"%s:%%v",this.%s)`,
				columnName,
				columnName)
		} else {
			pw.fprintLn("if this.%s != nil {", columnName)
			pw.indent()
			pw.fprintLn(`fmt.Fprintf(&buf,"%s:%%v",*this.%s)`,
				columnName,
				columnName,
			)
			pw.deindent()
			pw.fprintLn("} else {")
			pw.indent()
			pw.fprintLn(`fmt.Fprintf(&buf,"%s:nil")`,
				columnName,
			)
			pw.deindent()
			pw.fprintLn("}")

		}

		if i != len(columns)-1 {
			pw.fprintLn(`(&buf).WriteRune(' ')`)
		}
		pw.deindent()

		pw.fprintLn("}")

	}
	pw.fprintLn(`(&buf).WriteRune('}')`)
	pw.fprintLn("return (&buf).String()")
	pw.deindent()
	pw.fprintLn("}")

	//---

	privateModelName := strings.ToLower(modelName[0:1])
	privateModelName = privateModelName + modelName[1:]

	columnEnumType := fmt.Sprintf("%sColumnEnum", privateModelName)
	pw.fprintLn("type %s int", columnEnumType)
	columnNameToEnum := make(map[string]string)

	for i, column := range columns {
		columnName := this.TableNameToCodeName(column.Name())
		tname := fmt.Sprintf("%sColumn%s",
			modelName,
			columnName)

		pw.fprintLn("const %s = %s(%d) //Column Name:%s",
			tname,
			columnEnumType,
			i+1,
			column.Name())

		columnNameToEnum[column.Name()] = tname
	}
	//---
	pw.fprintLn("")

	for _, column := range columns {
		columnName := this.ColumnNameToCodeName(column.Name())
		fmt.Fprintf(pw, "func (this *%s) Get%s(db *sql.DB) (",
			singluarModelName,
			columnName,
		)

		dt := this.ColumnToDataType(column)
		err = pw.printDataType(dt)
		if err != nil {
			return err
		}
		fmt.Fprintf(pw, ", error) {\n")
		pw.indent()

		pw.fprintLn("if this.isLoaded.%s {", columnName)
		pw.indent()
		pw.fprintLn("return this.%s, nil", columnName)
		pw.deindent()
		pw.fprintLn("}")

		if !isIdentifiable {
			pw.fprintLn("return this.%s, %s.RowNotUniquelyIdentifiableError{Instance:this, TableName:%q}",
				columnName,
				sillyquil_runtime_pkg_name,
				table.Name())
		} else {
			pw.fprintLn("uniqueIdentifiers, err := this.identifyingColumns()")
			//TODO return the zero-value of the data type
			pw.returnIf("err != nil", fmt.Sprintf("this.%s, err", columnName))

			pw.fprintLn("err = this.loadColumnsWhere(db,uniqueIdentifiers,%s)",
				columnNameToEnum[column.Name()])
			pw.fprintLn("return this.%s, err", columnName)
		}
		pw.deindent()
		pw.fprintLn("}")
	}
	pw.fprintLn("")

	//---
	pw.fprintLn("")
	pw.fprintLn("func (this %s) String() string{", columnEnumType)
	pw.indent()
	pw.fprintLn("switch(this) {")

	for columnName, enumName := range columnNameToEnum {
		pw.fprintLn("case %s:", enumName)
		pw.indent()
		pw.fprintLn("return %q", columnName)
		pw.deindent()
	}
	pw.fprintLn("}")
	pw.fprintLn("return \"\"")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//---
	pw.fprintLn("")
	pw.fprintLn("func (this %s) pointerInto(m *%s) interface{} {", columnEnumType,
		singluarModelName)
	pw.indent()
	pw.fprintLn("switch(this) {")

	for columnName, enumName := range columnNameToEnum {
		pw.fprintLn("case %s:", enumName)
		pw.indent()
		pw.fprintLn("return &m.%s", this.ColumnNameToCodeName(columnName))
		pw.deindent()
	}
	pw.fprintLn("}")
	pw.fprintLn("return nil")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//--

	pw.fprintLn("func (this %s) setLoaded(m *%s) {", columnEnumType, singluarModelName)

	pw.indent()
	pw.fprintLn("switch(this) {")

	for columnName, enumName := range columnNameToEnum {
		pw.fprintLn("case %s:", enumName)
		pw.indent()
		pw.fprintLn("m.isLoaded.%s = true", this.ColumnNameToCodeName(columnName))
		pw.deindent()
	}

	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//---

	pw.fprintLn("func (this %s) valueOf(m *%s) interface{} {", columnEnumType, singluarModelName)

	pw.indent()
	pw.fprintLn("switch(this) {")

	for columnName, enumName := range columnNameToEnum {
		pw.fprintLn("case %s:", enumName)
		pw.indent()
		pw.fprintLn("return m.%s", this.ColumnNameToCodeName(columnName))
		pw.deindent()
	}

	pw.fprintLn("}")
	pw.fprintLn("return nil")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//---

	columnEnumListType := fmt.Sprintf("%sList", columnEnumType)
	pw.fprintLn("type %s []%s", columnEnumListType, columnEnumType)

	pw.fprintLn("func (this %s) pointersInto(m *%s) []interface{} {",
		columnEnumListType,
		singluarModelName,
	)

	pw.indent()
	pw.fprintLn("result := make([]interface{},len(this))")
	pw.fprintLn("for i, v := range this {")
	pw.indent()
	pw.fprintLn("result[i] = v.pointerInto(m)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return result")
	pw.deindent()

	pw.fprintLn("}")
	pw.fprintLn("")

	pw.fprintLn("func (this %s) setLoaded(m *%s) {", columnEnumListType, singluarModelName)
	pw.indent()
	pw.fprintLn("for _, v := range this {")
	pw.indent()
	pw.fprintLn("v.setLoaded(m)")
	pw.deindent()
	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")

	//---

	if isIdentifiable {
		pw.fprintLn("func (this %s) andClauseOf(m *%s, pi int) string {",
			columnEnumListType,
			singluarModelName,
		)

		pw.indent()
		pw.fprintLn("var buf bytes.Buffer")
		pw.fprintLn("for i, v := range this {")
		pw.indent()
		pw.fprintLn("(&buf).WriteString(v.String())")
		pw.fprintLn(`(&buf).WriteRune('=')`)
		pw.fprintLn(`fmt.Fprintf((&buf),"$%%d",pi)`)
		pw.fprintLn("pi++")
		pw.fprintLn("if i != len(this) - 1 {")
		pw.indent()
		pw.fprintLn(`(&buf).WriteString("and")`)
		pw.deindent()
		pw.fprintLn("}")
		pw.deindent()
		pw.fprintLn("}")
		pw.fprintLn("return (&buf).String()")
		pw.deindent()

		pw.fprintLn("}")
		pw.fprintLn("")
	}
	//---

	if isIdentifiable {
		pw.fprintLn("func (this %s) valuesOf(m *%s) []interface{} {",
			columnEnumListType,
			singluarModelName,
		)

		pw.indent()
		pw.fprintLn("result := make([]interface{},len(this))")
		pw.fprintLn("for i, v := range this {")
		pw.indent()
		pw.fprintLn("result[i] = v.valueOf(m)")
		pw.deindent()
		pw.fprintLn("}")
		pw.fprintLn("return result")
		pw.deindent()

		pw.fprintLn("}")
		pw.fprintLn("")
	}
	//---

	columnAnalyzerFunctionName := fmt.Sprintf("analyze%sColumns",
		modelName)
	pw.fprintLn("func %s(names []string) (%s, error) {", columnAnalyzerFunctionName, columnEnumListType)
	pw.indent()
	pw.fprintLn("var result %s", columnEnumListType)

	pw.fprintLn("for i, name := range names {")
	pw.indent()
	pw.fprintLn("switch(name) {")

	for _, column := range columns {
		pw.fprintLn("case %q:",
			column.Name())
		pw.indent()
		pw.fprintLn("result = append(result,%s)",
			columnNameToEnum[column.Name()])
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

	//---
	pw.fprintLn("func (this *%s) loadWithColumns(columns %s, scanner sillyquill_rt.Scanner) error {",
		singluarModelName,
		columnEnumListType,
	)
	pw.indent()

	pw.fprintLn("args := columns.pointersInto(this)")
	pw.fprintLn("err := scanner.Scan(args...)")
	pw.fprintLn("if err != nil { ")
	pw.indent()
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("columns.setLoaded(this)")
	pw.fprintLn("return nil")
	pw.deindent()

	pw.fprintLn("}")

	//---

	if isIdentifiable {
		pw.fprintLn("func (this *%s) identifyingColumns() (%s,error) {",
			singluarModelName,
			columnEnumListType,
		)

		pw.indent()
		for _, column := range uniqueColumns {

			structName := this.ColumnNameToCodeName(column)
			pw.fprintLn("if this.isLoaded.%s {", structName)
			pw.indent()
			pw.fprintLn("return %s{%s}, nil", columnEnumListType, columnNameToEnum[column])
			pw.deindent()
			pw.fprintLn("}")

		}

		if len(primaryKeyColumns) != 0 {
			pw.fprintLn("if ")
			for i, column := range primaryKeyColumns {
				var operator string

				if i != len(primaryKeyColumns)-1 {
					operator = "&&"
				} else {
					operator = "{"
				}
				columnName := this.ColumnNameToCodeName(column)
				pw.fprintLn("this.isLoaded.%s %s",
					columnName,
					operator)

			}
			pw.indent()
			pw.fprintLn("return %s{", columnEnumListType)
			for _, column := range primaryKeyColumns {

				pw.fprintLn("%s,",
					columnNameToEnum[column])

			}
			pw.fprintLn("}, nil")

			pw.deindent()
			pw.fprintLn("}")
		}

		pw.fprintLn("return nil, %s.RowNotUniquelyIdentifiableError{Instance:*this,TableName:%q}",
			sillyquil_runtime_pkg_name,
			table.Name())
		pw.deindent()

		pw.fprintLn("}")
	}

	//---

	pw.fprintLn("")

	pw.fprintLn("func LoadMany%s(rows %s.Rows) ([]%s,error) {",
		modelName,
		sillyquil_runtime_pkg_name,
		singluarModelName,
	)
	pw.indent()
	pw.fprintLn("columnNames, err := rows.Columns()")
	pw.returnIf("err != nil", "nil,err")

	pw.fprintLn("columns, err := %s(columnNames)", columnAnalyzerFunctionName)
	pw.returnIf("err != nil", "nil, err")
	pw.fprintLn("var result []%s", singluarModelName)
	pw.fprintLn("for rows.Next() {")
	pw.indent()
	pw.fprintLn("var m %s", singluarModelName)
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

	if isIdentifiable {
		pw.fprintLn("func (this *%s) loadColumnsWhere(db *sql.DB, where %s,columns ...%s) error {",
			singluarModelName,
			columnEnumListType,
			columnEnumType)

		pw.indent()
		pw.fprintLn("var buf bytes.Buffer")
		pw.fprintLn(`(&buf).WriteString("Select ")`)
		pw.fprintLn("for i, column := range columns {")
		pw.indent()
		pw.fprintLn(`fmt.Fprintf(&buf,"%%q",column)`)
		pw.fprintLn("if i != len(columns) - 1 {")
		pw.indent()
		pw.fprintLn(`(&buf).WriteString(",")`)
		pw.deindent()
		pw.fprintLn("}")
		pw.deindent()
		pw.fprintLn("}")
		pw.fprintLn("(&buf).WriteString(` from %q where `)", table.Name())
		pw.fprintLn("(&buf).WriteString(where.andClauseOf(this,1))")

		pw.fprintLn("row := db.QueryRow(buf.String(),where.valuesOf(this)...)")
		pw.fprintLn("return this.loadWithColumns(columns,row)")
		pw.deindent()
		pw.fprintLn("}")
	}

	return nil
}

const sillyquil_runtime_pkg_name = "sillyquill_rt"

func columnToDataType(c Column) []interface{} {
	dt := c.DataType()

	var stub []interface{}

	if c.Nullable() {
		stub = []interface{}{reflect.Ptr}
	}

	switch dt {
	case SqlInt:
		return append(stub, reflect.Int32)
	case SqlBigInt, SqlNumeric: //TODO fix this should use big.Int
		return append(stub, reflect.Int64)
	case SqlBoolean:
		return append(stub, reflect.Bool)
	case SqlTimestamp, SqlDate:
		return append(stub, reflect.Struct, time.Time{})
	case SqlVarChar, SqlText:
		return append(stub, reflect.String)
	case SqlFloat64:
		return append(stub, reflect.Float64)
	case SqlByteArray:
		return append(stub, reflect.Slice, reflect.Uint8)
	}

	return nil
}

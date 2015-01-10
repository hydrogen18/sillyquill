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

func (this *ModelEmitter) Emit(table Table, w io.Writer) error {
	pw := &panicWriter{
		Writer: w,
		level:  0,
		tab:    this.Tab,
	}
	pw.fprintLn("package %s", this.Package)
	pw.fprintLn("")
	pw.fprintLn(`import "github.com/hydrogen18/sillyquill/rt"`)

	columns, err := table.Columns()
	if err != nil {
		return err
	}

	//TODO - figure out how to remove hack ?
	for _, column := range columns {
		if column.DataType() == SqlTimestamp {
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

		rk, ok := dt[0].(reflect.Kind)
		if !ok {
			panic(dt[0])
		}

		switch rk {
		case reflect.Int32,
			reflect.Int64,
			reflect.Bool,
			reflect.String:
			fmt.Fprintf(pw, "%v", rk)
		case reflect.Struct:
			fmt.Fprintf(pw, "%T", dt[1])
		case reflect.Slice:
			fmt.Fprintf(pw, "[]%v", dt[1])
		}

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

	return nil
}

const sillyquil_runtime_pkg_name = "sillyquill_rt"

func columnToDataType(c Column) []interface{} {
	dt := c.DataType()

	switch dt {
	case SqlInt:
		return []interface{}{reflect.Int32}
	case SqlBigInt:
		return []interface{}{reflect.Int64}
	case SqlBoolean:
		return []interface{}{reflect.Bool}
	case SqlTimestamp:
		return []interface{}{reflect.Struct, time.Time{}}
	case SqlVarChar:
		return []interface{}{reflect.String}
	case SqlByteArray:
		return []interface{}{reflect.Slice, reflect.Uint8}
	}

	return nil
}

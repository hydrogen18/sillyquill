package main

import "fmt"
import "io"

import "github.com/spiceworks/spicelog"
import "reflect"
import "time"
import "strings"
import "os"
import "path/filepath"
import "github.com/hydrogen18/sillyquill/rt"

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
	v, err := dataTypeToString(dt)
	if err != nil {
		return err
	}
	fmt.Fprintf(pw, v)
	return nil

}

func tempModelNamer(v string) (string, string) {
	name := UnderscoresToCamelCase(v)

	var pluralName string
	var singularName string
	if name[len(name)-1] == 's' {
		pluralName = name
		singularName = pluralName[0 : len(pluralName)-1]
	} else {
		singularName = name
		pluralName = name + "s"
	}

	return pluralName, singularName
}

func (this *ModelEmitter) writeToFile(emitter CodeEmitter, filename string) error {
	w, err := os.OpenFile(filename,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0644)
	if err != nil {
		return err
	}
	defer w.Close()
	spicelog.Infof("Writing file %q", filename)
	pw := &panicWriter{
		Writer: w,
		level:  0,
		tab:    this.Tab,
	}

	pw.fprintLn("package %s", this.Package)

	imports := make(map[string]int)

	emitterImports := emitter.Imports()
	for _, v := range emitterImports {
		imports[v] = 0
	}

	pw.fprintLn("")

	for v, _ := range imports {
		pw.fprintLn("import %q", v)
	}
	pw.fprintLn("")

	err = emitter.Emit(pw)
	return err
}

func (this *ModelEmitter) Emit(table Table, outputPath string) error {

	columnizedStruct, err := NewColumnizedStruct(table,
		tempModelNamer,
		this.ColumnNameToCodeName,
		this.ColumnToDataType)

	if err != nil {
		return err
	}

	columnType := NewColumnType(columnizedStruct)
	columnizedStruct.TheColumnType = columnType

	columnLoader := NewColumnLoaderFor(columnizedStruct,
		columnType)

	columnSaver := NewColumnSaverFor(columnizedStruct,
		columnType)

	for _, emitter := range []CodeEmitter{
		columnizedStruct,
		columnType,
		columnLoader,
		columnSaver,
	} {
		filename := fmt.Sprintf("%s%s.go",
			columnizedStruct.TableName,
			emitter.Suffix())
		filename = filepath.Join(outputPath, filename)
		err = this.writeToFile(emitter, filename)
		if err != nil {
			return err
		}
	}
	fks, err := table.ForeignKeys()
	for _, v := range fks {
		spicelog.Infof("Table %q Foreign Key %q", table.Name(), v)
	}
	return nil
}

const sillyquil_runtime_pkg_name = "sillyquill_rt"

func columnToDataType(c Column) []interface{} {
	dt := c.DataType()

	var stub []interface{}

	var isPtr bool
	if c.Nullable() {
		stub = []interface{}{reflect.Ptr}
		isPtr = true
	}

	switch dt {
	case SqlSmallInt:
		return append(stub, reflect.Int16)
	case SqlInt:
		return append(stub, reflect.Int32)
	case SqlBigInt:
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
	case SqlReal:
		return append(stub, reflect.Float32)
	case SqlNumeric:
		if isPtr {
			return append(stub, reflect.Struct, sillyquill_rt.NullNumeric{})
		} else {
			return append(stub, reflect.Struct, sillyquill_rt.Numeric{})
		}
	}

	return nil
}

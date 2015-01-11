package main

import "fmt"
import "github.com/spiceworks/spicelog"
import "reflect"
import "strings"
import "bytes"
import "time"

type CodeEmitter interface {
	Emit(*panicWriter) error
	Imports() []string
}

type ColumnizedField struct {
	DataTypeDefn []interface{}
	Name         string
	DataType     string
}

type ColumnizedStruct struct {
	PluralModelName   string
	SingularModelName string
	ListTypeName      string
	Fields            []ColumnizedField
	Columns           []Column
	PrimaryKey        []ColumnizedField
	Unique            []ColumnizedField
	TableName         string

	TheColumnType *ColumnType
}

func NewColumnizedStruct(t Table,
	tableNameToStructNames func(string) (string, string),
	columnNameToFieldName func(string) string,
	columnnToDataType func(Column) []interface{}) (*ColumnizedStruct, error) {
	this := new(ColumnizedStruct)
	this.TableName = t.Name()

	this.PluralModelName, this.SingularModelName = tableNameToStructNames(t.Name())
	this.ListTypeName = fmt.Sprintf("%sList", this.SingularModelName)

	spicelog.Infof("Table %q Plural %q Singular %q",
		t.Name(),
		this.PluralModelName,
		this.SingularModelName)
	var err error
	this.Columns, err = t.Columns()
	if err != nil {
		return nil, err
	}

	primaryKey, err := t.PrimaryKey()
	if err != nil {
		return nil, err
	}
	primaryKeys := make(map[string]int)
	for _, v := range primaryKey {
		primaryKeys[v] = 0
	}

	unique, err := t.Unique()
	if err != nil {
		return nil, err
	}
	uniques := make(map[string]int)
	for _, v := range unique {
		uniques[v] = 0
	}

	for _, column := range this.Columns {
		field := ColumnizedField{}

		field.Name = columnNameToFieldName(column.Name())
		field.DataTypeDefn = columnToDataType(column)

		field.DataType, err = dataTypeToString(field.DataTypeDefn)
		if err != nil {
			return nil, err
		}

		this.Fields = append(this.Fields, field)
		spicelog.Infof("Table %q Column %q Field %q",
			t.Name(),
			column.Name(),
			field.Name)

		_, ok := uniques[column.Name()]
		if ok {
			this.Unique = append(this.Unique, field)
		}

		_, ok = primaryKeys[column.Name()]
		if ok {
			this.PrimaryKey = append(this.PrimaryKey, field)
		}

	}

	return this, nil
}

func dataTypeToString(dt []interface{}) (string, error) {
	buf := new(bytes.Buffer)
	rk, ok := dt[0].(reflect.Kind)
	if !ok {
		return "", fmt.Errorf("First element not %T", rk)
	}
	i := 1

	if rk == reflect.Ptr {
		fmt.Fprintf(buf, "*")

		rk, ok = dt[i].(reflect.Kind)
		i++
		if !ok {
			return "", fmt.Errorf("First element not %T", rk)
		}
	}

	switch rk {
	case reflect.Int32,
		reflect.Int64,
		reflect.Bool,
		reflect.String,
		reflect.Float64:
		fmt.Fprintf(buf, "%v", rk)
	case reflect.Struct:
		fmt.Fprintf(buf, "%T", dt[i])
		i++
	case reflect.Slice:
		fmt.Fprintf(buf, "[]%v", dt[i])
		i++
	default:
		return "", fmt.Errorf("Not convertable %v", dt)
	}

	return buf.String(), nil

}

func (this *ColumnizedStruct) Imports() []string {

	var result []string
	result = append(result, "bytes")
	result = append(result, "fmt")
	result = append(result, "database/sql")
	for _, field := range this.Fields {
		var i int
		kind, ok := field.DataTypeDefn[i].(reflect.Kind)
		i++
		if ok {

			if kind == reflect.Ptr {
				kind, ok = field.DataTypeDefn[i].(reflect.Kind)
				i++
				if !ok {
					continue
				}
			}

			if kind == reflect.Struct {
				_, ok := field.DataTypeDefn[i].(time.Time)
				i++
				if ok {
					result = append(result, "time")
				}
			}

		}
	}

	return result
}

func (this *ColumnizedStruct) Emit(pw *panicWriter) error {
	//--Emit a definition of the model
	pw.fprintLn("type %s struct {",
		this.SingularModelName)
	pw.indent()
	for i, field := range this.Fields {
		column := this.Columns[i]
		pw.fprintLn("%s %s //Column:%s",
			field.Name,
			field.DataType,
			column.Name())
	}
	pw.fprintLn("")
	//--Emit a nested struct that has a boolean
	//indicating if each column is loaded
	pw.fprintLn("IsLoaded struct {")
	pw.indent()
	for i, field := range this.Fields {
		column := this.Columns[i]
		pw.fprintLn("%s bool //Column:%s",
			field.Name,
			column.Name())
	}
	pw.deindent()
	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")
	//--Emit a type definition for a list of the model type
	pw.fprintLn("type %s []%s", this.ListTypeName, this.SingularModelName)

	pw.fprintLn("")
	//--Emit a String() for the model type
	pw.fprintLn("func (this *%s) String() string {", this.SingularModelName)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn(`(&buf).WriteString("%s{")`, this.SingularModelName)
	for _, field := range this.Fields {

		pw.fprintLn("if this.IsLoaded.%s {", field.Name)
		pw.indent()

		if field.DataTypeDefn[0] != reflect.Ptr {
			pw.fprintLn(`fmt.Fprintf(&buf,"%s:%%v, ",this.%s)`,
				field.Name,
				field.Name)
		} else {
			pw.fprintLn("if this.%s != nil {", field.Name)
			pw.indent()
			pw.fprintLn(`fmt.Fprintf(&buf,"%s:%%v, ",*this.%s)`,
				field.Name,
				field.Name,
			)
			pw.deindent()
			pw.fprintLn("} else {")
			pw.indent()
			pw.fprintLn(`(&buf).WriteString("%s:nil, ")`,
				field.Name,
			)
			pw.deindent()
			pw.fprintLn("}")

		}

		pw.deindent()

		pw.fprintLn("}")

	}
	pw.fprintLn(`(&buf).WriteRune('}')`)
	pw.fprintLn("return (&buf).String()")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit an accessor for field of the struct
	for i, field := range this.Fields {
		pw.fprintLn("func (this *%s) Get%s(db *sql.DB) (%s, error) {",
			this.SingularModelName,
			field.Name,
			field.DataType)
		pw.indent()
		pw.fprintLn("if this.IsLoaded.%s {", field.Name)
		pw.indent()
		pw.fprintLn("return this.%s, nil", field.Name)
		pw.deindent()
		pw.fprintLn("}")

		pw.fprintLn("columns, err := this.identifyingColumns()")
		pw.fprintLn("if err != nil {")
		pw.indent()
		pw.fprintLn("return this.%s, err", field.Name)
		pw.deindent()
		pw.fprintLn("}")

		pw.fprintLn("return this.%s, this.loadColumnsWhere(db,columns,%s)",
			field.Name,
			this.TheColumnType.Defns[i].InstanceName)
		pw.deindent()
		pw.fprintLn("}")
		pw.fprintLn("")
	}

	return nil
}

type ColumnType struct {
	InterfaceName string
	ListTypeName  string
	Defns         []ColumnTypeDefn
	Parent        *ColumnizedStruct
}

type ColumnTypeDefn struct {
	ColumnName   string
	TypeName     string
	InstanceName string
	FieldName    string
	Nullable     bool
}

func privatizeTypeName(v string) string {
	result := strings.ToLower(string(v[0]))
	result = result + v[1:]
	return result
}

func NewColumnType(that *ColumnizedStruct) *ColumnType {
	this := new(ColumnType)
	this.Parent = that
	this.InterfaceName = fmt.Sprintf("%sColumn", that.SingularModelName)
	this.ListTypeName = fmt.Sprintf("%sColumnList", that.SingularModelName)
	spicelog.Infof("Model %q column type %q",
		that.SingularModelName,
		this.InterfaceName)
	for i, column := range that.Columns {
		field := that.Fields[i]

		defn := ColumnTypeDefn{}
		defn.ColumnName = column.Name()
		defn.FieldName = field.Name
		defn.TypeName = fmt.Sprintf("%sColumn%s",
			privatizeTypeName(that.SingularModelName),
			field.Name)
		defn.InstanceName = fmt.Sprintf("%sColumn%s",
			that.SingularModelName,
			field.Name,
		)
		defn.Nullable = column.Nullable()

		this.Defns = append(this.Defns, defn)
	}

	return this
}

func (this *ColumnType) Imports() []string {
	return nil
}

func (this *ColumnType) ColumnTypeInstanceByFieldName(fieldName string) string {
	for _, defn := range this.Defns {
		if defn.FieldName == fieldName {
			return defn.InstanceName
		}
	}
	panic(fieldName)
}

func (this *ColumnType) Emit(pw *panicWriter) error {
	//--Emit type definition for column interface type
	pw.fprintLn("type %s interface { ",
		this.InterfaceName)
	pw.indent()
	pw.fprintLn(" Name() string")
	pw.fprintLn(" PointerTo(m *%s) interface{}",
		this.Parent.SingularModelName)
	pw.fprintLn(" ValueOf(m *%s) interface{}",
		this.Parent.SingularModelName)
	pw.fprintLn(" SetLoaded(m *%s,isLoaded bool)",
		this.Parent.SingularModelName)
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")
	//--Emit type definition for slice of column interface type
	pw.fprintLn("type %s []%s", this.ListTypeName, this.InterfaceName)
	//--Emit function to call PointerTo on each element and return
	//the result as a slice
	pw.fprintLn("func(this %s) PointersTo(m *%s) []interface{} {",
		this.ListTypeName,
		this.Parent.SingularModelName,
	)
	pw.indent()
	pw.fprintLn("result := make([]interface{},len(this))")
	pw.fprintLn("for i, v := range this {")
	pw.indent()
	pw.fprintLn("result[i] = v.PointerTo(m)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return result")
	pw.deindent()
	pw.fprintLn("}")
	//--Emit function to call SetLoaded on each element
	pw.fprintLn("func(this %s) SetLoaded(m *%s,isLoaded bool)  {",
		this.ListTypeName,
		this.Parent.SingularModelName,
	)
	pw.indent()

	pw.fprintLn("for _, v := range this {")
	pw.indent()
	pw.fprintLn("v.SetLoaded(m,isLoaded)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")

	//--Emit function to call ValueOf on each element
	pw.fprintLn("func(this %s) ValuesOf(m *%s) []interface{} {",
		this.ListTypeName,
		this.Parent.SingularModelName,
	)
	pw.indent()
	pw.fprintLn("result := make([]interface{},len(this))")
	pw.fprintLn("for i, v := range this {")
	pw.indent()
	pw.fprintLn("result[i] = v.ValueOf(m)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return result")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit function to return an and clause representing each column
	pw.fprintLn("func(this %s) andEqualClauseOf(parameterIndex int) string {",
		this.ListTypeName,
	)
	pw.indent()
	pw.fprintLn("var buf bytes.Buffer")
	pw.fprintLn("for i, v := range this {")
	pw.indent()
	pw.fprintLn(`fmt.Fprintf(&buf,"%%q",v.Name())`)
	pw.fprintLn(`(&buf).WriteRune('=')`)
	pw.fprintLn(`fmt.Fprintf(&buf,"$%%d", i + parameterIndex)`)
	pw.fprintLn("")
	pw.fprintLn("if i != len(this) - 1 {")
	pw.indent()
	pw.fprintLn(`(&buf).WriteString(" and ")`)
	pw.deindent()
	pw.fprintLn("}")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return buf.String()")
	pw.deindent()
	pw.fprintLn("}")

	for _, defn := range this.Defns {
		//--Emit type definition for each column. The value is meaningless
		pw.fprintLn("type %s int", defn.TypeName)
		//--Emit instantiation of a singleton of each column type
		pw.fprintLn("var %s = new(%s)", defn.InstanceName, defn.TypeName)
		//--
		pw.fprintLn("func (%s) Name() string {",
			defn.TypeName,
		)
		pw.indent()
		pw.fprintLn("return %q", defn.ColumnName)
		pw.deindent()
		pw.fprintLn("}")
		//---

		pw.fprintLn("func (%s) PointerTo(m *%s) interface{} {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		pw.fprintLn("return &m.%s", defn.FieldName)
		pw.deindent()
		pw.fprintLn("}")
		//---
		pw.fprintLn("func (%s) ValueOf(m *%s) interface{} {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		if defn.Nullable {
			pw.fprintLn("if m.%s != nil {", defn.FieldName)
			pw.indent()
			pw.fprintLn("return *m.%s", defn.FieldName)
			pw.deindent()
			pw.fprintLn("}")
			pw.fprintLn("return nil")
		} else {
			pw.fprintLn("return m.%s", defn.FieldName)
		}
		pw.deindent()
		pw.fprintLn("}")

		//---

		pw.fprintLn("func (%s) SetLoaded(m *%s, v bool) {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		pw.fprintLn("m.IsLoaded.%s = v", defn.FieldName)
		pw.deindent()
		pw.fprintLn("}")

		pw.fprintLn("")
	}

	//--Emit a receiver that returns the minimum set of identifying columns
	//for the type
	pw.fprintLn("func (this %s) identifyingColumns() (%s, error) {",
		this.Parent.SingularModelName,
		this.ListTypeName,
	)

	pw.indent()
	for _, u := range this.Parent.Unique {
		pw.fprintLn("if this.IsLoaded.%s {",
			u.Name)
		pw.indent()
		pw.fprintLn("return %s{%s}, nil",
			this.ListTypeName,
			this.ColumnTypeInstanceByFieldName(u.Name),
		)
		pw.deindent()
		pw.fprintLn("}")
	}

	if len(this.Parent.PrimaryKey) != 0 {
		var pkLoaded []string
		for _, pk := range this.Parent.PrimaryKey {
			pkLoaded = append(pkLoaded,
				fmt.Sprintf("this.IsLoaded.%s", pk.Name))
		}

		pw.fprintLn("if %s {",
			strings.Join(pkLoaded, " && "))
		pw.indent()
		pw.fprintLn("return %s{", this.ListTypeName)

		for _, pk := range this.Parent.PrimaryKey {
			pw.fprintLn("%s,", this.ColumnTypeInstanceByFieldName(pk.Name))
		}

		pw.fprintLn("}, nil")
		pw.deindent()
		pw.fprintLn("}")
	}

	pw.fprintLn("return nil, %s.RowNotUniquelyIdentifiableError{Instance: this}",
		sillyquil_runtime_pkg_name)
	pw.deindent()
	pw.fprintLn("}")
	return nil
}

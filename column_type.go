package main

import "fmt"
import "strings"
import "github.com/spiceworks/spicelog"

type ColumnType struct {
	InterfaceName         string
	ListTypeName          string
	Defns                 []ColumnTypeDefn
	Parent                *ColumnizedStruct
	AllColumnsName        string
	PrimaryKeyColumnsName string
}

type ColumnTypeDefn struct {
	ColumnName   string
	TypeName     string
	InstanceName string
	FieldName    string
	Nullable     bool
	Index        int
	DataType     SqlDataType
}

func privatizeTypeName(v string) string {
	result := strings.ToLower(string(v[0]))
	result = result + v[1:]
	return result
}

func (this *ColumnType) Suffix() string {
	return "_columns"
}

func NewColumnType(that *ColumnizedStruct) *ColumnType {
	this := new(ColumnType)
	this.Parent = that
	this.InterfaceName = fmt.Sprintf("%sColumn", that.SingularModelName)
	this.ListTypeName = fmt.Sprintf("%sColumnList", that.SingularModelName)
	this.AllColumnsName = fmt.Sprintf("%sColumns", that.PluralModelName)
	this.PrimaryKeyColumnsName = fmt.Sprintf("%sPrimaryKeyColumns", that.PluralModelName)
	spicelog.Infof("Model %q column type %q",
		that.SingularModelName,
		this.InterfaceName)
	for i, column := range that.Columns {
		field := that.Fields[i]

		defn := ColumnTypeDefn{}
		defn.Index = i
		defn.DataType = column.DataType()
		defn.ColumnName = column.Name()
		defn.FieldName = field.Name
		defn.TypeName = fmt.Sprintf("%sColumn%s",
			privatizeTypeName(that.SingularModelName),
			field.Name)

		defn.InstanceName = fmt.Sprintf("%s.%s",
			that.PluralModelName,
			field.Name,
		)
		defn.Nullable = column.Nullable()

		this.Defns = append(this.Defns, defn)
	}

	return this
}

func (this *ColumnType) Imports() []string {
	return []string{
		"github.com/hydrogen18/sillyquill/rt",
	}
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
	pw.fprintLn("Name() string")
	pw.fprintLn("Index() int")
	pw.fprintLn("PointerTo(m *%s) interface{}",
		this.Parent.SingularModelName)
	pw.fprintLn("ValueOf(m *%s) interface{}",
		this.Parent.SingularModelName)
	pw.fprintLn("SetLoaded(m *%s,isLoaded bool)",
		this.Parent.SingularModelName)
	pw.fprintLn("SetSet(m *%s,isSet bool)",
		this.Parent.SingularModelName)
	pw.fprintLn("IsLoaded(m *%s) bool",
		this.Parent.SingularModelName)
	pw.fprintLn("IsSet(m *%s) bool",
		this.Parent.SingularModelName)
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("")
	//--Emit type definition for slice of column interface type
	pw.fprintLn("type %s []%s", this.ListTypeName, this.InterfaceName)
	//--Emit a function to return the names
	pw.fprintLn("func(this %s) Names() []string {", this.ListTypeName)
	pw.indent()
	pw.fprintLn("var names []string")
	pw.fprintLn("names = make([]string,len(this))")
	pw.fprintLn("for i, v := range this {")
	pw.indent()
	pw.fprintLn("names[i] = v.Name()")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return names")
	pw.deindent()
	pw.fprintLn("}")

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

	//--Emit function to call SetSet on each element
	pw.fprintLn("func(this %s) SetSet(m *%s,isSet bool)  {",
		this.ListTypeName,
		this.Parent.SingularModelName,
	)
	pw.indent()

	pw.fprintLn("for _, v := range this {")
	pw.indent()
	pw.fprintLn("v.SetSet(m,isSet)")
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

	//--Emit function to check for the presence of a column in the list
	pw.fprintLn("func(this %s) Contains(c %s) bool {",
		this.ListTypeName,
		this.InterfaceName,
	)
	pw.indent()
	pw.fprintLn("for _, v := range this {")
	pw.indent()
	pw.returnIf("v.Index() == c.Index()", "true")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return false")
	pw.deindent()
	pw.fprintLn("}")

	//Create an instance of an anonymous struct with the plural
	//name as the identifier
	pw.fprintLn("var %s = struct {", this.Parent.PluralModelName)
	pw.indent()
	for _, defn := range this.Defns {
		pw.fprintLn("%s %s", defn.FieldName, this.InterfaceName)
	}
	pw.deindent()
	pw.fprintLn("}{")
	pw.indent()
	//--Emit instantiation of a singleton of each column type
	//as the members
	for _, defn := range this.Defns {
		pw.fprintLn("%s : new(%s),", defn.FieldName, defn.TypeName)
	}
	pw.deindent()
	pw.fprintLn("}")

	//Create a list that is all the column types
	pw.fprintLn("var %s = []%s{", this.AllColumnsName, this.InterfaceName)
	for _, defn := range this.Defns {
		pw.indent()
		pw.fprintLn("new(%s),", defn.TypeName)
		pw.deindent()
	}
	pw.fprintLn("}")

	for _, defn := range this.Defns {
		//--Emit type definition for each column. The value is meaningless
		pw.fprintLn("type %s int", defn.TypeName)
		//--
		pw.fprintLn("func (%s) Name() string {",
			defn.TypeName,
		)
		pw.indent()
		pw.fprintLn("return %q", defn.ColumnName)
		pw.deindent()
		pw.fprintLn("}")

		//--
		pw.fprintLn("func (%s) Index() int {",
			defn.TypeName,
		)
		pw.indent()
		pw.fprintLn("return %d", defn.Index)
		pw.deindent()
		pw.fprintLn("}")

		//---

		pw.fprintLn("func (%s) IsSet(m *%s) bool {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		pw.fprintLn("return m.IsSet.%s", defn.FieldName)
		pw.deindent()
		pw.fprintLn("}")

		//---

		pw.fprintLn("func (%s) IsLoaded(m *%s) bool {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		pw.fprintLn("return m.IsLoaded.%s", defn.FieldName)
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

		//---

		pw.fprintLn("func (%s) SetSet(m *%s, v bool) {",
			defn.TypeName,
			this.Parent.SingularModelName)
		pw.indent()
		pw.fprintLn("m.IsSet.%s = v", defn.FieldName)
		pw.deindent()
		pw.fprintLn("}")

		pw.fprintLn("")
	}

	//--Emit a slice that is the primary keys for this table
	pw.fprintLn("var %s = %s{",
		this.PrimaryKeyColumnsName,
		this.ListTypeName)
	pw.indent()
	for _, pk := range this.Parent.PrimaryKey {
		pw.fprintLn("%s,", this.ColumnTypeInstanceByFieldName(pk.Name))
	}

	pw.deindent()
	pw.fprintLn("}")

	//--Emit a receiver that returns the minimum set of identifying columns
	//for the type
	pw.fprintLn("func (this %s) identifyingColumns() (%s, error) {",
		this.Parent.SingularModelName,
		this.ListTypeName,
	)

	pw.indent()
	for _, u := range this.Parent.Unique {
		pw.fprintLn("if this.IsLoaded.%s || this.IsSet.%s {",
			u.Name,
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
				fmt.Sprintf("(this.IsLoaded.%s || this.IsSet.%s)", pk.Name, pk.Name))
		}

		pw.fprintLn("if %s {",
			strings.Join(pkLoaded, " && "))
		pw.indent()
		pw.fprintLn("return %s, nil", this.PrimaryKeyColumnsName)
		pw.deindent()
		pw.fprintLn("}")
	}

	pw.fprintLn("return nil, %s.RowNotUniquelyIdentifiableError{Instance: this}",
		sillyquil_runtime_pkg_name)
	pw.deindent()
	pw.fprintLn("}")
	return nil
}

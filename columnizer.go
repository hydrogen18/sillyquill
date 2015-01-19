package main

import "fmt"
import "github.com/spiceworks/spicelog"
import "reflect"
import "bytes"
import "time"

type CodeEmitter interface {
	Emit(*panicWriter) error
	Imports() []string
	Suffix() string
}

type ColumnizedField struct {
	DataTypeDefn []interface{}
	Name         string
	DataType     string
	SqlType      SqlDataType
	Pointer      bool
}

type ColumnizedStruct struct {
	PluralModelName   string
	SingularModelName string
	ListTypeName      string
	Fields            []ColumnizedField
	Columns           []Column
	PrimaryKey        []ColumnizedField
	Unique            []ColumnizedField
	PreferredUnique   *ColumnizedField
	UpdatedAt         *ColumnizedField
	CreatedAt         *ColumnizedField
	TableName         string

	TheColumnType *ColumnType
}

func (this *ColumnizedStruct) Suffix() string {
	return ""
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
		field.SqlType = column.DataType()

		field.DataType, err = dataTypeToString(field.DataTypeDefn)
		if err != nil {
			return nil, err
		}

		field.Pointer = column.Nullable()

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

		if column.IsCreationTimestamp() {
			this.CreatedAt = &field
		}

		if column.IsUpdateTimestamp() {
			this.UpdatedAt = &field
		}
	}

	if len(this.Unique) != 0 {
		for _, v := range this.Unique {
			if v.SqlType == SqlInt {
				this.PreferredUnique = &v
				break
			}
		}

		if this.PreferredUnique == nil {
			this.PreferredUnique = &this.Unique[0]
		}

		spicelog.Infof("Preferred unique for %q is %q",
			t.Name(),
			this.PreferredUnique.Name)
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
		reflect.Float64,
		reflect.Int16,
		reflect.Float32:
		fmt.Fprintf(buf, "%v", rk)
	case reflect.Struct:
		fmt.Fprintf(buf, "%T", dt[i])
		i++
	case reflect.Slice:
		fmt.Fprintf(buf, "[]%v", dt[i])
		i++
	default:
		return "", fmt.Errorf("Prototype not convertable %v", dt)
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
	pw.fprintLn("}") //close IsLoaded nested struct

	//--Emit a nested struct that has a boolean indicating
	//if each column is set
	pw.fprintLn("IsSet struct {")
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
	pw.fprintLn("}") //close struct
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

		pw.fprintLn("if this.IsLoaded.%s || this.IsSet.%s {",
			field.Name,
			field.Name,
		)
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

	//--Emit an accessor to load multiple fields of the struct
	pw.fprintLn("func (this *%s) Reload(db *sql.DB, columns ...%s) error {",
		this.SingularModelName,
		this.TheColumnType.InterfaceName,
	)
	pw.indent()
	pw.fprintLn("if len(columns) == 0 {")
	pw.indent()
	pw.fprintLn("columns = %s", this.TheColumnType.AllColumnsName)
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("idColumns, err := this.identifyingColumns()")
	pw.returnIf("err != nil", "err")
	pw.fprintLn("err = this.loadColumnsWhere(db,idColumns,columns...)")
	pw.returnIf("err != nil", "err")
	pw.fprintLn("%s(columns).SetLoaded(this,true)",
		this.TheColumnType.ListTypeName)
	pw.fprintLn("return nil")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit an accessor to load multiple fields of the struct
	//if not already loaded
	pw.fprintLn("func (this *%s) Get(db *sql.DB, columns ...%s) error {",
		this.SingularModelName,
		this.TheColumnType.InterfaceName,
	)
	pw.indent()
	pw.fprintLn("if len(columns) == 0 {")
	pw.indent()
	pw.fprintLn("columns = %s", this.TheColumnType.AllColumnsName)
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("var unloadedColumns []%s", this.TheColumnType.InterfaceName)
	pw.fprintLn("for _, v := range columns {")
	pw.indent()
	pw.fprintLn("if ! v.IsLoaded(this) {")
	pw.indent()
	pw.fprintLn("unloadedColumns = append(unloadedColumns,v)")
	pw.deindent()
	pw.fprintLn("}") //end if
	pw.deindent()
	pw.fprintLn("}") //end for
	pw.returnIf("len(unloadedColumns) == 0", "nil")
	pw.fprintLn("return this.Reload(db,unloadedColumns...)")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit a setter for each field of the struct
	for _, field := range this.Fields {
		pw.fprintLn("func (this *%s) Set%s(v %s) {",
			this.SingularModelName,
			field.Name,
			field.DataType)
		pw.indent()
		//Timestamps are handled as a special case.
		//Always convert the given values to UTC
		switch field.SqlType {
		case SqlTimestamp:
			if field.Pointer {
				//Do a juggling act to prevent modifying the
				//instance of time.Time pointed
				pw.fprintLn("if v != nil {")
				pw.indent()
				pw.fprintLn("utcV := new(time.Time)")
				pw.fprintLn("*utcV = v.UTC()")
				pw.fprintLn("v = utcV")
				pw.deindent()
				pw.fprintLn("}")
				pw.fprintLn("this.%s = v", field.Name)
			} else {
				pw.fprintLn("this.%s = v.UTC()", field.Name)
			}

		default:
			pw.fprintLn("this.%s = v", field.Name)

		}

		pw.fprintLn("this.IsSet.%s = true", field.Name)
		pw.deindent()
		pw.fprintLn("}")
		pw.fprintLn("")
	}

	if this.UpdatedAt != nil {
		//--Emit a function to set updated_at style column if not set
		pw.fprintLn("func (this *%s) touchUpdatedAt() {", this.SingularModelName)
		pw.indent()
		pw.fprintLn("if ! this.IsSet.%s { ", this.UpdatedAt.Name)
		pw.indent()
		pw.fprintLn("now := time.Now()")
		if this.UpdatedAt.Pointer {
			pw.fprintLn("this.Set%s(&now)", this.UpdatedAt.Name)
		} else {
			pw.fprintLn("this.Set%s(now)", this.UpdatedAt.Name)
		}
		pw.deindent()
		pw.fprintLn("}")
		pw.deindent()
		pw.fprintLn("}")
	}

	if this.CreatedAt != nil {
		//--Emit a function to set created_at style column if not set or loaded
		pw.fprintLn("func (this *%s) touchCreatedAt() {", this.SingularModelName)
		pw.indent()
		pw.fprintLn("if ! this.IsLoaded.%s && ! this.IsSet.%s { ",
			this.CreatedAt.Name,
			this.CreatedAt.Name)
		pw.indent()
		pw.fprintLn("now := time.Now()")
		if this.CreatedAt.Pointer {
			pw.fprintLn("this.Set%s(&now)", this.CreatedAt.Name)
		} else {
			pw.fprintLn("this.Set%s(now)", this.CreatedAt.Name)
		}
		pw.deindent()
		pw.fprintLn("}")
		pw.deindent()
		pw.fprintLn("}")
	}

	//--Emit a save function

	//call SetUpdatedAt(time.Now()) if not already set
	pw.fprintLn("func (this *%s) Save(db *sql.DB) error {", this.SingularModelName)
	pw.indent()
	//check if table has an "updated_at" style column and
	if this.UpdatedAt != nil {
		pw.fprintLn("this.touchUpdatedAt()")
	}
	pw.fprintLn("idColumns, err := this.identifyingColumns()")
	pw.returnIf("err != nil", "err")
	pw.fprintLn("var columnsToSave %s", this.TheColumnType.ListTypeName)
	pw.fprintLn("for _, v := range %s {", this.TheColumnType.AllColumnsName)
	pw.indent()
	pw.fprintLn("if v.IsSet(this) {")
	pw.indent()
	pw.fprintLn("columnsToSave = append(columnsToSave, v)")
	pw.deindent()
	pw.fprintLn("}") //end if
	pw.deindent()
	pw.fprintLn("}") //end for
	pw.fprintLn("err = this.updateColumnsWhere(db,idColumns,columnsToSave...)")
	pw.fprintLn("if err == nil {")
	pw.indent()
	pw.fprintLn("columnsToSave.SetLoaded(this,true)")
	pw.fprintLn("columnsToSave.SetSet(this,false)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit a create function
	pw.fprintLn("func (this *%s) Create(db *sql.DB) error {", this.SingularModelName)
	pw.indent()
	//check for "created_at" style column
	if this.CreatedAt != nil {
		pw.fprintLn("this.touchCreatedAt()")
	}
	//check for "updated_at" style column that is declared non-null
	if this.UpdatedAt != nil && !this.UpdatedAt.Pointer {
		pw.fprintLn("this.touchUpdatedAt()")
	}
	pw.fprintLn("var columnsToCreate %s", this.TheColumnType.ListTypeName)
	pw.fprintLn("for _, v := range %s {", this.TheColumnType.AllColumnsName)
	pw.indent()
	pw.fprintLn("if v.IsSet(this) {")
	pw.indent()
	pw.fprintLn("columnsToCreate = append(columnsToCreate, v)")
	pw.deindent()
	pw.fprintLn("}") //end if
	pw.deindent()
	pw.fprintLn("}") //end for

	//Check for id-style column type and append if not in the list to load
	//This make sures that the result of the Create is identifiable.
	pw.fprintLn("var columnsToLoad %s", this.TheColumnType.ListTypeName)
	if this.PreferredUnique != nil {
		instanceName := this.TheColumnType.ColumnTypeInstanceByFieldName(this.PreferredUnique.Name)
		pw.fprintLn("if ! columnsToLoad.Contains(%s) {",
			instanceName)
		pw.indent()
		pw.fprintLn("columnsToLoad = append(columnsToLoad,%s)", instanceName)
		pw.deindent()
		pw.fprintLn("}")
	}

	pw.fprintLn("err := this.insertColumns(db,columnsToLoad,columnsToCreate)")
	pw.fprintLn("if err == nil {")
	pw.indent()
	pw.fprintLn("columnsToCreate.SetLoaded(this,true)")
	pw.fprintLn("columnsToCreate.SetSet(this,false)")
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")

	//--Emit a FindOrCreate function

	pw.fprintLn("func (this *%s) FindOrCreate(db *sql.DB, columnsToLoad ...%s) error {",
		this.SingularModelName,
		this.TheColumnType.InterfaceName,
	)
	pw.indent()
	//TODO check for zero columns being set and return an error indicating such
	//check for "created_at" style column
	if this.CreatedAt != nil {
		pw.fprintLn("this.touchCreatedAt()")
	}
	//check for "updated_at" style column that is declared non-null
	if this.UpdatedAt != nil && !this.UpdatedAt.Pointer {
		pw.fprintLn("this.touchUpdatedAt()")
	}
	pw.fprintLn("idColumns, err := this.identifyingColumns()")
	pw.returnIf("err != nil", "err")

	pw.fprintLn("var columnsToSave %s", this.TheColumnType.ListTypeName)
	pw.fprintLn("for _, v := range %s {", this.TheColumnType.AllColumnsName)
	pw.indent()
	pw.fprintLn("if v.IsSet(this) {")
	pw.indent()
	pw.fprintLn("columnsToSave = append(columnsToSave, v)")
	pw.deindent()
	pw.fprintLn("}") //end if
	pw.deindent()
	pw.fprintLn("}") //end for

	pw.fprintLn("if len(columnsToLoad) == 0 {") //Load all columns if not specified
	pw.indent()
	pw.fprintLn("columnsToLoad = %s", this.TheColumnType.AllColumnsName)
	pw.deindent()
	pw.fprintLn("} else {") //Load the columns specified plus those that are set
	//This is mandatory because not all columns are unique and the find-or-create
	//paradigm could return an object that is inconsistent with respect to
	//the database if the row already exists

	pw.indent()
	pw.fprintLn("columnsToLoad = append(columnsToLoad,columnsToSave...)")
	pw.deindent()
	pw.fprintLn("}")

	pw.fprintLn("err = this.findOrCreateColumnsWhere(db,idColumns,columnsToSave,columnsToLoad)")
	pw.fprintLn("if err == nil {")
	pw.indent()
	pw.fprintLn("%s(columnsToLoad).SetLoaded(this,true)",
		this.TheColumnType.ListTypeName) //Set the columns that are loaded
	pw.fprintLn("%s(columnsToLoad).SetSet(this,false)",
		this.TheColumnType.ListTypeName) //Clear the flags for columns that are
	pw.deindent()
	pw.fprintLn("}")
	pw.fprintLn("return err")
	pw.deindent()
	pw.fprintLn("}")

	return nil
}

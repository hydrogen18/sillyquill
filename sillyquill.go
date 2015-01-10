package main

import "go/parser"
import "go/token"
import "os"
import "fmt"
import "go/ast"

type Table struct {
	Name    string
	Columns []*Column
}

type SqlType struct {
	Name string
}

var SqlInt = &SqlType{"integer"}
var SqlSmallInt = &SqlType{"smallint"}
var SqlBigInt = &SqlType{"bigint"}

var SqlReal = &SqlType{"real"}
var SqlDouble = &SqlType{"double"}

var SqlSerial = &SqlType{"serial"}

var SqlBool = &SqlType{"bool"}

var SqlString = &SqlType{"text"}

type Column struct {
	Type *SqlType
	Name string
}

func TableName(t *ast.TypeSpec) string {
	return t.Name.String()
}

func ColumnName(f *ast.Field) string {
	return f.Names[0].String()
}

func ColumnType(identity *ast.Ident) *SqlType {
	//Check for identity.Obj != nil
	switch identity.Name {
	case "uint":
		return SqlInt
	case "bool":
		return SqlBool
	case "float32":
		return SqlReal
	case "string":
		return SqlString
	}

	panic(identity)
}

type SQLizer interface {
	ToTableName(t *ast.TypeSpec) string
	ToColumnName(t *ast.TypeSpec) string
	ColumnType(identity *ast.Ident) *SqlType
}

func Analyze(filename string, sqlizer SQLizer) ([]*Table, error) {
	var tables []*Table
	return tables, nil
}

func main() {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, os.Args[1], nil, 0)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", parsed)

	var tables []*Table

	for i, d := range parsed.Decls {
		fmt.Printf("Decls[%d](%T) = %v\n", i, d, d)

		switch d := d.(type) {
		case *ast.GenDecl:
			fmt.Printf("\t%v\n", d.Tok)
			for j, s := range d.Specs {

				s := s.(*ast.TypeSpec)
				fmt.Printf("\tSpecs[%d] = %v\n", j, s)
				fmt.Printf("\t\t%v\n", s.Name)

				table := &Table{Name: TableName(s)}

				fmt.Printf("\t\t(%T)%v\n", s.Type, s.Type)
				st := s.Type.(*ast.StructType)
				for k, f := range st.Fields.List {
					column := &Column{Name: ColumnName(f)}
					fmt.Printf("\t\t\tFields[%d](%T) = %v\n", k, f, f)
					typeId := f.Type.(*ast.Ident)
					fmt.Printf("\t\t\tFields[%d].Type(%T) = %v\n", k, typeId, typeId.Name)
					fmt.Printf("\t\t\tFields[%d].Type.Obj = %v\n", k, typeId.Obj)

					if f.Tag != nil {
						fmt.Printf("\t\t\tFields[%d].Tag = %v\n", k, f.Tag)
						continue //TODO check tag contents
					}

					column.Type = ColumnType(typeId)

					table.Columns = append(table.Columns, column)
				}
				tables = append(tables, table)
			}
		}
	}

	for i, table := range tables {
		fmt.Printf("table[%d] = %v\n", i, table.Name)
		for k, column := range table.Columns {
			fmt.Printf("\tcolumn[%d](%v) = %v\n", k, column.Type.Name, column.Name)
		}
	}

}

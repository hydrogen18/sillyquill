package main

import "flag"
import "github.com/spiceworks/spicelog"
import _ "github.com/lib/pq"
import "database/sql"
import "fmt"
import "strings"
import "os"

type Table interface {
	Name() string
	Columns() ([]Column, error)
}

type SqlDataType int

const sqlUnknown = SqlDataType(0)
const SqlInt = SqlDataType(1)
const SqlBigInt = SqlDataType(2)
const SqlByteArray = SqlDataType(3)
const SqlVarChar = SqlDataType(4)
const SqlBoolean = SqlDataType(5)
const SqlTimestamp = SqlDataType(6)

type ErrNoSuchDataType string

func (this ErrNoSuchDataType) Error() string {
	return fmt.Sprintf("No match for datatype %q", string(this))
}

func StringToSqlDataType(v string) (SqlDataType, error) {

	dataType := strings.Trim(strings.ToUpper(v), " ")

	switch dataType {
	case "INT", "INTEGER":
		return SqlInt, nil
	case "BIGINT":
		return SqlBigInt, nil
	case "BOOLEAN":
		return SqlBoolean, nil
	case "CHARACTER VARYING":
		return SqlVarChar, nil
	case "BYTEA":
		return SqlByteArray, nil

	}

	if strings.HasPrefix(dataType, "TIMESTAMP") {
		return SqlTimestamp, nil
	}

	return sqlUnknown, ErrNoSuchDataType(v)
}

type Column interface {
	DataType() SqlDataType
	Name() string
	Nullable() bool
}

type InformationSchemaAdapter struct {
	TableSchema string
	db          *sql.DB
}

type InformationSchemaColumn struct {
	name     string
	dataType SqlDataType
	nullable bool
}

func (this *InformationSchemaColumn) Name() string {
	return this.name
}

func (this *InformationSchemaColumn) DataType() SqlDataType {
	return this.dataType
}

func (this *InformationSchemaColumn) Nullable() bool {
	return this.nullable
}

type InformationSchemaTable struct {
	name   string
	parent *InformationSchemaAdapter
}

func (this *InformationSchemaTable) Name() string {
	return this.name
}

func (this *InformationSchemaTable) Columns() ([]Column, error) {
	const query = `Select column_name,data_type,is_nullable from 
	information_schema.columns  
	where 
		table_name = $1
	and 
		table_schema = $2`

	rows, err := this.parent.db.Query(query, this.name, this.parent.TableSchema)
	if err != nil {
		return nil, err
	}

	var result []Column

	for rows.Next() {
		var column_name string
		var data_type string
		var is_nullable string
		err := rows.Scan(&column_name, &data_type, &is_nullable)
		if err != nil {
			return nil, err
		}

		sdt, err := StringToSqlDataType(data_type)
		if err != nil {
			return nil, err
		}

		var isNullable bool
		switch is_nullable {
		case "NO":
			isNullable = false
		case "YES":
			isNullable = true
		default:
			return nil, fmt.Errorf("Bad value for is_nullable table %q column %q:%v",
				this.name,
				column_name,
				is_nullable)
		}

		result = append(result,
			&InformationSchemaColumn{
				nullable: isNullable,
				dataType: sdt,
				name:     column_name,
			})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return result, nil
}

func (this *InformationSchemaAdapter) Tables() ([]Table, error) {
	const query = "Select table_name from information_schema.tables where table_schema=$1"

	rows, err := this.db.Query(query, this.TableSchema)
	if err != nil {
		return nil, err
	}
	var results []Table
	for rows.Next() {
		var table_name string
		err = rows.Scan(&table_name)
		if err != nil {
			return nil, err
		}

		results = append(results,
			&InformationSchemaTable{
				name:   table_name,
				parent: this,
			})
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func main() {
	flag.Parse()
	const dbconfig = "dbname=ericu sslmode=disable"

	db, err := sql.Open("postgres", dbconfig)

	if err != nil {
		spicelog.Fatalf("%v", err)
	}
	defer db.Close()

	adapter := &InformationSchemaAdapter{
		db:          db,
		TableSchema: "public",
	}

	tables, err := adapter.Tables()
	if err != nil {
		spicelog.Fatalf("%v", err)
	}

	for _, table := range tables {
		var err error

		//spicelog.Infof("Table %q", table.Name())
		me := NewModelEmitter()
		me.Package = "dal"

		if table.Name() == "examples" {
			err = me.Emit(table, os.Stdout)
			if err != nil {
				spicelog.Fatalf("err:%v", err)
			}
		}
		os.Stdout.Sync()

	}

	spicelog.Stop()
}

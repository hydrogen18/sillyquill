package main

import "flag"
import "github.com/spiceworks/spicelog"
import _ "github.com/lib/pq"
import "database/sql"
import "fmt"
import "strings"
import "sync"

type Table interface {
	Name() string
	Columns() ([]Column, error)
	Unique() ([]string, error)
	PrimaryKey() ([]string, error)
}

type SqlDataType int

const sqlUnknown = SqlDataType(0)
const SqlInt = SqlDataType(1)
const SqlBigInt = SqlDataType(2)
const SqlByteArray = SqlDataType(3)
const SqlVarChar = SqlDataType(4)
const SqlBoolean = SqlDataType(5)
const SqlTimestamp = SqlDataType(6)
const SqlFloat64 = SqlDataType(7)
const SqlText = SqlDataType(8)
const SqlNumeric = SqlDataType(9)
const SqlDate = SqlDataType(10)

type ErrNoSuchDataType struct {
	SqlTypeName string
	TableName   string
	ColumnName  string
}

func (this ErrNoSuchDataType) Error() string {
	return fmt.Sprintf("No such datatype %q for column %q of table %q",
		this.SqlTypeName,
		this.ColumnName,
		this.TableName,
	)
}

func (this *InformationSchemaColumn) StringToSqlDataType(v string) (SqlDataType, error) {

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
	case "TEXT":
		return SqlText, nil
	case "BYTEA":
		return SqlByteArray, nil
	case "DOUBLE PRECISION":
		return SqlFloat64, nil
	case "DATE":
		return SqlDate, nil
	case "NUMERIC": //TODO handle variable precision on type
		return SqlNumeric, nil
	case "TIMESTAMP WITHOUT TIME ZONE", "TIMESTAMP":
		//TIMESTAMP - timestamp without timezone
		//TIMESTAMP WITHOUT TIME ZONE - timestamp without time zone
		//TIMESTAMP WITH TIME ZONE - timestamp with time zone - unsupported
		//TIMESTAMPTZ - timestamp with time zone - unsupported

		return SqlTimestamp, nil
	}

	return sqlUnknown, ErrNoSuchDataType{
		ColumnName:  this.name,
		TableName:   this.parent.name,
		SqlTypeName: v,
	}
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
	parent   *InformationSchemaTable
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

func (this *InformationSchemaTable) PrimaryKey() ([]string, error) {
	return this.columnNamesWhereConstraintType("PRIMARY KEY")
}

func (this *InformationSchemaTable) Unique() ([]string, error) {
	return this.columnNamesWhereConstraintType("UNIQUE")
}

func (this *InformationSchemaTable) columnNamesWhereConstraintType(constraint_type string) ([]string, error) {
	const query = `Select 
	column_name 
	from 
		information_schema.table_constraints
	left join
		information_schema.constraint_column_usage	
	on
		information_schema.constraint_column_usage.constraint_name
	= 
		information_schema.table_constraints.constraint_name
	
	where 
		information_schema.table_constraints.table_schema = $1
	and 
		information_schema.table_constraints.table_name = $2
	and 
		constraint_type = $3`

	var uniques []string
	var err error

	rows, err := this.parent.db.Query(query, this.parent.TableSchema,
		this.name,
		constraint_type)
	if err != nil {
		return nil, err
	}
	for rows.Next() {

		var u string
		err = rows.Scan(&u)
		if err != nil {
			return nil, err
		}
		uniques = append(uniques, u)

	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return uniques, nil

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

		col := &InformationSchemaColumn{}
		col.parent = this
		col.name = column_name

		col.dataType, err = col.StringToSqlDataType(data_type)
		if err != nil {
			return nil, err
		}

		switch is_nullable {
		case "NO":
			col.nullable = false
		case "YES":
			col.nullable = true
		default:
			return nil, fmt.Errorf("Bad value for is_nullable table %q column %q:%v",
				this.name,
				column_name,
				is_nullable)
		}

		result = append(result,
			col)
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
	//const dbconfig = "user=skypal password=ismypal dbname=ccm_development sslmode=disable host=10.192.4.57"
	const dbconfig = "dbname=ericu sslmode=disable host=localhost"
	db, err := sql.Open("postgres", dbconfig)
	db.SetMaxOpenConns(16)
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

	exclude_tables := make(map[string]int)
	exclude_tables["how_to_steps"] = 0
	exclude_tables["meetings"] = 0
	exclude_tables["messaging_inbox_entries"] = 0
	exclude_tables["messaging_topics"] = 0
	exclude_tables["messaging_messages"] = 0
	exclude_tables["spice_list_items"] = 0
	exclude_tables["spice_lists"] = 0
	exclude_tables["topics"] = 0
	exclude_tables["vendor_pages"] = 0
	exclude_tables["meeting_comments"] = 0
	exclude_tables["posts"] = 0
	exclude_tables["vendors"] = 0
	exclude_tables["products"] = 0
	exclude_tables["how_to_articles"] = 0

	wg := new(sync.WaitGroup)
	for _, table := range tables {

		spicelog.Infof("Table %q", table.Name())
		if _, ok := exclude_tables[table.Name()]; ok {
			continue
		}

		wg.Add(1)
		go func(t Table) {
			defer wg.Done()

			me := NewModelEmitter()
			me.Package = "dal"

			err = me.Emit(t, "/home/ericu/sillyquill/src/dummy/dal/")
			if err != nil {
				spicelog.Errorf("Error:%v", err)
			}

		}(table)

	}
	wg.Wait()

	spicelog.Stop()
}

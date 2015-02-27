#Purpose of Sillyquill
---
The purpose of the sillyquill package is to generate Golang source code to interface to an existnig Postgres schema. 

##Design philosophy
---
The design of this is influenced heavily by ActiveRecord from the Ruby on Rails project. My design goals are the following

1. Don't be all things to all people
1. Don't sacrifice existing functionality of the `database/sql` package
1. Bring relational database concepts to Golang, don't bring object-oriented concepts to a database.

##Configuration file format
---

The configuration file format is TOML and the basic format is very simple.

At the top level there are three keys

* `db` - The full connection string that is passed to the `github.com/lib/pq` package
* `output-dir` - The full path on the filesystem that files are generated to
* `package` - The package name of generated source files
* `connection-max` - The maximum number of connections to open

##Table mapping
---
The name of the table is used to determine the name of the generated structure. The table name is expected to be plural with underscores. For example a table named `products` becomes a structure named `Product`. Likewise `vendor_invoices` becomes `VendorInvoice`.

This functionality needs to be abstracted out into something that is both pluggable and manually overrideable. For example a table name like `geese` does not presently become `Goose`. By building some sort of abstraction around this, users would be able to adapt the software to  any naming convention.

##Column mapping to types
---
SQL standardizes a large number of types. To generate a `struct` to model rows in a table you must decide on a consistent mapping between SQL's types and types in Golang. These are the obvious mappings.

* `SMALLINT` - `int16`
* `INT` - `int32`
* `BIGINT` - `int64`
* `VARCHAR` - `string`
* `BYTEA` - `[]byte`
* `TIMESTAMP` - `time.Time`
* `REAL` - `float32`
* `DOUBLE PRECISION` - `float64`

There are other types that have no real answer. The `NUMERIC` type does not map to any sort of builtin type in Go. I settled on mapping it to 

To further complicate matters, PostgreSQL implements types like `HSTORE` and `TSVECTOR`. For now, `TSVECTOR` is simply ignored if encountered.

An important note is that all `TIMESTAMP` columns are assumed to be stored in UTC.
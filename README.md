#Sillyquill
---

Sillyquill is a tool for generating Golang code from an exising Postgres database.

The generated code models each row as an instance of a `struct`. The API supports basic CRUD semantics, but can also handle rows from your own SQL queries. The generated code is built on top of the project [github.com/lib/pq][https://github.com/lib/pq].

##Example usage of generated code
---

Creating a new row in the `users` table.

```

u := new(dal.User)
u.SetName("foobar")
u.SetEmail("foo@bar.com")
err := u.Save(dbconn)
//Check the value of err

```

Lookup the same user in the `users` table by the `name` column.

```

u := new(dal.User)
u.SetName("foobar")
err := u.Get(dbconn)
//Check the value of err

```


##Quick Start
---
Create a file called `sillyquill.toml`. Place it in the following

```
db= "<full connection string passed to github.com/lib/pq>"
output-dir= "/the/full/path/to/generate/to"
package= "<name of package to generate>"
```

Run the following commands

```
go get github.com/hydrogen18/sillyquill
go install github.com/hydrogen18/sillyquill
$GOPATH/bin/sillyquill #In the directory with sillyquill.toml
```

Sillyquill now runs and generates source files in the configured directory.

###TODO
---
Foreign key support
Schema support
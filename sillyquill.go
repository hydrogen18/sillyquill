package main

import "flag"
import "github.com/spiceworks/spicelog"
import _ "github.com/lib/pq"
import "database/sql"
import "sync"

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

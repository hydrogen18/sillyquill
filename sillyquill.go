package main

import "flag"
import "github.com/spiceworks/spicelog"
import _ "github.com/lib/pq"
import "database/sql"
import "sync"
import "github.com/BurntSushi/toml"

type table struct {
	Exclude bool `toml:"exclude"`
}

type config struct {
	DB            string `toml:"db"`
	Schema        string `toml:"schema"`
	OutputDir     string `toml:"output-dir"`
	Package       string `toml:"package"`
	ConnectionMax int    `toml:"connection-max"`

	Tables map[string]table `oml:"tables"`
}

func main() {
	tomlFile := flag.String("conf", "sillyquill.toml", "TOML configuration file path")
	flag.Parse()

	var conf config
	_, err := toml.DecodeFile(*tomlFile, &conf)
	if err != nil {
		spicelog.Fatalf("Failed parsing file %q:%v", *tomlFile, err)
	}

	db, err := sql.Open("postgres", conf.DB)
	if err != nil {
		spicelog.Fatalf("Failed opening database:%v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(conf.ConnectionMax)

	adapter := &InformationSchemaAdapter{
		db:          db,
		TableSchema: conf.Schema,
	}

	tables, err := adapter.Tables()
	if err != nil {
		spicelog.Fatalf("Failed querying for tables:%v", err)
	}

	wg := new(sync.WaitGroup)
	for _, table := range tables {
		spicelog.Infof("Processing table %q", table.Name())

		tableConf, ok := conf.Tables[table.Name()]
		if ok {
			if tableConf.Exclude {
				spicelog.Infof("Skipping table %q", table.Name())
				continue
			}
		}

		wg.Add(1)
		go func(t Table) {
			defer wg.Done()

			me := NewModelEmitter()
			me.Package = conf.Package

			err = me.Emit(t, conf.OutputDir)
			if err != nil {
				spicelog.Errorf("Error processing table %q:%v", table.Name(), err)
			}

		}(table)
	}
	wg.Wait()

	spicelog.Stop()
}

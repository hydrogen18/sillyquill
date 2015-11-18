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
	DB            string           `toml:"db"`
	Schema        string           `toml:"schema"`
	OutputDir     string           `toml:"output-dir"`
	Package       string           `toml:"package"`
	ConnectionMax int              `toml:"connection-max"`
	Tables        map[string]table `toml:"tables"`
	TableMode   string `toml:"table-mode"`
}

func main() {
	tomlFile := flag.String("conf", "sillyquill.toml", "TOML configuration file path")
	flag.Parse()
	defer spicelog.Stop()

	var conf config
	_, err := toml.DecodeFile(*tomlFile, &conf)
	if err != nil {
		spicelog.Fatalf("Failed parsing file %q:%v", *tomlFile, err)
	}
	spicelog.Infof("Parsed file %q", *tomlFile)

	if conf.ConnectionMax <= 0 {
		conf.ConnectionMax = 1
	}

	if conf.Schema == "" {
		conf.Schema = "public"
	}
	
	if conf.TableMode == "" {
	    conf.TableMode = "normal"
	}
	
	var explicit bool
	
	if conf.TableMode == "explicit" {
		explicit = true
		spicelog.Infof("The selected table mode is explicit. Only generating models for the tables listed in the configuration file.")
	}

	db, err := sql.Open("postgres", conf.DB)
	if err != nil {
		spicelog.Fatalf("Failed opening database:%v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(conf.ConnectionMax)

	err = db.Ping()
	if err != nil {
		spicelog.Fatalf("Database unreachable:%v", err)
	}

	adapter := &InformationSchemaAdapter{
		db:          db,
		TableSchema: conf.Schema,
	}
	spicelog.Infof("Querying schema %q", conf.Schema)
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
		} else {
			if explicit {
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
				spicelog.Errorf("Error processing table %q:%v", t.Name(), err)
			} else {
        spicelog.Infof("Emitted model for table %q", t.Name())
      }

		}(table)
	}
	wg.Wait()

}

package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type database struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

type query struct {
	Name   string `json:"name"`
	Q      string `json:"sql"`
	Params string `json:"params"`
	Out    string `json:"out"`
}

type update struct {
	Name   string `json:"name"`
	U      string `json:"sql"`
	Params string `json:"params"`
}

type config struct {
	Database *database `json:"database"`
	Queries  []*query  `json:"queries"`
	Updates  []*update `json:"updates"`
}

func readConfig() (*config, error) {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		return nil, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getDB(cfg *config) (*sql.DB, error) {
	db, err := sql.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func execUpdate(db *sql.DB, u *update, flags *flag.FlagSet) error {
	args := make([]interface{}, 0, flags.NFlag())
	flags.Visit(func(fn *flag.Flag) {
		t, ok := fn.Value.(flag.Getter)
		if ok {
			args = append(args, t.Get())
		} else {
			log.Println("Value for", fn.Name, "is not a getter")
		}
	})

	_, err := db.Exec(u.U, args...)
	if err != nil {
		return err
	}
	return nil
}

func execQuery(db *sql.DB, q *query, flags *flag.FlagSet) error {
	tmpl, err := template.New("q").Parse(q.Out)
	if err != nil {
		return err
	}

	args := make([]interface{}, 0, flags.NFlag())
	flags.Visit(func(fn *flag.Flag) {
		t, ok := fn.Value.(flag.Getter)
		if ok {
			args = append(args, t.Get())
		} else {
			log.Println("Value for", fn.Name, "is not a getter")
		}
	})

	rows, err := db.Query(q.Q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	names, err := rows.Columns()
	if err != nil {
		return err
	}

	vals := make([]interface{}, 0, len(names))
	for i := 0; i < len(names); i++ {
		vals = append(vals, new(string))
	}

	for rows.Next() {
		if err := rows.Scan(vals...); err != nil {
			return err
		}
		data := make(map[string]interface{})
		for i := 0; i < len(names); i++ {
			data[names[i]] = vals[i]
		}
		tmpl.Execute(os.Stdout, data)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

var cfgFile = flag.String("cfg", "./config.json", "config file")

func flagDriver() {
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("toyq: ")
	flag.Parse()

	cfg, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := getDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	opname := flag.Arg(0)
	if opname == "" {
		log.Fatal("no op name")
	}

	var query *query
	var update *update

	for _, q := range cfg.Queries {
		if q.Name == opname {
			query = q
			break
		}
	}
	for _, u := range cfg.Updates {
		if u.Name == opname {
			update = u
			break
		}
	}
	if query == nil && update == nil {
		log.Fatal("no operation with name ", opname)
	}

	if query != nil {
		flags := flag.NewFlagSet("args", flag.ExitOnError)
		for _, param := range strings.Fields(query.Params) {
			flags.String(param, "", "set query param")
		}
		flags.Parse(flag.Args()[1:])

		if err := execQuery(db, query, flags); err != nil {
			log.Fatal(err)
		}
	} else if update != nil {
		flags := flag.NewFlagSet("args", flag.ExitOnError)
		for _, param := range strings.Fields(update.Params) {
			flags.String(param, "", "set update param")
		}
		flags.Parse(flag.Args()[1:])

		if err := execUpdate(db, update, flags); err != nil {
			log.Fatal(err)
		}
	}
}

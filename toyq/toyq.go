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

type config struct {
	Database *database `json:"database"`
	Queries  []*query  `json:"queries"`
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

func exec(db *sql.DB, q *query, flags *flag.FlagSet) error {
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

	qname := flag.Arg(0)
	if qname == "" {
		log.Fatal("no query name")
	}
	var query *query
	for _, q := range cfg.Queries {
		if q.Name == qname {
			query = q
			break
		}
	}
	if query == nil {
		log.Fatal("no query with name ", qname)
	}

	flags := flag.NewFlagSet("args", flag.ExitOnError)
	for _, param := range strings.Fields(query.Params) {
		flags.String(param, "", "set query param")
	}
	flags.Parse(flag.Args()[1:])

	if err := exec(db, query, flags); err != nil {
		log.Fatal(err)
	}
}

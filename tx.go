package main

/*
Implements a key-value REST web service in go (http://golang.org)

It uses
  github.com/bmizerany/pat A sinatra style pattern muxer
  github.com/bmizerany/pq  A pure Go postgres driver

The urls are
POST /store/:key put a new key-value pair
PUT  /store/:key update an existing key-value pair
GET  /store/:key get the value of key

Tested with go1.0.2 and PostgreSQL 9.1 on ubuntu 11.10 (64-bit)

Usage:
create the kv_store table using psql or pgadmin

CREATE TABLE kv_store (
	kv_key character varying primary key,
	kv_val bytea[]
);

and then build the go program with 'go build tx.go' and run it
*/

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/bmizerany/pq"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

var dbUrl = flag.String("url", "postgres://postgres:postgres@localhost:5432/postgres", "postgres connection url")
var host = flag.String("host", "localhost", "web server host")
var port = flag.String("port", "8080", "web server port")

// use this script to create the table for the key-value store
/*

CREATE TABLE kv_store (
	kv_key character varying primary key,
	kv_val bytea[]
);

*/

var ErrKeyAlreadyExists = errors.New("key already exists")
var ErrKeyDoesNotExists = errors.New("key does not exists")

var db *sql.DB

// database.sql does not cache prepared statements per connection
// but the API Tx.Stmt can do it underneath. So instead of doing
// Tx.Exec and Tx.Query we prepare all the Stmts once and use them
// with Tx.Stmt. This is currently inefficient but seems the proper
// way to use the API. The performance will be fixed when
// the implementation of database.sql introduces Stmt caching.
// check the source of database.sql for more comments
var stmts map[string]*sql.Stmt = make(map[string]*sql.Stmt)


// database.sql prepares the query in one, random connection
// of its internal pool but as the queries are multiplexed
// in the whole pool, eventually it will be re-prepared
// in all of them
func prepareStmtOrExit(name, q string) {
	stmt, err := db.Prepare(q)
	if err != nil {
		log.Fatal("error: db.Prepare: ", q, err)
	} else {
		stmts[name] = stmt
	}
}


// get the transaction-specific prepared statement f
func getStmt(tx *sql.Tx, name string) *sql.Stmt {
	return tx.Stmt(stmts[name])
}

// run the function fn inside a db transaction. If fn
// returns an error then the transaction is rollbacked
// else it is commited. runInTransaction returns the
// result of fn
func runInTransaction(fn func(tx *sql.Tx) error) (err error) {
	var tx *sql.Tx
	defer func() {
		if e := recover(); e != nil && tx != nil {
			if e1 := tx.Rollback(); e1 != nil {
				err = e1
			} else {
				err = e.(error)
			}
		}
	}()

	if tx, err = db.Begin(); err == nil {
		err = fn(tx)
		if err == nil {
			err = tx.Commit()
		} else if e := tx.Rollback(); e != nil {
			err = e
		}
	}
	return err
}

// check if a key is already stored in the db
func isKeyInDB(tx *sql.Tx, key string) bool {
	row := getStmt(tx, "isKeyInDB").QueryRow(key)
	var n int
	_ = row.Scan(&n)
	return n == 1
}

// insert a new key-value pair
// prerequisite: the key is not already used
func insertNewKeyValue(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get(":key")
	val, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Print("error: ioutil.ReadAll: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = runInTransaction(func(tx *sql.Tx) error {
		if isKeyInDB(tx, key) {
			return ErrKeyAlreadyExists
		}
		_, err = getStmt(tx, "insertNewKeyValue").Exec(key, val)
		return err
	})
	if err == ErrKeyAlreadyExists {
		http.Error(w, fmt.Sprintf("key %q already exists", key), 400)
	} else if err != nil {
		log.Print("error: insertNewKeyValue: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else {
		http.Error(w, "", http.StatusCreated)
	}
}

// update a key-value pair
// prerequisite: the key is in use
func updateExistingKeyValue(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get(":key")
	val, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Print("error: ioutil.ReadAll: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = runInTransaction(func(tx *sql.Tx) error {
		if !isKeyInDB(tx, key) {
			return ErrKeyDoesNotExists
		}
		_, err = getStmt(tx, "updateExistingKeyValue").Exec(key, val)
		return err
	})
	if err == ErrKeyDoesNotExists {
		http.Error(w, fmt.Sprintf("key %q does not exists", key), http.StatusNotFound)
	} else if err != nil {
		log.Print("error: insertNewKeyValue: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else {
		http.Error(w, "", http.StatusOK)
	}
}

// get the value of a key
func retrieveKeyValue(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get(":key")

	var val []byte
	err := runInTransaction(func(tx *sql.Tx) error {
		row := getStmt(tx, "retrieveKeyValue").QueryRow(key)
		return row.Scan(&val)
	})
	if err == sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("key %q does not exists", key), http.StatusNotFound)
	} else if err != nil {
		log.Print("error: retrieveKeyValue: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(val)
	}
}

func main() {
	flag.Parse()

	dsn, err := pq.ParseURL(*dbUrl)
	if err != nil {
		log.Fatal("pq.ParseURL: ", err)
	}

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("error: sql.Open: ", err)
	}

	// initially all the following stmts are prepared in one
	// connection, maybe different for each. The sql package
	// will re-prepare them in every subsequent connection they are used
	prepareStmtOrExit("isKeyInDB", "select count(*)  from kv_store where kv_key = $1")
	prepareStmtOrExit("insertNewKeyValue", "insert into kv_store values($1, $2)")
	prepareStmtOrExit("updateExistingKeyValue", "update kv_store set kv_val = $2 where kv_key = $1")
	prepareStmtOrExit("retrieveKeyValue", "select kv_val from kv_store where kv_key = $1")

	m := pat.New()
	m.Post("/store/:key", http.HandlerFunc(insertNewKeyValue))
	m.Put("/store/:key", http.HandlerFunc(updateExistingKeyValue))
	m.Get("/store/:key", http.HandlerFunc(retrieveKeyValue))

	http.Handle("/", m)
	err = http.ListenAndServe(net.JoinHostPort(*host, *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
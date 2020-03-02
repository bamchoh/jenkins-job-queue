package main

import (
	"flag"
	"fmt"
	"net/http"

	bolt "go.etcd.io/bbolt"

	"test_httpserver/job"
)

var db *bolt.DB
var rootName []byte

func initDB(dbfile string) (err error) {
	db, err = bolt.Open(dbfile, 0666, nil)
	if err != nil {
		err = fmt.Errorf("open DB error: %s", err)
		fmt.Println(err)
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(rootName)
		if err != nil {
			err = fmt.Errorf("create bucket: %s", err)
			fmt.Println(err)
			return err
		}
		return err
	})
	return err
}

func main() {
	dbname := flag.String("db", "", "database filename")
	addr := flag.String("addr", ":20000", "server address")
	bucket := flag.String("bucket", "MyBucket", "root bucket name")

	flag.Parse()

	rootName = []byte(*bucket)

	var err error
	err = initDB(*dbname)
	if err != nil {
		err = fmt.Errorf("initDB error: %s", err)
		fmt.Println(err)
		return
	}
	defer db.Close()

	go job.Execute(db, rootName)

	fmt.Println("server start")
	http.HandleFunc("/job", handler)
	http.ListenAndServe(*addr, nil)
}

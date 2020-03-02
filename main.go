package main

import (
	"flag"
	"fmt"
	"net/http"

	bolt "go.etcd.io/bbolt"

	"test_httpserver/job"
)

func initDB(dbfile string, rootName []byte) (db *bolt.DB, err error) {
	db, err = bolt.Open(dbfile, 0666, nil)
	if err != nil {
		err = fmt.Errorf("open DB error: %s", err)
		fmt.Println(err)
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(rootName)
		if err != nil {
			err = fmt.Errorf("create bucket: %s", err)
			fmt.Println(err)
			return nil, err
		}
		return nil, err
	})
	return db, err
}

func main() {
	dbname := flag.String("db", "", "database filename")
	addr := flag.String("addr", ":20000", "server address")
	bucket := flag.String("bucket", "MyBucket", "root bucket name")

	flag.Parse()

	rootName := []byte(*bucket)

	var err error
	db, err = initDB(*dbname, rootName)
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

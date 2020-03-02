package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	bolt "go.etcd.io/bbolt"

	"test_httpserver/job"
)

var db *bolt.DB
var bucketName = []byte("MyBucket")

func initDB(dbfile string) (err error) {
	db, err = bolt.Open(dbfile, 0666, nil)
	if err != nil {
		err = fmt.Errorf("open DB error: %s", err)
		fmt.Println(err)
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			err = fmt.Errorf("create bucket: %s", err)
			fmt.Println(err)
			return err
		}
		return err
	})
	return err
}

func handlerJob(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		job.Update(db, bucketName, w, r)
	case http.MethodGet:
		fmt.Println("Get Method")
		err := db.View(func(tx *bolt.Tx) error {
			fmt.Println("DB View")
			bucket := tx.Bucket(bucketName)
			if bucket != nil {
				fmt.Println("start")
				bucket.ForEach(func(k, v []byte) error {
					fmt.Fprintf(w, "[%s] = \n", string(k))
					idBucket := bucket.Bucket(k)
					if idBucket != nil {
						idBucket.ForEach(func(kk, vv []byte) error {
							fmt.Fprintf(w, "  %s = %s\n", string(kk), string(vv))
							return nil
						})
					}
					return nil
				})
				fmt.Println("end")
				return nil
			}
			fmt.Fprint(w, "No item")
			return nil
		})
		if err != nil {
			fmt.Fprint(w, "Error happened:", err)
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func execJob() {
	for {
		user := ""
		buildURL := ""
		observeURL := ""
		err := db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(bucketName)
			if bucket != nil {
				cursor := bucket.Cursor()
				k, _ := cursor.First()
				if len(k) > 0 {
					idBucket := bucket.Bucket(k)
					err := bucket.DeleteBucket(k)
					if err != nil {
						return err
					}
					buildURL = string(idBucket.Get([]byte("buildURL")))
					observeURL = string(idBucket.Get([]byte("observeURL")))
					user = string(idBucket.Get([]byte("user")))
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println("execJob Error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if user == "" || buildURL == "" || observeURL == "" {
			time.Sleep(5 * time.Second)
			continue
		}

		cmd := exec.Command("curl", "--user", user, buildURL)
		fmt.Println(cmd)
		err = cmd.Run()
		if err != nil {
			fmt.Println("command execution error: ", err)
		}
		fmt.Println("end - cmd")

		err = waitJobComplete(user, observeURL)
		if err != nil {
			fmt.Println("wait job complete error: ", err)
		}
	}
}

func main() {
	var err error
	err = initDB("./test.db")
	if err != nil {
		err = fmt.Errorf("initDB error: %s", err)
		fmt.Println(err)
		return
	}
	defer db.Close()

	go execJob()

	fmt.Println("server start")
	http.HandleFunc("/job", handlerJob)
	http.ListenAndServe(":20000", nil)
}

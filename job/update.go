package job

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Job struct {
	Title      string `json:"title"`
	BuildURL   string `json:"buildURL"`
	ObserveURL string `json:"observeURL"`
	User       string `json:"user"`
}

func decodeJSONFromRequest(r *http.Request, jsonData *Job) (err error) {
	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		return fmt.Errorf("Get Content Length Error: %s", err)
	}

	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		return fmt.Errorf("Request Read Body Error: %s", err)
	}

	if err := json.Unmarshal(body, &jsonData); err != nil {
		return fmt.Errorf("Json Unmarshal Error: %s", err)
	}
	return nil
}

func fetchIDBucket(bucket *bolt.Bucket, jobName string) (*bolt.Bucket, error) {
	var idBucket *bolt.Bucket
	err := bucket.ForEach(func(key, val []byte) error {
		tmpBucket := bucket.Bucket(key)
		title := tmpBucket.Get([]byte("title"))
		if string(title) == jobName {
			idBucket = tmpBucket
		}
		return nil
	})
	return idBucket, err
}

func genID(now time.Time, job string) []byte {
	return []byte(now.Format("2006-01-02 15-04-05 ") + job)
}

func Update(db *bolt.DB, rootName []byte, w http.ResponseWriter, r *http.Request) error {
	var jsonData Job
	err := decodeJSONFromRequest(r, &jsonData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("Decode Json Error: %s", err)
	}

	return db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(rootName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		fmt.Println("jsonData:", jsonData)

		idBucket, err := fetchIDBucket(bucket, jsonData.Title)
		if err != nil {
			return fmt.Errorf("delete bucket error: %s", err)
		}

		fmt.Println("idBucket:", idBucket)

		var id []byte
		if idBucket == nil {
			id = genID(time.Now(), jsonData.Title)
			fmt.Printf("id:%s, title:%s\n", string(id), jsonData.Title)
			idBucket, err = bucket.CreateBucketIfNotExists(id)
			if err != nil {
				return fmt.Errorf("create id(%d) bucket: %s", id, err)
			}
		}

		err = idBucket.Put([]byte("add_time"), []byte(time.Now().Format("2006-01-02 15:04:05")))
		if err != nil {
			return fmt.Errorf("Put 'title' error: %s", err)
		}
		err = idBucket.Put([]byte("title"), []byte(jsonData.Title))
		if err != nil {
			return fmt.Errorf("Put 'title' error: %s", err)
		}
		err = idBucket.Put([]byte("buildURL"), []byte(jsonData.BuildURL))
		if err != nil {
			return fmt.Errorf("Put 'buildURL' error: %s", err)
		}
		err = idBucket.Put([]byte("observeURL"), []byte(jsonData.ObserveURL))
		if err != nil {
			return fmt.Errorf("Put 'observeURL' error: %s", err)
		}
		err = idBucket.Put([]byte("user"), []byte(jsonData.User))
		if err != nil {
			return fmt.Errorf("Put 'user' error: %s", err)
		}
		return nil
	})
}

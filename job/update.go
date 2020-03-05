package job

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	dproxy "github.com/koron/go-dproxy"
	bolt "go.etcd.io/bbolt"
)

func decodeJSONFromRequest(r *http.Request) (p dproxy.Proxy, err error) {
	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		return nil, fmt.Errorf("Get Content Length Error: %s", err)
	}

	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("Request Read Body Error: %s", err)
	}

	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("Json Unmarshal Error: %s", err)
	}
	return dproxy.New(v), nil
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
	p, err := decodeJSONFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("Decode Json Error: %s", err)
	}

	return db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(rootName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}

		title, err := p.M("title").String()
		fmt.Println("title:", title)
		if err != nil {
			return fmt.Errorf("Getting title: %s", err)
		}
		idBucket, err := fetchIDBucket(bucket, title)
		if err != nil {
			return fmt.Errorf("delete bucket error: %s", err)
		}

		var id []byte
		if idBucket == nil {
			id = genID(time.Now(), title)
			idBucket, err = bucket.CreateBucketIfNotExists(id)
			if err != nil {
				return fmt.Errorf("create id(%d) bucket: %s", id, err)
			}
		}

		err = idBucket.Put([]byte("add_time"), []byte(time.Now().Format("2006-01-02 15:04:05")))
		if err != nil {
			return fmt.Errorf("Put 'title' error: %s", err)
		}

		m, err := p.Map()
		if err != nil {
			return fmt.Errorf("Getting Map from proxy: %s", err)
		}
		for k, v := range m {
			switch v.(type) {
			case string:
				err = idBucket.Put([]byte(k), []byte(v.(string)))
				if err != nil {
					return fmt.Errorf("Put '%s' error: %s", k, err)
				}
			case map[string]interface{}:
				paramBucket, err := idBucket.CreateBucketIfNotExists([]byte(k))
				for kk, vv := range v.(map[string]interface{}) {
					if _, ok := vv.(string); ok {
						err = paramBucket.Put([]byte(kk), []byte(vv.(string)))
						if err != nil {
							return fmt.Errorf("Put '%s' error: %s", kk, err)
						}
					}
				}
			default:
				fmt.Printf("%v's value is not string/map[string]interface{}: %T\n", k, v)
			}
		}
		return nil
	})
}

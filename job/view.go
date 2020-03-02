package job

import (
	"fmt"
	"net/http"

	bolt "go.etcd.io/bbolt"
)

func View(db *bolt.DB, rootName []byte, w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Method")
	err := db.View(func(tx *bolt.Tx) error {
		fmt.Println("DB View")
		bucket := tx.Bucket(rootName)
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
}

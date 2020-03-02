package job

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

func Execute(db *bolt.DB, rootName []byte) {
	for {
		user := ""
		buildURL := ""
		observeURL := ""
		err := db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(rootName)
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

type BuildNumber struct {
	Number int `json:"number"`
}

type WaitJson struct {
	InQueue            bool        `json:"inQueue"`
	LastBuild          BuildNumber `json:"lastBuild"`
	LastCompletedBuild BuildNumber `json:"lastCompletedBuild"`
}

func waitJobComplete(token, obtainURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", obtainURL, nil)
	if err != nil {
		return fmt.Errorf("request generation error: %s", err)
	}
	authText := strings.SplitN(token, ":", 2)
	user := authText[0]
	pass := authText[1]
	req.SetBasicAuth(user, pass)

	for {
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request sending error: %s", err)
		}
		defer resp.Body.Close()

		var jsonData WaitJson
		err = decodeJSONFromResponse(resp, &jsonData)
		if err != nil {
			return fmt.Errorf("Decode JSON Error: %s", err)
		}

		if jsonData.InQueue {
			time.Sleep(1 * time.Second)
			continue
		}

		if jsonData.LastBuild.Number != jsonData.LastCompletedBuild.Number {
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
}

func decodeJSONFromResponse(r *http.Response, jsonData *WaitJson) (err error) {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&jsonData); err != nil {
		return fmt.Errorf("Json Unmarshal Error: %s", err)
	}
	return nil
}

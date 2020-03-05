package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

func (jh *JobHandler) Execute() {
	for {
		params := make(map[string]string, 0)
		user := ""
		buildURL := ""
		observeURL := ""
		err := jh.Db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(jh.RootName)
			if bucket != nil {
				cursor := bucket.Cursor()
				k, _ := cursor.First()
				if len(k) > 0 {
					idBucket := bucket.Bucket(k)
					fmt.Println(idBucket)
					idBucket.ForEach(func(k, v []byte) error {
						switch string(k) {
						case "user":
							user = string(v)
						case "buildURL":
							buildURL = string(v)
						case "observeURL":
							observeURL = string(v)
						case "parameter":
							paramBucket := idBucket.Bucket(k)
							paramBucket.ForEach(func(kk, vv []byte) error {
								params[string(kk)] = string(vv)
								return nil
							})
						}
						return nil
					})
					err := bucket.DeleteBucket(k)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println("execJob Error:", err)
			<-jh.UpdateCh
			continue
		}

		if user == "" || buildURL == "" || observeURL == "" {
			<-jh.UpdateCh
			continue
		}

		fmt.Println("Execute")
		select {
		case jh.Ch <- 1:
		default:
		}

		err = TriggerJob(user, buildURL, params)
		if err != nil {
			fmt.Println("trigger job error: ", err)
			<-jh.UpdateCh
			continue
		}

		err = WaitJobComplete(user, observeURL)
		if err != nil {
			fmt.Println("wait job complete error: ", err)
		}
	}
}

func splitToken(token string) (user, pass string) {
	authText := strings.SplitN(token, ":", 2)
	user = authText[0]
	pass = authText[1]
	return
}

func generateJsonString(kv map[string]string) (io.Reader, error) {
	if len(kv) == 0 {
		return nil, nil
	}

	jsonMap := make(map[string][]JobParams, 0)
	jsonParams := make([]JobParams, 0)
	for k, v := range kv {
		jsonParams = append(jsonParams, JobParams{Name: k, Value: v})
	}
	jsonMap["parameter"] = jsonParams
	text, err := json.Marshal(jsonMap)
	jsonStr := []byte("json=")
	jsonStr = append(jsonStr, text...)

	return bytes.NewBuffer(jsonStr), err
}

type JobParams struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func TriggerJob(token, url string, kv map[string]string) error {
	jsonReader, err := generateJsonString(kv)
	if err != nil {
		return fmt.Errorf("map cannot be marshaled: %s", err)
	}
	var req *http.Request
	if jsonReader == nil {
		req, err = http.NewRequest("GET", url, nil)
	} else {
		req, err = http.NewRequest("POST", url, jsonReader)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if err != nil {
		return fmt.Errorf("request generation url error: %s", err)
	}
	user, pass := splitToken(token)
	req.SetBasicAuth(user, pass)

	fmt.Println("== request ==")
	fmt.Println("Method:", req.Method)
	fmt.Println("URL:", req.URL)
	fmt.Println("Header:", req.Header)
	fmt.Println("Body:", req.Body)
	fmt.Println("=============")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request sending error: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("response status is not OK: %d", resp.StatusCode)
	}

	return nil
}

type BuildNumber struct {
	Number int `json:"number"`
}

type WaitJson struct {
	InQueue            bool        `json:"inQueue"`
	LastBuild          BuildNumber `json:"lastBuild"`
	LastCompletedBuild BuildNumber `json:"lastCompletedBuild"`
}

func WaitJobComplete(token, obtainURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", obtainURL, nil)
	if err != nil {
		return fmt.Errorf("request generation error: %s", err)
	}
	user, pass := splitToken(token)
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

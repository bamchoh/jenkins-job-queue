package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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

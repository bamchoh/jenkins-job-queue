package main

import (
	"net/http"
	"test_httpserver/job"
)

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		job.Update(db, bucketName, w, r)
	case http.MethodGet:
		job.View(db, bucketName, w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

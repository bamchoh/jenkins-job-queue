package main

import (
	"net/http"
	"test_httpserver/job"
)

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		job.Update(db, rootName, w, r)
	case http.MethodGet:
		job.View(db, rootName, w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

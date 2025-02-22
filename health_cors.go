package main

import (
	"fmt"
	"net/http"
	"slices"
)

func handleHeyGot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "")
}

func handleOptions(w http.ResponseWriter, r *http.Request) {

	origin := r.Header.Get("Origin")

	if slices.Contains(allowedOrigins, origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PUT")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		fmt.Fprint(w, "")
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

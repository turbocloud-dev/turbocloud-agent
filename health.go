package main

import (
	"fmt"
	"net/http"
)

func handleHeyGot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "")
}

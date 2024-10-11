package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Proxy struct {
	ContainerId     string
	ServerPrivateIP string
	Port            string
	Domain          string
}

func handleProxyPost(w http.ResponseWriter, r *http.Request) {
	//g := r.PathValue("id")
	//fmt.Printf("POST proxy id: %v\n", g)

	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return
		}
	}

	var proxy Proxy

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&proxy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "POST /proxy, body: %+v", proxy)
	fmt.Printf("POST /proxy, body: %s", proxy.ContainerId)

}

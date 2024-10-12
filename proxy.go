package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type Proxy struct {
	Id              int64
	ContainerId     string
	ServerPrivateIP string
	Port            string
	Domain          string
}

func handleProxyPost(w http.ResponseWriter, r *http.Request) {
	//g := r.PathValue("id")
	//fmt.Printf("POST proxy id: %v\n", g)

	var proxy Proxy
	err := decodeJSONBody(w, r, &proxy)

	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	addProxy(&proxy)

	jsonBytes, err := json.Marshal(proxy)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleProxyGet(w http.ResponseWriter, r *http.Request) {

	jsonBytes, err := json.Marshal(getAllProxies())
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

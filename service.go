package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type Service struct {
	Id           string
	GitURL       string
	ProjectId    string
	Environments []Proxy
}

func handleServicePost(w http.ResponseWriter, r *http.Request) {
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

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Proxy:", err)
		return
	}

	proxy.Id = id
	addProxy(&proxy)

	jsonBytes, err := json.Marshal(proxy)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

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
	Environments []Environment
}

func handleServicePost(w http.ResponseWriter, r *http.Request) {
	var service Service
	err := decodeJSONBody(w, r, &service)

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

	addService(&service)

	jsonBytes, err := json.Marshal(service)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleServiceGet(w http.ResponseWriter, r *http.Request) {

	jsonBytes, err := json.Marshal(getAllServices())
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

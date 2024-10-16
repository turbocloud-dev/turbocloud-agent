package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

type Environment struct {
	Id        string
	Name      string
	Branch    string
	Domains   []string
	Port      string
	ServiceId string
}

func handleEnvironmentPost(w http.ResponseWriter, r *http.Request) {
	var environment Environment
	err := decodeJSONBody(w, r, &environment)

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

	addEnvironment(&environment)

	jsonBytes, err := json.Marshal(environment)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleEnvironmentByServiceIdGet(w http.ResponseWriter, r *http.Request) {

	serviceId := r.PathValue("serviceId")

	jsonBytes, err := json.Marshal(loadEnvironmentsByServiceId(serviceId))
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func handleEnvironmentDelete(w http.ResponseWriter, r *http.Request) {
	environmentId := r.PathValue("id")

	if !deleteEnvironment(environmentId) {
		fmt.Println("Cannot delete a record from Environment table")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "")

}

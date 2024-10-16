package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rqlite/gorqlite"
)

type Environment struct {
	Id         string
	Name       string
	Branch     string
	Domains    []string
	MachineIds []string
	Port       string
	ServiceId  string
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

/*Database*/
/*Environments*/

func addEnvironment(environment *Environment) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	environment.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Environment( Id, ServiceId, Name, Branch, Domains, Port, MachineIds) VALUES(?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{environment.Id, environment.ServiceId, environment.Name, environment.Branch, strings.Join(environment.Domains, ";"), environment.Port, strings.Join(environment.MachineIds, ";")},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Environment table: %s\n", err.Error())
	}
}

func loadEnvironmentsByServiceId(serviceId string) []Environment {
	var environments = []Environment{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ServiceId, Name, Branch, Domains, Port from Environment where ServiceId = ?",
			Arguments: []interface{}{serviceId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Environment table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Name string
		var Branch string
		var Domains string
		var Port string
		var ServiceId string

		err := rows.Scan(&Id, &ServiceId, &Name, &Branch, &Domains, &Port)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedEnvironment := Environment{
			Id:        Id,
			ServiceId: ServiceId,
			Name:      Name,
			Branch:    Branch,
			Domains:   strings.Split(Domains, ";"),
			Port:      Port,
		}
		environments = append(environments, loadedEnvironment)
	}

	return environments
}

func getEnvironmentById(environmentId string) *Environment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ServiceId, Name, Branch, Domains, Port from Environment where Id = ?",
			Arguments: []interface{}{environmentId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Environment table: %s\n", err.Error())
		return nil
	}

	if rows.NumRows() == 0 {
		return nil
	}

	rows.Next()

	var Id string
	var Name string
	var Branch string
	var Domains string
	var Port string
	var ServiceId string

	err = rows.Scan(&Id, &ServiceId, &Name, &Branch, &Domains, &Port)
	if err != nil {
		fmt.Printf(" Cannot run Scan: %s\n", err.Error())
	}
	loadedEnvironment := Environment{
		Id:        Id,
		ServiceId: ServiceId,
		Name:      Name,
		Branch:    Branch,
		Domains:   strings.Split(Domains, ";"),
		Port:      Port,
	}
	return &loadedEnvironment

}

func deleteEnvironment(environmentId string) (result bool) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "DELETE FROM Environment WHERE Id = ?",
				Arguments: []interface{}{environmentId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot delete a record from Environment table: %s\n", err.Error())
		return false
	}

	return true
}

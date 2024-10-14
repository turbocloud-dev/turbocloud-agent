package main

import (
	"fmt"
	"strings"

	"github.com/rqlite/gorqlite"
)

var connection *gorqlite.Connection

func databaseInit() {
	var err error
	connection, err = gorqlite.Open("http://") // same only explicitly
	if err != nil {
		fmt.Printf(" Cannot open database: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Proxy (Id TEXT NOT NULL PRIMARY KEY, ContainerId TEXT, ServerPrivateIP TEXT, Port TEXT, Domain TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Proxy: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Service (Id TEXT NOT NULL PRIMARY KEY, ProjectId TEXT, GitURL TEXT, Environments TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Service: %s\n", err.Error())
	}

	getAllProxies()
}

func addProxy(proxy *Proxy) {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Proxy:", err)
		return
	}

	proxy.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Proxy( Id, ContainerId, ServerPrivateIP, Port, Domain) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{proxy.Id, proxy.ContainerId, proxy.ServerPrivateIP, proxy.Port, proxy.Domain},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Proxy table: %s\n", err.Error())
	}

}

func deleteProxy(proxyId string) (result bool) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "DELETE FROM Proxy WHERE Id = ?",
				Arguments: []interface{}{proxyId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot delete a record from Proxy table: %s\n", err.Error())
		return false
	}

	return true
}

func getAllProxies() []Proxy {
	var proxies = []Proxy{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ContainerId, ServerPrivateIP, Port, Domain from Proxy where Id > ?",
			Arguments: []interface{}{0},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Proxy table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var ContainerId string
		var ServerPrivateIP string
		var Port string
		var Domain string

		err := rows.Scan(&Id, &ContainerId, &ServerPrivateIP, &Port, &Domain)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadProxy := Proxy{
			Id:              Id,
			ContainerId:     ContainerId,
			ServerPrivateIP: ServerPrivateIP,
			Port:            Port,
			Domain:          Domain,
		}
		proxies = append(proxies, loadProxy)
	}
	return proxies
}

/*Services*/

func addService(service *Service) {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Service:", err)
		return
	}

	service.Id = id

	//Save Environemnts and generate string with environment IDs to save in DB
	environemntIds := []string{}
	for envIndex := range service.Environments {
		fmt.Printf(" Environment: %s\n", service.Environments[envIndex].Name)
		addEnvironment(&service.Environments[envIndex])
		environemntIds = append(environemntIds, service.Environments[envIndex].Id)
	}

	environemntIdsString := strings.Join(environemntIds, ";")

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Service( Id, ProjectId, GitURL, Environments) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{service.Id, service.ProjectId, service.GitURL, environemntIdsString},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Proxy table: %s\n", err.Error())
	}

}

func getAllServices() []Service {
	var services = []Service{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ProjectId, GitURL, Environments from Service where Id > ?",
			Arguments: []interface{}{0},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Proxy table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var ProjectId string
		var GitURL string
		var EnvironmentIdsString string

		err := rows.Scan(&Id, &ProjectId, &GitURL, &EnvironmentIdsString)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}

		environmentIds := strings.Split(EnvironmentIdsString, ";")
		environments := []Environment{}

		for _, environmentId := range environmentIds {
			environments = append(environments, loadEnvironmentById(environmentId))
		}

		loadedService := Service{
			Id:           Id,
			ProjectId:    ProjectId,
			GitURL:       GitURL,
			Environments: environments,
		}
		services = append(services, loadedService)
	}
	return services
}

func addEnvironment(environment *Environment) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	environment.Id = id
}

func loadEnvironmentById(environmentId string) Environment {
	var loadedEnvironment Environment
	loadedEnvironment.Id = environmentId
	return loadedEnvironment
}

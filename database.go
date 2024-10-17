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
				Query:     "CREATE TABLE Service (Id TEXT NOT NULL PRIMARY KEY, Name TEXT, ProjectId TEXT, GitURL TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Service: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Environment (Id TEXT NOT NULL PRIMARY KEY, ServiceId TEXT, Name TEXT, Branch TEXT, Domains TEXT, Port TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Environment: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Deployment (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, MachineIds TEXT, EnvironmentId TEXT, ImageId TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Deployment: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Machine (Id TEXT NOT NULL PRIMARY KEY, VPNIp TEXT, PublicIp TEXT, CloudPrivateIp TEXT, Name TEXT, Types TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Deployment: %s\n", err.Error())
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
			Query:     "SELECT Id, ContainerId, ServerPrivateIP, Port, Domain from Proxy",
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
	/*environemntIds := []string{}
	for envIndex := range service.Environments {
		fmt.Printf(" Environment: %s\n", service.Environments[envIndex].Name)
		addEnvironment(&service.Environments[envIndex])
		environemntIds = append(environemntIds, service.Environments[envIndex].Id)
	}

	environemntIdsString := strings.Join(environemntIds, ";")
	*/
	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Service( Id, Name, ProjectId, GitURL) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{service.Id, service.Name, service.ProjectId, service.GitURL},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Service table: %s\n", err.Error())
	}

}

func getAllServices() []Service {
	var services = []Service{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Name, ProjectId, GitURL from Service where Id > ?",
			Arguments: []interface{}{0},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Service table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Name string
		var ProjectId string
		var GitURL string

		err := rows.Scan(&Id, &Name, &ProjectId, &GitURL)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}

		/*environmentIds := strings.Split(EnvironmentIdsString, ";")
		environments := []Environment{}

		for _, environmentId := range environmentIds {
			environments = append(environments, loadEnvironmentById(environmentId))
		}*/

		loadedService := Service{
			Id:        Id,
			Name:      Name,
			ProjectId: ProjectId,
			GitURL:    GitURL,
		}
		services = append(services, loadedService)
	}
	return services
}

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
				Query:     "INSERT INTO Environment( Id, ServiceId, Name, Branch, Domains, Port) VALUES(?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{environment.Id, environment.ServiceId, environment.Name, environment.Branch, strings.Join(environment.Domains, ";"), environment.Port},
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

/*Deployments*/
func addDeployment(deployment *Deployment) {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Service:", err)
		return
	}

	deployment.Id = id
	machineIds := ""

	if len(deployment.MachineIds) > 0 {
		machineIds = strings.Join(deployment.MachineIds, ";")
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Deployment( Id, Status, MachineIds, EnvironmentId, ImageId) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{deployment.Id, deployment.Status, machineIds, deployment.EnvironmentId, deployment.ImageId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}

}

func addMachine(machine *Machine) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	machine.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Machine( Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types) VALUES(?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{machine.Id, machine.VPNIp, machine.PublicIp, machine.CloudPrivateIp, machine.Name, strings.Join(machine.Types, ";")},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Machine table: %s\n", err.Error())
	}
}

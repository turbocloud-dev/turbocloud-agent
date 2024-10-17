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
				Query:     "CREATE TABLE Environment (Id TEXT NOT NULL PRIMARY KEY, ServiceId TEXT, Name TEXT, Branch TEXT, Domains TEXT, Port TEXT, MachineIds TEXT)",
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
				Query:     "CREATE TABLE Deployment (Id TEXT NOT NULL PRIMARY KEY, EnvironmentId TEXT, ImageId TEXT, DeploymentJobIds TEXT)",
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

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE Image (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, DeploymentId TEXT, ErrorMsg TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table Image: %s\n", err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "CREATE TABLE DeploymentJob (Id TEXT NOT NULL PRIMARY KEY, Status TEXT, DeploymentId TEXT, MachineId TEXT)",
				Arguments: []interface{}{},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot create table DeploymentJob: %s\n", err.Error())
	}

	getAllProxies()
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

/*Deployments*/
func addDeployment(deployment *Deployment) {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Deployment( Id, EnvironmentId, ImageId, DeploymentJobIds) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{deployment.Id, deployment.EnvironmentId, deployment.ImageId, strings.Join(deployment.DeploymentJobIds, ";")},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}

}

func getDeploymentsById(deploymentId string) []Deployment {
	var deployments = []Deployment{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, EnvironmentId, ImageId, DeploymentJobIds from Deployment where id = ?",
			Arguments: []interface{}{deploymentId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Deployment table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var EnvironmentId string
		var ImageId string
		var DeploymentJobIds string

		err := rows.Scan(&Id, &EnvironmentId, &ImageId, &DeploymentJobIds)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedDeployment := Deployment{
			Id:               Id,
			EnvironmentId:    EnvironmentId,
			ImageId:          ImageId,
			DeploymentJobIds: strings.Split(DeploymentJobIds, ";"),
		}
		deployments = append(deployments, loadedDeployment)
	}

	return deployments
}

/*Machines*/
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

func getMachines() []Machine {
	var machines = []Machine{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, VPNIp, PublicIp, CloudPrivateIp, Name, Types from Machine",
			Arguments: []interface{}{},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Environment table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var VPNIp string
		var PublicIp string
		var CloudPrivateIp string
		var Name string
		var Types string

		err := rows.Scan(&Id, &VPNIp, &PublicIp, &CloudPrivateIp, &Name, &Types)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedMachine := Machine{
			Id:             Id,
			VPNIp:          VPNIp,
			PublicIp:       PublicIp,
			CloudPrivateIp: CloudPrivateIp,
			Name:           Name,
			Types:          strings.Split(Types, ";"),
		}
		machines = append(machines, loadedMachine)
	}

	return machines
}

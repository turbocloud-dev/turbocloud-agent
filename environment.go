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
	Id                   string
	Name                 string
	Branch               string
	GitTag               string
	Domains              []string
	MachineIds           []string
	Port                 string
	ServiceId            string
	LastDeploymentStatus string
}

func handleEnvironmentPost(w http.ResponseWriter, r *http.Request) {
	var environment Environment
	err := decodeJSONBody(w, r, &environment, true)

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

func handleEnvironmentByServiceIdPut(w http.ResponseWriter, r *http.Request) {

	var environment Environment
	err := decodeJSONBody(w, r, &environment, true)

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

	if !updateEnvironment(environment) {
		fmt.Println("Cannot update a record from Environment table")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "")
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

func addEnvironment(environment *Environment) {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Environment:", err)
		return
	}

	environment.Id = id

	//Check if we should assign a domain automatically
	if len(environment.Domains) == 0 {
		//Get IP address of load balancers
		machines := getMachinesWithType(MachineTypeBalancer)
		for _, machine := range machines {
			publicIp := machine.PublicIp
			if publicIp != "" {
				domain := strings.ToLower(environment.Id) + "-" + strings.Replace(publicIp, ".", "-", -1) + ".dns.turbocloud.dev"
				environment.Domains = append(environment.Domains, domain)
			}
		}
	}

	//Check if we should add to Environment.MachineIds the first machine because there is no MachineIds in the request body
	if len(environment.MachineIds) == 0 {
		machines := getMachines()
		if len(machines) > 0 {
			environment.MachineIds = append(environment.MachineIds, machines[0].Id)
		}
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Environment( Id, ServiceId, Name, Branch, Domains, Port, MachineIds, GitTag) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{environment.Id, environment.ServiceId, environment.Name, environment.Branch, strings.Join(environment.Domains, ";"), environment.Port, strings.Join(environment.MachineIds, ";"), environment.GitTag},
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
			Query:     "SELECT Id, ServiceId, Name, Branch, Domains, Port, MachineIds, GitTag from Environment where ServiceId = ?",
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
		var MachineIds string
		var GitTag string

		err := rows.Scan(&Id, &ServiceId, &Name, &Branch, &Domains, &Port, &MachineIds, &GitTag)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedEnvironment := Environment{
			Id:         Id,
			ServiceId:  ServiceId,
			Name:       Name,
			Branch:     Branch,
			Domains:    strings.Split(Domains, ";"),
			Port:       Port,
			MachineIds: strings.Split(MachineIds, ";"),
			GitTag:     GitTag,
		}

		//Get a status of the most recent deployment
		deployments := getLastDeploymentByEnvironmentId(Id)
		if len(deployments) > 0 {
			loadedEnvironment.LastDeploymentStatus = deployments[0].Status
		} else {
			loadedEnvironment.LastDeploymentStatus = "no deployments"
		}

		environments = append(environments, loadedEnvironment)
	}

	return environments
}

func getEnvironmentById(environmentId string) *Environment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ServiceId, Name, Branch, Domains, Port, MachineIds, GitTag from Environment where Id = ?",
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
	var MachineIds string
	var GitTag string

	err = rows.Scan(&Id, &ServiceId, &Name, &Branch, &Domains, &Port, &MachineIds, &GitTag)
	if err != nil {
		fmt.Printf(" Cannot run Scan: %s\n", err.Error())
	}
	loadedEnvironment := Environment{
		Id:         Id,
		ServiceId:  ServiceId,
		Name:       Name,
		Branch:     Branch,
		Domains:    strings.Split(Domains, ";"),
		Port:       Port,
		MachineIds: strings.Split(MachineIds, ";"),
		GitTag:     GitTag,
	}
	return &loadedEnvironment

}

func getEnvironmentByServiceIdAndName(serviceId string, branchName string) *Environment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, ServiceId, Name, Branch, Domains, Port, MachineIds, GitTag from Environment where ServiceId = ? AND Branch = ?",
			Arguments: []interface{}{serviceId, branchName},
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
	var MachineIds string
	var GitTag string

	err = rows.Scan(&Id, &ServiceId, &Name, &Branch, &Domains, &Port, &MachineIds, &GitTag)
	if err != nil {
		fmt.Printf(" Cannot run Scan: %s\n", err.Error())
	}
	loadedEnvironment := Environment{
		Id:         Id,
		ServiceId:  ServiceId,
		Name:       Name,
		Branch:     Branch,
		Domains:    strings.Split(Domains, ";"),
		Port:       Port,
		MachineIds: strings.Split(MachineIds, ";"),
		GitTag:     GitTag,
	}
	return &loadedEnvironment

}

func updateEnvironment(environment Environment) (result bool) {
	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Environment SET Name = ?, Branch = ?, Domains = ?, Port = ?, MachineIds = ?, GitTag = ? WHERE Id = ?",
				Arguments: []interface{}{environment.Name, environment.Branch, strings.Join(environment.Domains, ";"), environment.Port, strings.Join(environment.MachineIds, ";"), environment.GitTag, environment.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a record from Environment table: %s\n", err.Error())
		return false
	}

	return true
}

func deleteEnvironment(environmentId string) (result bool) {

	environment := getEnvironmentById(environmentId)
	//Schedule Jobs to delete containers
	for _, machineId := range environment.MachineIds {
		var job ContainerJob
		job.MachineId = machineId
		job.Status = ContainerJobStatusPlanned
		job.JobType = ContainerJobTypeDelete
		job.EnvironmentId = environmentId
		addContainerJob(job)
	}

	//Schedule ImageJob to delete images
	var imageJob ImageJob
	imageJob.Status = ImageJobStatusPlanned
	imageJob.JobType = ImageJobTypeDelete
	imageJob.EnvironmentId = environmentId
	addImageJob(imageJob)

	//Remove a proxy record to update Caddyfile
	deleteProxyByEnvironmentId(environmentId)

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

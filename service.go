package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/rqlite/gorqlite"
)

type Service struct {
	Id        string
	Name      string
	GitURL    string
	ProjectId string
}

func handleServicePost(w http.ResponseWriter, r *http.Request) {
	var service Service
	err := decodeJSONBody(w, r, &service, true)

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

/*Database*/

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
			Query:     "SELECT Id, Name, ProjectId, GitURL from Service",
			Arguments: []interface{}{},
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

func getServiceById(serviceId string) *Service {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Name, ProjectId, GitURL from Service where Id = ?",
			Arguments: []interface{}{serviceId},
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
	var ProjectId string
	var GitURL string

	err = rows.Scan(&Id, &Name, &ProjectId, &GitURL)
	if err != nil {
		fmt.Printf(" Cannot run Scan: %s\n", err.Error())
	}
	loadedService := Service{
		Id:        Id,
		ProjectId: ProjectId,
		GitURL:    GitURL,
	}
	return &loadedService

}

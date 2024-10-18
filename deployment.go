/*
We create Deployment after get POST /deploy/:environmnent_id
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/rqlite/gorqlite"
)

const DeploymentStatusScheduled = "scheduled"
const DeploymentStatusBuildingImage = "building_image"
const DeploymentStatusStartingContainers = "starting_containers"
const DeploymentStatusFinished = "finished"

type Deployment struct {
	Id               string
	Status           string
	EnvironmentId    string
	ImageId          string
	DeploymentJobIds []string
}

func handleDeploymentPost(w http.ResponseWriter, r *http.Request) {
	var deployment Deployment
	environmentId := r.PathValue("environmentId")

	var environment = getEnvironmentById(environmentId)
	if environment == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	err := decodeJSONBody(w, r, &deployment)

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
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return
	}

	deployment.Id = id
	deployment.EnvironmentId = environmentId
	deployment.Status = DeploymentStatusScheduled

	//Create a new image and schedule image building
	var image Image
	image.DeploymentId = deployment.Id
	image.Status = ImageStatusToBuild
	newImage := addImage(image)

	//Create a deployment
	deployment.ImageId = newImage.Id
	addDeployment(&deployment)

	//Schedule DeploymentJobs
	for _, machineId := range environment.MachineIds {
		var job DeploymentJob
		job.MachineId = machineId
		job.Status = StatusToDeploy
		job.DeploymentId = deployment.Id
		addDeploymentJob(job)
	}

	jsonBytes, err := json.Marshal(deployment)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func startDeploymentCheckerWorker() {
	for range time.Tick(time.Second * 3) {
		go func() {
			scheduledDeployments := getDeploymentsByStatus(DeploymentStatusScheduled)
			for _, deployment := range scheduledDeployments {
				fmt.Println("Deployment: " + deployment.Id + ": " + " Status: " + deployment.Status)
				images := getImageByDeploymentIdAndStatus(deployment.Id, ImageStatusToBuild)
				if len(images) > 0 {
					fmt.Println("Image: " + images[0].Id + ": " + images[0].Status)
					buildImage(images[0], deployment)
				}
			}
		}()
	}
}

/*Database*/

func addDeployment(deployment *Deployment) {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Deployment( Id, Status, EnvironmentId, ImageId, DeploymentJobIds) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{deployment.Id, deployment.Status, deployment.EnvironmentId, deployment.ImageId, strings.Join(deployment.DeploymentJobIds, ";")},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}

}

func getDeploymentsById(deploymentId string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, DeploymentJobIds from Deployment where id = ?",
			Arguments: []interface{}{deploymentId},
		},
	)

	return handleQuery(rows, err)

}

func getDeploymentsByStatus(status string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, DeploymentJobIds from Deployment where Status = ?",
			Arguments: []interface{}{status},
		},
	)

	return handleQuery(rows, err)
}

func handleQuery(rows gorqlite.QueryResult, err error) []Deployment {

	var deployments = []Deployment{}

	if err != nil {
		fmt.Printf(" Cannot read from Deployment table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var EnvironmentId string
		var ImageId string
		var DeploymentJobIds string

		err := rows.Scan(&Id, &Status, &EnvironmentId, &ImageId, &DeploymentJobIds)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedDeployment := Deployment{
			Id:               Id,
			Status:           Status,
			EnvironmentId:    EnvironmentId,
			ImageId:          ImageId,
			DeploymentJobIds: strings.Split(DeploymentJobIds, ";"),
		}
		deployments = append(deployments, loadedDeployment)
	}

	return deployments

}

/*
We create Deployment for each deployment on each server. We Environment has 5 MachineIds where we should deploy we should create 5 Deployments
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Deployment struct {
	Id               string
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
	deployment.EnvironmentId = environmentId

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
			deploymentsToDeploy := getDeploymentJobsByStatus(StatusToDeploy)
			for _, deploymentJob := range deploymentsToDeploy {
				fmt.Println("Deployment: " + deploymentJob.Id + ": " + " Status: " + deploymentJob.Status)
				images := getImageByDeploymentIdAndStatus(deploymentJob.DeploymentId, ImageStatusToBuild)
				if len(images) > 0 {
					fmt.Println("Image: " + images[0].Id + ": " + images[0].Status)
				}
			}
		}()
	}
}

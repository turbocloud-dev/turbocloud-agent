/*
Deployments
*/

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rqlite/gorqlite"
)

const DeploymentStatusScheduled = "scheduled"
const DeploymentStatusBuildingImage = "building_image"
const DeploymentStatusStartingContainers = "starting_containers"
const DeploymentStatusFinished = "finished"

type Deployment struct {
	Id            string
	Status        string
	EnvironmentId string
	ImageId       string
	SourceFolder  string
}

type GitHubPayload struct {
	Ref string `json:"ref"`
}

type BitbucketPayload struct {
	Push struct {
		Changes []struct {
			New struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"new"`
			Old struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"old"`
		} `json:"changes"`
	} `json:"push"`
}

func handleEnvironmentDeploymentGet(w http.ResponseWriter, r *http.Request) {
	var deployment Deployment
	environmentId := r.PathValue("environmentId")

	var environment = getEnvironmentById(environmentId)
	if environment == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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
	image.EnvironmentId = environmentId
	image.Status = ImageStatusToBuild
	newImage := addImage(image)

	//Create a deployment
	deployment.ImageId = newImage.Id
	addDeployment(&deployment)

	fmt.Println("Scheduling DeploymentJobs ")
	fmt.Println(len(environment.MachineIds))

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

func handleEnvironmentDeploymentPost(w http.ResponseWriter, r *http.Request) {
	var deployment Deployment
	environmentId := r.PathValue("environmentId")

	var environment = getEnvironmentById(environmentId)
	if environment == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	err := decodeJSONBody(w, r, &deployment, false)

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
	image.EnvironmentId = environmentId
	image.Status = ImageStatusToBuild
	newImage := addImage(image)

	//Create a deployment
	deployment.ImageId = newImage.Id
	addDeployment(&deployment)

	fmt.Println("Scheduling DeploymentJobs ")
	fmt.Println(len(environment.MachineIds))

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

func handleServiceDeploymentPost(w http.ResponseWriter, r *http.Request) {
	var deployment Deployment

	serviceId := r.PathValue("serviceId")

	var service = getServiceById(serviceId)
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	//We should load environment by branch name and service Id
	//Check header to know the webhook source
	branchName := ""

	agentHeader := r.Header.Get("User-Agent")
	if agentHeader != "" {
		if strings.Contains(agentHeader, "GitHub") {
			fmt.Println("New GitHub webhook event is received")

			var githubPayload GitHubPayload
			err := decodeJSONBody(w, r, &githubPayload, false)

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

			if !strings.Contains(githubPayload.Ref, "refs/heads/") {
				fmt.Println("GitHub Handler: Current version doesn't support deployments by Git tags:", err)
				return
			}
			branchName = strings.Replace(githubPayload.Ref, "refs/heads/", "", 1)
		} else if strings.Contains(agentHeader, "Bitbucket") {
			fmt.Println("New Bitbucket webhook event is received")

			var bitbucketPayload BitbucketPayload
			err := decodeJSONBody(w, r, &bitbucketPayload, false)

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
			if bitbucketPayload.Push.Changes[0].New.Type == "branch" {
				branchName = bitbucketPayload.Push.Changes[0].New.Name
			} else {
				fmt.Println("Bitbucket Handler: Current version doesn't support deployments by Git tags:", err)
			}

		}
	}

	if branchName == "" {
		fmt.Println("Cannot parse Git branch from payload")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	environment := getEnvironmentByServiceIdAndName(serviceId, branchName)
	fmt.Println("Found environment getEnvironmentByServiceIdAndName:", environment.Id)

	/////////////////////////////////////////////////////////
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return
	}

	deployment.Id = id
	deployment.EnvironmentId = environment.Id
	deployment.Status = DeploymentStatusScheduled

	//Create a new image and schedule image building
	var image Image
	image.DeploymentId = deployment.Id
	image.Status = ImageStatusToBuild
	image.EnvironmentId = environment.Id
	newImage := addImage(image)

	//Create a deployment
	deployment.ImageId = newImage.Id
	addDeployment(&deployment)

	fmt.Println("Scheduling DeploymentJobs ")
	fmt.Println(len(environment.MachineIds))

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
	for range time.Tick(time.Second * 2) {
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

			readyToStartContainersDeployments := getDeploymentsByStatus(DeploymentStatusStartingContainers)
			for _, deployment := range readyToStartContainersDeployments {
				fmt.Println("Deployment: " + deployment.Id + ": " + " Status: " + deployment.Status)

				//Check if there is already a built image for this deployment
				images := getImageByDeploymentIdAndStatus(deployment.Id, ImageStatusReady)

				if len(images) > 0 {
					//The image has been built and uploaded to a container registry
					//Time to pull the image and start containers

					//Get DeploymentJobs with status StatusToDeploy and DeploymentId = deployment.Id
					jobs := getDeploymentJobsByDeploymentIdAndStatus(deployment.Id, StatusToDeploy)

					if len(jobs) > 0 {
						//Deploy on this machine if job.MachineId = machine_id, where we run this code
						for _, job := range jobs {
							if job.MachineId == thisMachine.Id {
								deployImage(images[0], job, deployment)
							}
						}
					} else {
						//All DeploymentJobs are finished, update status of Deployment
						updateDeploymentStatus(deployment, DeploymentStatusFinished)
					}
				}
			}
		}()
	}
}

func updateDeploymentStatus(deployment Deployment, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Deployment SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, deployment.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf("Cannot update a row in Deployment: %s\n", err.Error())
		return err
	}

	return nil
}

func deployImage(image Image, job DeploymentJob, deployment Deployment) {
	fmt.Println("Pulling and starting a container from Image " + image.Id)
	//Pull the image from a container registry over VPN
	//Start a container
	err := updateDeploymentJobStatus(job, StatusInProgress)

	if err != nil {
		fmt.Println("Cannot update DeploymentJob row:", err)
		return
	}

	//Get environment
	environment := getEnvironmentById(deployment.EnvironmentId)

	portInt, err := GetFreePort()
	if err != nil {
		fmt.Println("Cannot get a free port on this machine:", err)
	}
	port := strconv.Itoa(portInt)

	scriptTemplate := createTemplate("caddyfile", `
	#!/bin/sh
	docker image pull {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
	docker container run -p {{.MACHINE_PORT}}:{{.SERVICE_PORT}} -d --restart unless-stopped --log-driver=journald --name {{.DEPLOYMENT_ID}} {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
`)

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"IMAGE_ID":              image.Id,
		"SERVICE_PORT":          environment.Port,
		"MACHINE_PORT":          port,
		"DEPLOYMENT_ID":         deployment.Id,
		"CONTAINER_REGISTRY_IP": containerRegistryIp,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	err, _ = executeScriptString(scriptString)
	if err != nil {
		fmt.Println("Cannot start the image")
		return
	}

	fmt.Println("Image " + image.Id + " has been started on machine " + job.MachineId)
	err = updateDeploymentJobStatus(job, StatusDeployed)

	if err != nil {
		fmt.Printf(" Cannot update a row in DeploymentJob: %s\n", err.Error())
		return
	}

	//Delete all proxies with the same EnvironmentId as this deployment
	deleteProxiesIfDeploymentIdNotEqual(deployment.EnvironmentId, deployment.Id)

	//Add a Proxy record
	var proxy Proxy
	proxy.ServerPrivateIP = thisMachine.VPNIp
	proxy.Port = port
	proxy.Domain = environment.Domains[0]
	proxy.EnvironmentId = deployment.EnvironmentId
	proxy.DeploymentId = deployment.Id
	addProxy(&proxy)

	stopPreviousContainer(deployment.EnvironmentId)

}

/*Database*/
func addDeployment(deployment *Deployment) {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Deployment( Id, Status, EnvironmentId, ImageId, SourceFolder) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{deployment.Id, deployment.Status, deployment.EnvironmentId, deployment.ImageId, deployment.SourceFolder},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}

}

func getDeploymentById(deploymentId string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, SourceFolder from Deployment where id = ?",
			Arguments: []interface{}{deploymentId},
		},
	)

	return handleQuery(rows, err)

}

func getLastDeploymentByEnvironmentId(environmentId string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, SourceFolder from Deployment where EnvironmentId = ? ORDER BY CreatedAt DESC LIMIT 1",
			Arguments: []interface{}{environmentId},
		},
	)

	return handleQuery(rows, err)

}

func getDeploymentsByEnvironmentId(environmentId string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, SourceFolder from Deployment where EnvironmentId = ? ORDER BY CreatedAt DESC",
			Arguments: []interface{}{environmentId},
		},
	)

	return handleQuery(rows, err)

}

func getDeploymentsByStatus(status string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId, SourceFolder from Deployment where Status = ? ORDER BY CreatedAt DESC",
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
		var SourceFolder string

		err := rows.Scan(&Id, &Status, &EnvironmentId, &ImageId, &SourceFolder)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedDeployment := Deployment{
			Id:            Id,
			Status:        Status,
			EnvironmentId: EnvironmentId,
			ImageId:       ImageId,
			SourceFolder:  SourceFolder,
		}
		deployments = append(deployments, loadedDeployment)
	}

	return deployments

}

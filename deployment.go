/*
We create Deployment after get POST /deploy/:environmnent_id
*/

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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
				} else {
					//Check if there is already built image for this deployment
					images := getImageByDeploymentIdAndStatus(deployment.Id, ImageStatusToBuild)
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
						}
					}
				}
			}
		}()
	}
}

func deployImage(image Image, job DeploymentJob, deployment Deployment) {
	//Pull the image from a container registry over VPN
	//Start a container
	err := updateDeploymentJobStatus(job, StatusInProgress)

	if err != nil {
		fmt.Println("Cannot update DeploymentJob row:", err)
		return
	}

	//Create default scripting
	randomId, err := NanoId(10)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return
	}

	//Get environment
	environment := getEnvironmentById(deployment.EnvironmentId)

	//Get service
	service := getServiceById(environment.ServiceId)

	scriptTemplate := createTemplate("caddyfile", `
	cd {{.HOME_DIR}}
	git clone --recurse-submodules -b {{.BRANCH_NANE}} {{.REPOSITORY_CLONE_URL}} {{.LOCAL_FOLDER}} 
	docker build {{.LOCAL_FOLDER}} -t {{.IMAGE_ID}}
	docker image tag {{.IMAGE_ID}} localhost:7000/{{.IMAGE_ID}}
	docker image push localhost:7000/{{.IMAGE_ID}}
	#docker manifest inspect --insecure localhost:7000/{{.IMAGE_ID}}
`)

	homeDir, _ := os.UserHomeDir()
	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"HOME_DIR":             homeDir,
		"BRANCH_NANE":          environment.Branch,
		"REPOSITORY_CLONE_URL": service.GitURL,
		"LOCAL_FOLDER":         randomId,
		"IMAGE_ID":             image.Id,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	err = executeScriptString(scriptString)
	if err != nil {
		fmt.Println("Cannot build the image")
		return
	}

	fmt.Println("Image " + image.Id + " has been built. Update status to " + ImageStatusReady)

	err = updateImageStatus(image, ImageStatusReady)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return
	}
}

/*Database*/

func addDeployment(deployment *Deployment) {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Deployment( Id, Status, EnvironmentId, ImageId) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{deployment.Id, deployment.Status, deployment.EnvironmentId, deployment.ImageId},
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
			Query:     "SELECT Id, Status, EnvironmentId, ImageId from Deployment where id = ?",
			Arguments: []interface{}{deploymentId},
		},
	)

	return handleQuery(rows, err)

}

func getDeploymentsByStatus(status string) []Deployment {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, ImageId from Deployment where Status = ?",
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

		err := rows.Scan(&Id, &Status, &EnvironmentId, &ImageId)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedDeployment := Deployment{
			Id:            Id,
			Status:        Status,
			EnvironmentId: EnvironmentId,
			ImageId:       ImageId,
		}
		deployments = append(deployments, loadedDeployment)
	}

	return deployments

}

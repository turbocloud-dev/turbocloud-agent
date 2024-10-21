/*
We create Image each time when we POST /deploy/:environment_id
One Environment can have only one Image
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/rqlite/gorqlite"
)

const ImageStatusToBuild = "to_build"
const ImageStatusBuilding = "building"
const ImageStatusReady = "ready"
const ImageStatusError = "error"

type Image struct {
	Id           string
	Status       string
	DeploymentId string
	ErrorMsg     string
}

func addImage(image Image) Image {

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
		return image
	}

	image.Id = strings.ToLower(id)

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO Image( Id, Status, DeploymentId, ErrorMsg) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{image.Id, image.Status, image.DeploymentId, image.ErrorMsg},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to Deployment table: %s\n", err.Error())
	}
	return image
}

func updateImageStatus(image Image, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Image SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, image.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return err
	}

	return nil
}

func getImageByDeploymentIdAndStatus(deploymentId string, status string) []Image {

	var images = []Image{}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, DeploymentId, ErrorMsg from Image WHERE STATUS = ? AND DeploymentId = ?",
			Arguments: []interface{}{status, deploymentId},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from Image table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var DeploymentId string
		var ErrorMsg string

		err := rows.Scan(&Id, &Status, &DeploymentId, &ErrorMsg)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedImage := Image{
			Id:           Id,
			Status:       Status,
			DeploymentId: DeploymentId,
			ErrorMsg:     ErrorMsg,
		}
		images = append(images, loadedImage)
	}

	return images

}

func buildImage(image Image, deployment Deployment) {

	err := updateImageStatus(image, ImageStatusBuilding)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE Deployment SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{DeploymentStatusBuildingImage, deployment.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot update a row in Deployment: %s\n", err.Error())
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

	fmt.Println("Image " + image.Id + "has been built. Update status to " + ImageStatusReady)

	err = updateImageStatus(image, ImageStatusReady)

	if err != nil {
		fmt.Printf(" Cannot update a row in Image: %s\n", err.Error())
		return
	}
}
